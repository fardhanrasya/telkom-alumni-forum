package middleware

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type userLimit struct {
	count     int
	resetTime time.Time
}

type RateLimiter struct {
	redisClient *redis.Client
	memoryStore map[string]*userLimit
	mu          sync.Mutex
}

func NewRateLimiter(redisClient *redis.Client) *RateLimiter {
	limiter := &RateLimiter{
		redisClient: redisClient,
		memoryStore: make(map[string]*userLimit),
	}

	// In-memory cleanup goroutine
	go func() {
		ticker := time.NewTicker(2 * time.Minute)
		for range ticker.C {
			limiter.mu.Lock()
			now := time.Now()
			for k, v := range limiter.memoryStore {
				if now.After(v.resetTime) {
					delete(limiter.memoryStore, k)
				}
			}
			limiter.mu.Unlock()
		}
	}()

	return limiter
}

// RateLimitMiddleware limits requests per user_id or IP to maxRequests within windowDuration
func (r *RateLimiter) Limit(actionName string, maxRequests int, windowDuration time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		keyIdentifier := c.ClientIP()
		if userID, exists := c.Get("user_id"); exists {
			if idStr, ok := userID.(string); ok && idStr != "" {
				keyIdentifier = idStr
			}
		}

		key := fmt.Sprintf("ratelimit:%s:%s", actionName, keyIdentifier)
		ctx := c.Request.Context()

		if r.redisClient != nil {
			allowed, err := r.checkRedisLimit(ctx, key, maxRequests, windowDuration)
			if err == nil && !allowed {
				c.JSON(http.StatusTooManyRequests, gin.H{
					"error": "Terlalu banyak permintaan. Silakan tunggu sebentar sebelum mencoba lagi.",
				})
				c.Abort()
				return
			}
		} else {
			if !r.checkMemoryLimit(key, maxRequests, windowDuration) {
				c.JSON(http.StatusTooManyRequests, gin.H{
					"error": "Terlalu banyak permintaan. Silakan tunggu sebentar sebelum mencoba lagi.",
				})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

func (r *RateLimiter) checkRedisLimit(ctx context.Context, key string, max int, window time.Duration) (bool, error) {
	pipe := r.redisClient.Pipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, window)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return true, err // Fallback to allow if Redis has error
	}
	return incr.Val() <= int64(max), nil
}

func (r *RateLimiter) checkMemoryLimit(key string, max int, window time.Duration) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	ul, exists := r.memoryStore[key]

	if !exists || now.After(ul.resetTime) {
		r.memoryStore[key] = &userLimit{
			count:     1,
			resetTime: now.Add(window),
		}
		return true
	}

	if ul.count >= max {
		return false
	}

	ul.count++
	return true
}
