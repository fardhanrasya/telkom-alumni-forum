package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	missionDto "anoa.com/telkomalumiforum/internal/modules/mission/dto"
	missionRepo "anoa.com/telkomalumiforum/internal/modules/mission/repository"
	walletRepo "anoa.com/telkomalumiforum/internal/modules/wallet/repository"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrMissionNotFound = errors.New("misi tidak ditemukan")
	ErrNotClaimable    = errors.New("misi belum siap diklaim")
	ErrAlreadyClaimed  = errors.New("misi ini sudah diklaim")
)

type MissionService interface {
	// RecordProgressAsync records progress toward every active mission that
	// matches actionType, same fire-and-forget shape as
	// leaderboard.AddGamificationPointsAsync — called after the source
	// action (create_thread, like_received, ...) has already succeeded.
	RecordProgressAsync(userID uuid.UUID, actionType string, amount int)
	// RecordLoginStreak updates every active "login_streak" mission for a
	// user's current streak length. Daily missions (target=1, "login hari
	// ini") get a plain increment — already-1-today is a harmless capped
	// no-op. Achievement missions (streak length thresholds) get their
	// progress SET to currentStreak rather than accumulated, since a broken
	// streak must be able to go back down — an additive counter never could.
	RecordLoginStreak(userID uuid.UUID, currentStreak int)
	GetMissions(ctx context.Context, userID uuid.UUID) ([]missionDto.MissionResponse, error)
	Claim(ctx context.Context, userID uuid.UUID, missionID uint) (*missionDto.ClaimResponse, error)
}

type missionService struct {
	repo       missionRepo.MissionRepository
	walletRepo walletRepo.WalletRepository
}

func NewMissionService(repo missionRepo.MissionRepository, walletRepo walletRepo.WalletRepository) MissionService {
	return &missionService{repo: repo, walletRepo: walletRepo}
}

func (s *missionService) RecordProgressAsync(userID uuid.UUID, actionType string, amount int) {
	go func() {
		ctx := context.Background()

		defs, err := s.repo.GetActiveDefinitionsByActionType(ctx, actionType)
		if err != nil {
			log.Printf("Failed to load mission definitions for action %s: %v", actionType, err)
			return
		}

		now := time.Now()
		for _, def := range defs {
			period := PeriodFor(def.Kind, now)
			if err := s.repo.IncrementProgress(ctx, userID, def.ID, period, amount, def.Target); err != nil {
				log.Printf("Failed to record mission progress (user=%s mission=%d period=%s): %v", userID, def.ID, period, err)
			}
		}
	}()
}

func (s *missionService) RecordLoginStreak(userID uuid.UUID, currentStreak int) {
	go func() {
		ctx := context.Background()

		defs, err := s.repo.GetActiveDefinitionsByActionType(ctx, "login_streak")
		if err != nil {
			log.Printf("Failed to load login_streak mission definitions: %v", err)
			return
		}

		now := time.Now()
		for _, def := range defs {
			period := PeriodFor(def.Kind, now)
			var err error
			if def.Kind == "daily" {
				err = s.repo.IncrementProgress(ctx, userID, def.ID, period, 1, def.Target)
			} else {
				err = s.repo.SetProgress(ctx, userID, def.ID, period, currentStreak, def.Target)
			}
			if err != nil {
				log.Printf("Failed to record login streak progress (user=%s mission=%d period=%s): %v", userID, def.ID, period, err)
			}
		}
	}()
}

func (s *missionService) GetMissions(ctx context.Context, userID uuid.UUID) ([]missionDto.MissionResponse, error) {
	defs, err := s.repo.GetActiveDefinitions(ctx)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	periods := map[string]bool{AchievementPeriod: true}
	for _, def := range defs {
		periods[PeriodFor(def.Kind, now)] = true
	}
	periodList := make([]string, 0, len(periods))
	for p := range periods {
		periodList = append(periodList, p)
	}

	progressRows, err := s.repo.GetProgressForUser(ctx, userID, periodList)
	if err != nil {
		return nil, err
	}

	type progressKey struct {
		missionID uint
		period    string
	}
	progressMap := make(map[progressKey]int)
	claimedMap := make(map[progressKey]bool)
	for _, p := range progressRows {
		key := progressKey{missionID: p.MissionID, period: p.Period}
		progressMap[key] = p.Progress
		claimedMap[key] = p.ClaimedAt != nil
	}

	responses := make([]missionDto.MissionResponse, 0, len(defs))
	for _, def := range defs {
		period := PeriodFor(def.Kind, now)
		key := progressKey{missionID: def.ID, period: period}
		progress := progressMap[key]

		status := "in_progress"
		if claimedMap[key] {
			status = "claimed"
		} else if progress >= def.Target {
			status = "claimable"
		}

		responses = append(responses, missionDto.MissionResponse{
			ID:         def.ID,
			Name:       def.Name,
			Kind:       def.Kind,
			ActionType: def.ActionType,
			Target:     def.Target,
			Reward:     def.Reward,
			Progress:   progress,
			Status:     status,
		})
	}

	return responses, nil
}

func (s *missionService) Claim(ctx context.Context, userID uuid.UUID, missionID uint) (*missionDto.ClaimResponse, error) {
	def, err := s.repo.GetDefinitionByID(ctx, missionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrMissionNotFound
		}
		return nil, err
	}
	if !def.IsActive {
		return nil, ErrMissionNotFound
	}

	period := PeriodFor(def.Kind, time.Now())
	var result missionDto.ClaimResponse

	err = s.repo.DB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txMissionRepo := s.repo.WithTx(tx)
		txWalletRepo := s.walletRepo.WithTx(tx)

		progress, err := txMissionRepo.GetProgressForUpdate(tx, userID, missionID, period)
		if err != nil {
			return err
		}
		if progress.ClaimedAt != nil {
			return ErrAlreadyClaimed
		}
		if progress.Progress < def.Target {
			return ErrNotClaimable
		}

		if err := txMissionRepo.MarkClaimed(tx, progress.ID); err != nil {
			return err
		}

		sourceRefID := fmt.Sprintf("%s:%d:%s", userID, missionID, period)
		entry, err := txWalletRepo.ApplyLedgerEntry(tx, userID, def.Reward, "mission_claim", sourceRefID)
		if err != nil {
			return err
		}

		result = missionDto.ClaimResponse{Reward: def.Reward, NewBalance: entry.BalanceAfter}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &result, nil
}
