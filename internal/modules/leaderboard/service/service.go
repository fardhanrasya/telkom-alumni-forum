package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"anoa.com/telkomalumiforum/internal/entity"
	leaderboardDto "anoa.com/telkomalumiforum/internal/modules/leaderboard/dto"
	leaderboardRepo "anoa.com/telkomalumiforum/internal/modules/leaderboard/repository"
	notifService "anoa.com/telkomalumiforum/internal/modules/notification/service"
	userRepo "anoa.com/telkomalumiforum/internal/modules/user/repository"
	"anoa.com/telkomalumiforum/pkg/dto"
	"github.com/google/uuid"
)

const (
	ActionLikeReceived    = "like_received"
	ActionCommentReceived = "comment_received"
	ActionCreateThread    = "create_thread"

	PointsLikeReceived    = 10
	PointsCommentReceived = 5
	PointsCreateThread    = 2

	MaxDailyThreadPoints = 3
)

type LeaderboardService interface {
	// AddGamificationPointsAsync adds gamification points to a user asynchronously
	// actorID is optional (nil for non-like actions like create_thread) - it's the user who performed the action (e.g., the liker)
	AddGamificationPointsAsync(targetUserID uuid.UUID, actionType string, referenceID string, referenceTable string, actorID *uuid.UUID)
	GetLeaderboard(limit int, timeframe string) ([]leaderboardDto.LeaderboardEntry, error)
}

type leaderboardService struct {
	repo                leaderboardRepo.LeaderboardRepository
	userRepo            userRepo.UserRepository
	notificationService notifService.NotificationService
}

// We need UserRepository to check if user is bot (by username)
func NewLeaderboardService(repo leaderboardRepo.LeaderboardRepository, userRepo userRepo.UserRepository, notificationService notifService.NotificationService) LeaderboardService {
	return &leaderboardService{
		repo:                repo,
		userRepo:            userRepo,
		notificationService: notificationService,
	}
}

func (s *leaderboardService) AddGamificationPointsAsync(targetUserID uuid.UUID, actionType string, referenceID string, referenceTable string, actorID *uuid.UUID) {
	// Execute in background
	go func() {
		ctx := context.Background()

		// 1. Check if user is BOT
		user, err := s.userRepo.FindByID(ctx, targetUserID.String())
		if err != nil {
			log.Printf("Failed to find user %s for point calculation: %v", targetUserID, err)
			return
		}
		if user.Username == "Mading_Bot" {
			// Skip points for bot
			return
		}

		// 2. For like actions, check if points were already given from this actor
		// This prevents the like/unlike exploit
		if actionType == ActionLikeReceived && actorID != nil {
			exists, err := s.repo.HasLikePointExists(*actorID, actionType, referenceID)
			if err != nil {
				log.Printf("Error checking like point existence: %v", err)
				return
			}
			if exists {
				// Points already given for this like, skip to prevent exploit
				log.Printf("🚫 Duplicate like point prevented: actor=%s already liked reference=%s", actorID, referenceID)
				return
			}
		}

		// 3. Get current stats to check rank before adding points
		currentStats, _ := s.repo.GetUserStatsByUserID(targetUserID)
		var previousScore int
		if currentStats != nil {
			previousScore = currentStats.TotalScoreAllTime
		}
		previousRank := GetGamificationStatus(previousScore).RankName

		// 4. Calculate points to add
		points := 0
		switch actionType {
		case ActionLikeReceived:
			points = PointsLikeReceived
		case ActionCommentReceived:
			points = PointsCommentReceived
		case ActionCreateThread:
			// Check daily cap
			count, err := s.repo.GetDailyThreadCount(targetUserID, time.Now())
			if err != nil {
				log.Printf("Error getting daily thread count for user %s: %v", targetUserID, err)
				return
			}
			if count >= MaxDailyThreadPoints {
				log.Printf("User %s reached daily thread creation point cap", targetUserID)
				return
			}
			points = PointsCreateThread
		default:
			log.Printf("Unknown action type: %s", actionType)
			return
		}

		// 5. Create Log with ActorID
		logEntry := &entity.PointLog{
			UserID:         targetUserID,
			ActionType:     actionType,
			Points:         points,
			ReferenceID:    referenceID,
			ReferenceTable: referenceTable,
			ActorID:        actorID,
			CreatedAt:      time.Now(),
		}

		if err := s.repo.CreatePointLog(logEntry); err != nil {
			log.Printf("Failed to create point log for user %s: %v", targetUserID, err)
			return
		}

		// 6. Update Stats
		if err := s.repo.UpdateUserStats(targetUserID, points); err != nil {
			log.Printf("Failed to update user stats for user %s: %v", targetUserID, err)
			return
		}

		// 7. Check if rank changed (rank up!)
		newScore := previousScore + points
		newRank := GetGamificationStatus(newScore).RankName

		if newRank != previousRank && s.notificationService != nil {
			// User ranked up! Send notification
			s.sendRankUpNotification(ctx, targetUserID, previousRank, newRank, newScore)
		}
	}()
}

// sendRankUpNotification sends a notification when user ranks up
func (s *leaderboardService) sendRankUpNotification(ctx context.Context, userID uuid.UUID, previousRank, newRank string, newScore int) {
	notification := &entity.Notification{
		UserID:     userID,
		ActorID:    userID, // Self-triggered
		EntityID:   userID, // Reference to self
		EntitySlug: "",     // No slug for rank up
		EntityType: "gamification",
		Type:       "rank_up",
		Message:    fmt.Sprintf("🎉 Selamat! Kamu naik rank dari %s ke %s dengan %d poin!", previousRank, newRank, newScore),
		IsRead:     false,
	}

	if err := s.notificationService.CreateNotification(ctx, notification); err != nil {
		log.Printf("Failed to send rank up notification to user %s: %v", userID, err)
	} else {
		log.Printf("✅ Rank up notification sent to user %s: %s -> %s", userID, previousRank, newRank)
	}
}

func (s *leaderboardService) GetLeaderboard(limit int, timeframe string) ([]leaderboardDto.LeaderboardEntry, error) {
	stats, err := s.repo.GetTopUsers(limit, timeframe)
	if err != nil {
		return nil, err
	}

	// Convert to DTO with gamification status
	entries := make([]leaderboardDto.LeaderboardEntry, 0, len(stats))
	for i, stat := range stats {
		// Calculate gamification status - rank is ALWAYS based on all-time points
		// WeeklyLabel provides activity context
		gamificationStatus := GetGamificationStatusWithWeekly(
			stat.TotalScoreAllTime, // Rank is always based on all-time
			stat.TotalScoreWeekly,  // Weekly points for activity label
		)

		var role string
		if stat.User.Role.ID != 0 {
			role = stat.User.Role.Name
		}

		entries = append(entries, leaderboardDto.LeaderboardEntry{
			Username:  stat.User.Username,
			AvatarURL: stat.User.AvatarURL,
			Role:      role,
			Position:  i + 1, // 1-based position
			GamificationStatus: dto.GamificationStatus{
				RankName:      gamificationStatus.RankName,
				NextRank:      gamificationStatus.NextRank,
				CurrentPoints: gamificationStatus.CurrentPoints,
				TargetPoints:  gamificationStatus.TargetPoints,
				Progress:      gamificationStatus.Progress,
				WeeklyPoints:  gamificationStatus.WeeklyPoints,
				WeeklyLabel:   gamificationStatus.WeeklyLabel,
			},
		})
	}

	return entries, nil
}
