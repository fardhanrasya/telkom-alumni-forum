package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"anoa.com/telkomalumiforum/internal/model"
	"anoa.com/telkomalumiforum/internal/repository"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type MenfessService interface {
	CreateMenfess(ctx context.Context, userID uuid.UUID, content string) error
	GetMenfesses(ctx context.Context, offset, limit int) ([]*model.Menfess, int64, error)
}

type menfessService struct {
	repo        repository.MenfessRepository
	redisClient *redis.Client
}

func NewMenfessService(repo repository.MenfessRepository, redisClient *redis.Client) MenfessService {
	return &menfessService{
		repo:        repo,
		redisClient: redisClient,
	}
}

func (s *menfessService) CreateMenfess(ctx context.Context, userID uuid.UUID, content string) error {
	if s.redisClient == nil {
		return errors.New("redis is required for menfess feature")
	}

	// 1. Get/Generate Daily Salt
	// Rotate every day. Key: menfess:salt:YYYY-MM-DD
	today := time.Now().Format("2006-01-02")
	saltKey := fmt.Sprintf("menfess:salt:%s", today)

	salt, err := s.redisClient.Get(ctx, saltKey).Result()
	if err == redis.Nil {
		// Generate new salt
		salt = uuid.NewString() // Use UUID as random salt
		// Set with 25 hours expiry (just to be safe overlapping days)
		if err := s.redisClient.Set(ctx, saltKey, salt, 25*time.Hour).Err(); err != nil {
			return fmt.Errorf("failed to save salt: %v", err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to get salt: %v", err)
	}

	// 2. The Blind Gatekeeper: Compute Hash
	hashInput := userID.String() + salt
	hashBytes := sha256.Sum256([]byte(hashInput))
	userHash := hex.EncodeToString(hashBytes[:])

	// 3. Check Quota
	quotaKey := fmt.Sprintf("menfess_quota:%s", userHash)
	quotaStr, err := s.redisClient.Get(ctx, quotaKey).Result()
	if err != redis.Nil && err != nil {
		return fmt.Errorf("failed to check quota: %v", err)
	}

	quota := 0
	if quotaStr != "" {
		fmt.Sscanf(quotaStr, "%d", &quota)
	}

	if quota >= 2 { // Limit max 2 posts per day
		return errors.New("menfess quota exceeded (max 2 per day)")
	}

	// 4. Increment Quota
	pipe := s.redisClient.Pipeline()
	pipe.Incr(ctx, quotaKey)
	pipe.Expire(ctx, quotaKey, 24*time.Hour)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("failed to update quota: %v", err)
	}

	// 5. Create Menfess with fuzzy timestamp
	now := time.Now()
	// Round to nearest 5 minutes
	fuzzyTime := now.Truncate(5 * time.Minute)

	menfess := &model.Menfess{
		Content:   content,
		CreatedAt: fuzzyTime,
	}

	return s.repo.Create(ctx, menfess)
}

func (s *menfessService) GetMenfesses(ctx context.Context, offset, limit int) ([]*model.Menfess, int64, error) {
	return s.repo.FindAll(ctx, offset, limit)
}
