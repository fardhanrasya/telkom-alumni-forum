package reaction

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"anoa.com/telkomalumiforum/internal/entity"
	postRepo "anoa.com/telkomalumiforum/internal/modules/post/repository"
	reactionDto "anoa.com/telkomalumiforum/internal/modules/reaction/dto"
	reactionRepo "anoa.com/telkomalumiforum/internal/modules/reaction/repository"
	repo "anoa.com/telkomalumiforum/internal/modules/thread/repository"
	"anoa.com/telkomalumiforum/pkg/dto"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	leaderboard "anoa.com/telkomalumiforum/internal/modules/leaderboard/service"
	notifService "anoa.com/telkomalumiforum/internal/modules/notification/service"
)

type ReactionService interface {
	ToggleReaction(ctx context.Context, userID uuid.UUID, req reactionDto.ReactionToggleRequest) error
	GetReactions(ctx context.Context, userID *uuid.UUID, refID uuid.UUID, refType string) (*dto.ReactionsResponse, error)
}

type reactionService struct {
	repo                reactionRepo.ReactionRepository
	redisClient         *redis.Client
	leaderboardService  leaderboard.LeaderboardService
	notificationService notifService.NotificationService
	threadRepo          repo.Repository
	postRepo            postRepo.PostRepository
}

func NewReactionService(repo reactionRepo.ReactionRepository, redisClient *redis.Client, leaderboardService leaderboard.LeaderboardService, notificationService notifService.NotificationService, threadRepo repo.Repository, postRepo postRepo.PostRepository) ReactionService {
	return &reactionService{
		repo:                repo,
		redisClient:         redisClient,
		leaderboardService:  leaderboardService,
		notificationService: notificationService,
		threadRepo:          threadRepo,
		postRepo:            postRepo,
	}
}

func (s *reactionService) ToggleReaction(ctx context.Context, userID uuid.UUID, req reactionDto.ReactionToggleRequest) error {
	reaction := &entity.Reaction{
		UserID:        userID,
		ReferenceID:   req.ReferenceID,
		ReferenceType: req.ReferenceType,
		Emoji:         req.Emoji,
	}

	// 1. DB Toggle (Get old and new state)
	oldEmoji, newEmoji, err := s.repo.ToggleReaction(ctx, reaction)
	if err != nil {
		return err
	}

	// 2. Redis Update (Pipeline for atomicity-like behavior)
	redisKey := fmt.Sprintf("counts:%s:%s", req.ReferenceType, req.ReferenceID.String())
	pipe := s.redisClient.Pipeline()

	if oldEmoji != "" {
		// Decrement old emoji count
		pipe.HIncrBy(ctx, redisKey, oldEmoji, -1)
	}

	if newEmoji != "" {
		// Increment new emoji count
		pipe.HIncrBy(ctx, redisKey, newEmoji, 1)
	}

	// Execute Redis updates
	if _, err := pipe.Exec(ctx); err != nil {
		// We just log error, data already consistent in DB
		fmt.Printf("Redis reaction update failed: %v\n", err)
	}

	// 3. Side Effects (Gamification & Notification)
	// Only if a reaction is effectively active/added (newEmoji set).
	// We handle idempotency in LeaderboardService.
	// We exclude Menfess for both Gamification and Notification (Anonimity/Privacy).
	if newEmoji != "" && req.ReferenceType != "menfess" {
		go func() {
			// Determine authorID and Slug
			var authorID uuid.UUID
			var refTable string
			var slug string
			var titleSnippet string

			switch req.ReferenceType {
			case "thread":
				thread, err := s.threadRepo.FindByID(context.Background(), req.ReferenceID)
				if err == nil {
					authorID = thread.UserID
					refTable = "threads"
					slug = thread.Slug
					titleSnippet = thread.Title
				}
			case "post":
				post, err := s.postRepo.FindByID(context.Background(), req.ReferenceID)
				if err == nil {
					authorID = post.UserID
					refTable = "posts"
					// Need Thread for Slug
					t, err := s.threadRepo.FindByID(context.Background(), post.ThreadID)
					if err == nil {
						slug = t.Slug
						titleSnippet = t.Title
					}
				}
			}

			// Validate legality
			if authorID == uuid.Nil || authorID == userID {
				return
			}

			// A. Gamification
			if s.leaderboardService != nil {
				s.leaderboardService.AddGamificationPointsAsync(authorID, leaderboard.ActionLikeReceived, req.ReferenceID.String(), refTable, &userID)
			}

			// B. Notification
			if s.notificationService != nil {
				// We don't have actor's name here unless we fetch it, but usually "Someone" or utilizing ActorID metadata in frontend is fine.
				// Format: "Someone reacted with [Emoji] to your [Type]"
				msg := fmt.Sprintf("Someone reacted with %s to your %s", req.Emoji, req.ReferenceType)
				if titleSnippet != "" {
					if len(titleSnippet) > 20 {
						titleSnippet = titleSnippet[:20] + "..."
					}
					msg = fmt.Sprintf("Someone reacted with %s to your %s: %s", req.Emoji, req.ReferenceType, titleSnippet)
				}

				notif := &entity.Notification{
					UserID:     authorID,
					ActorID:    userID,
					EntityID:   req.ReferenceID,
					EntitySlug: slug,
					EntityType: req.ReferenceType, // "thread" or "post" so frontend knows where to route
					Type:       "reaction",        // Specific type for icon/styling
					Message:    msg,
					IsRead:     false,
				}
				_ = s.notificationService.CreateNotification(context.Background(), notif)
			}
		}()
	}

	return nil
}

func (s *reactionService) GetReactions(ctx context.Context, userID *uuid.UUID, refID uuid.UUID, refType string) (*dto.ReactionsResponse, error) {
	// 1. Try Redis for Counts
	redisKey := fmt.Sprintf("counts:%s:%s", refType, refID.String())
	val, err := s.redisClient.HGetAll(ctx, redisKey).Result()

	counts := make(map[string]int64)
	cacheHit := false

	if err == nil && len(val) > 0 {
		cacheHit = true
		for k, v := range val {
			count, _ := strconv.ParseInt(v, 10, 64)
			if count > 0 { // Don't return 0 or negative counts
				counts[k] = count
			}
		}
	}

	// 2. If Cache Miss, Rebuild from DB
	if !cacheHit {
		counts, err = s.repo.GetReactionsCount(ctx, refID, refType)
		if err != nil {
			return nil, err
		}

		// Repopulate Redis (Async or Blocking? Blocking is safer for consistency here)
		// Clean the key first just in case
		pipe := s.redisClient.Pipeline()
		pipe.Del(ctx, redisKey)
		for emoji, count := range counts {
			pipe.HSet(ctx, redisKey, emoji, count)
		}
		// Set generic TTL for cleanup (e.g., 7 days of inactivity)
		pipe.Expire(ctx, redisKey, 7*24*time.Hour)
		_, _ = pipe.Exec(ctx)
	}

	// 3. Get User Status (if logged in)
	var userReacted *string
	if userID != nil {
		reactions, err := s.repo.GetUserReactions(ctx, *userID, refID, refType)
		if err != nil {
			return nil, err
		}
		if len(reactions) > 0 {
			userReacted = &reactions[0]
		}
	}

	// Ensure counts map is not nil for JSON marshalling
	if counts == nil {
		counts = make(map[string]int64)
	}

	return &dto.ReactionsResponse{
		Counts:      counts,
		UserReacted: userReacted,
	}, nil
}
