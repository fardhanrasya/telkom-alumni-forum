package view

import (
	"context"
	"fmt"
	"time"

	repo "anoa.com/telkomalumiforum/internal/modules/thread/repository"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type ViewService interface {
	IncrementView(ctx context.Context, threadID uuid.UUID, userID uuid.UUID) error
	StartViewSyncWorker(ctx context.Context)
}

type viewService struct {
	redisClient *redis.Client
	threadRepo  repo.Repository
}

func NewViewService(redisClient *redis.Client, threadRepo repo.Repository) ViewService {
	return &viewService{
		redisClient: redisClient,
		threadRepo:  threadRepo,
	}
}

func (s *viewService) IncrementView(ctx context.Context, threadID uuid.UUID, userID uuid.UUID) error {
	// 1. Check if user viewed in last hour
	userViewKey := fmt.Sprintf("thread:user_view:%s:%s", threadID, userID)

	exists, err := s.redisClient.Exists(ctx, userViewKey).Result()
	if err != nil {
		return fmt.Errorf("failed to check user view: %w", err)
	}

	// If user already viewed within last hour, don't increment
	if exists == 1 {
		return nil
	}

	// 2. Increment view count in Redis
	viewKey := fmt.Sprintf("thread:views:%s", threadID)
	_, err = s.redisClient.Incr(ctx, viewKey).Result()
	if err != nil {
		return fmt.Errorf("failed to increment view: %w", err)
	}

	// 3. Add to pending sync set
	pendingKey := "pending:thread_views"
	_, err = s.redisClient.SAdd(ctx, pendingKey, threadID.String()).Result()
	if err != nil {
		return fmt.Errorf("failed to add to pending: %w", err)
	}

	// 4. Mark user as viewed (expires in 1 hour)
	_, err = s.redisClient.SetEx(ctx, userViewKey, "viewed", time.Hour).Result()
	if err != nil {
		return fmt.Errorf("failed to set user view: %w", err)
	}

	return nil
}

func (s *viewService) syncViewsToDB(ctx context.Context) {
	pendingKey := "pending:thread_views"

	// Get all thread IDs that need sync
	threadIDs, err := s.redisClient.SMembers(ctx, pendingKey).Result()
	if err != nil {
		fmt.Printf("Error getting pending thread views: %v\n", err)
		return
	}

	if len(threadIDs) == 0 {
		return
	}

	for _, threadIDStr := range threadIDs {
		threadID, err := uuid.Parse(threadIDStr)
		if err != nil {
			fmt.Printf("Invalid thread ID: %s: %v\n", threadIDStr, err)
			continue
		}

		// Get current view count from Redis
		viewKey := fmt.Sprintf("thread:views:%s", threadID)
		viewCountStr, err := s.redisClient.Get(ctx, viewKey).Result()
		if err != nil && err != redis.Nil {
			fmt.Printf("Error getting view count for thread %s: %v\n", threadID, err)
			continue
		}

		if viewCountStr == "" {
			// No views in Redis, skip
			continue
		}

		var viewCount int
		fmt.Sscanf(viewCountStr, "%d", &viewCount)

		if viewCount > 0 {
			// Update database
			thread, err := s.threadRepo.FindByID(ctx, threadID)
			if err != nil {
				fmt.Printf("Thread not found: %s: %v\n", threadID, err)
				continue
			}

			// Add Redis view count to DB view count
			thread.Views += viewCount

			if err := s.threadRepo.Update(ctx, thread); err != nil {
				fmt.Printf("Failed to update thread views in DB: %v\n", err)
				continue
			}

			// Reset Redis counter
			_, err = s.redisClient.Del(ctx, viewKey).Result()
			if err != nil {
				fmt.Printf("Failed to reset Redis counter: %v\n", err)
			}
		}
	}

	// Clear pending set
	_, err = s.redisClient.Del(ctx, pendingKey).Result()
	if err != nil {
		fmt.Printf("Failed to clear pending set: %v\n", err)
	}

	fmt.Printf("Synced views for %d threads\n", len(threadIDs))
}

func (s *viewService) StartViewSyncWorker(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.syncViewsToDB(ctx)
		case <-ctx.Done():
			return
		}
	}
}
