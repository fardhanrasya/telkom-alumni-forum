package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func CheckAndSetRateLimit(ctx context.Context, rdb *redis.Client, userID uuid.UUID, action string, limit time.Duration) (bool, error) {
	if rdb == nil {
		return true, nil
	}

	key := fmt.Sprintf("rate_limit:user:%s:%s", userID.String(), action)

	wasSet, err := rdb.SetNX(ctx, key, "locked", limit).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check rate limit in redis: %w", err)
	}

	return wasSet, nil
}

func GetRateLimitTTL(ctx context.Context, rdb *redis.Client, userID uuid.UUID, action string) (time.Duration, error) {
	if rdb == nil {
		return 0, nil
	}
	key := fmt.Sprintf("rate_limit:user:%s:%s", userID.String(), action)
	return rdb.TTL(ctx, key).Result()
}

func ClearRateLimit(ctx context.Context, rdb *redis.Client, userID uuid.UUID, action string) error {
	if rdb == nil {
		return nil
	}
	key := fmt.Sprintf("rate_limit:user:%s:%s", userID.String(), action)
	_, err := rdb.Del(ctx, key).Result()
	return err
}
