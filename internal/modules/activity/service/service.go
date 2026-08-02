package service

import (
	"context"
	"time"

	activityDto "anoa.com/telkomalumiforum/internal/modules/activity/dto"
	activityRepo "anoa.com/telkomalumiforum/internal/modules/activity/repository"
	missionService "anoa.com/telkomalumiforum/internal/modules/mission/service"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ActivityService interface {
	// Heartbeat marks the user active for "today" (WIB) and updates their
	// login streak — extends it if their last active day was yesterday,
	// resets it to 1 if there was a gap, and is a no-op if already called
	// today. Called once per FE session, not per request.
	Heartbeat(ctx context.Context, userID uuid.UUID) (*activityDto.StreakResponse, error)
}

type activityService struct {
	repo           activityRepo.ActivityRepository
	missionService missionService.MissionService
}

func NewActivityService(repo activityRepo.ActivityRepository, missionSvc missionService.MissionService) ActivityService {
	return &activityService{repo: repo, missionService: missionSvc}
}

func (s *activityService) Heartbeat(ctx context.Context, userID uuid.UUID) (*activityDto.StreakResponse, error) {
	now := time.Now()
	today := missionService.PeriodFor("daily", now)
	yesterday := missionService.PeriodFor("daily", now.AddDate(0, 0, -1))

	var currentStreak int
	err := s.repo.DB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		streak, err := s.repo.GetStreakForUpdate(tx, userID)
		if err != nil {
			return err
		}

		if streak.LastActiveDate == today {
			currentStreak = streak.CurrentStreak
			return nil
		}

		if streak.LastActiveDate == yesterday {
			streak.CurrentStreak++
		} else {
			streak.CurrentStreak = 1
		}
		streak.LastActiveDate = today
		currentStreak = streak.CurrentStreak

		return s.repo.SaveStreak(tx, streak)
	})
	if err != nil {
		return nil, err
	}

	if s.missionService != nil {
		s.missionService.RecordLoginStreak(userID, currentStreak)
	}

	return &activityDto.StreakResponse{CurrentStreak: currentStreak}, nil
}
