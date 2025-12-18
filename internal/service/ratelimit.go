package service

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type RateLimitError struct {
	Message    string
	RetryAfter time.Duration
}

func (e *RateLimitError) Error() string {
	return e.Message
}

func CheckAndSetRateLimit(ctx context.Context, rdb *redis.Client, userID uuid.UUID, action string, limit time.Duration) (bool, error) {
	if rdb == nil || limit <= 0 {
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
	ttl, err := rdb.TTL(ctx, key).Result()
	if err != nil {
		return 0, err
	}

	// If key exists but has no expiration (persistent), it might be a bug or misconfiguration.
	// Clean it up so the user isn't blocked forever.
	if ttl == -1 {
		rdb.Del(ctx, key)
		return 0, nil
	}

	return ttl, nil
}

func ClearRateLimit(ctx context.Context, rdb *redis.Client, userID uuid.UUID, action string) error {
	if rdb == nil {
		return nil
	}
	key := fmt.Sprintf("rate_limit:user:%s:%s", userID.String(), action)
	_, err := rdb.Del(ctx, key).Result()
	return err
}

func GetDurationFromEnv(key string, defaultDuration time.Duration) time.Duration {
	valStr := os.Getenv(key)
	if valStr == "" {
		return defaultDuration
	}

	// Try parsing as integer (seconds)
	if valInt, err := strconv.Atoi(valStr); err == nil {
		return time.Duration(valInt) * time.Second
	}

	// Try parsing as duration string (e.g., "5m", "30s")
	if valDur, err := time.ParseDuration(valStr); err == nil {
		return valDur
	}

	return defaultDuration
}
