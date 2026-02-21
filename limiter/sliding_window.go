package limiter

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type SlidingWindowLimiter struct {
	client     *redis.Client
	limit      int
	windowSize time.Duration
}

func NewSlidingWindowLimiter(client *redis.Client, limit int, windowSize time.Duration) *SlidingWindowLimiter {
	return &SlidingWindowLimiter{
		client:     client,
		limit:      limit,
		windowSize: windowSize,
	}
}

func (s *SlidingWindowLimiter) Allow(ctx context.Context, clientID string) (bool, int, error) {
	key := fmt.Sprintf("rate:sliding:%s", clientID)
	now := time.Now().UnixMicro()
	windowStart := now - s.windowSize.Microseconds()

	pipe := s.client.TxPipeline() // transaction pipeline (better than normal pipeline)

	pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart))
	countCmd := pipe.ZCard(ctx, key)

	if _, err := pipe.Exec(ctx); err != nil {
		return false, 0, err
	}

	count := int(countCmd.Val())

	if count >= s.limit {
		return false, 0, nil
	}

	pipe2 := s.client.TxPipeline()

	pipe2.ZAdd(ctx, key, redis.Z{
		Score:  float64(now),
		Member: now,
	})

	pipe2.Expire(ctx, key, s.windowSize)

	if _, err := pipe2.Exec(ctx); err != nil {
		return false, 0, err
	}

	return true, s.limit - count - 1, nil
}