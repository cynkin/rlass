package limiter

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type FixedWindowLimiter struct {
	client     *redis.Client
	limit      int
	windowSize time.Duration
}

func NewFixedWindowLimiter(client *redis.Client, limit int, windowSize time.Duration) *FixedWindowLimiter {
	return &FixedWindowLimiter{
		client:     client,
		limit:      limit,
		windowSize: windowSize,
	}
}

func (f *FixedWindowLimiter) Allow(ctx context.Context, clientID string) (bool, int, error) {
	windowStart := time.Now().Truncate(f.windowSize).Unix()
	key := fmt.Sprintf("rate:fixed:%s:%d", clientID, windowStart)

	count, err := f.client.Incr(ctx, key).Result()
	if err != nil {
		return false, 0, err
	}

	if count == 1 {
		f.client.Expire(ctx, key, f.windowSize)
	}

	remaining := f.limit - int(count)
	if remaining < 0 {
		remaining = 0
	}

	if int(count) > f.limit {
		return false, remaining, nil
	}

	return true, remaining, nil
}
