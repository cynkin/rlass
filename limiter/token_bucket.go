package limiter

import (
	"context"
	"fmt"
	"time"
	"math"
	"strconv"

	"github.com/redis/go-redis/v9"
)

type TokenBucketLimiter struct {
	client       *redis.Client
	capacity     float64       // max tokens the bucket can hold
	refillRate   float64       // tokens added per second
	windowSize   time.Duration // TTL for the Redis key
}

func NewTokenBucketLimiter(client *redis.Client, capacity float64, refillRate float64) *TokenBucketLimiter {
	return &TokenBucketLimiter{
		client:     client,
		capacity:   capacity,
		refillRate: refillRate,
		windowSize: time.Hour, // clean up inactive clients after 1 hour
	}
}

func (t *TokenBucketLimiter) Allow(ctx context.Context, clientID string) (bool, float64, error) {
	tokensKey := fmt.Sprintf("rate:bucket:%s:tokens", clientID)
	lastRefillKey := fmt.Sprintf("rate:bucket:%s:last_refill", clientID)

	now := time.Now().UnixMicro()
	pipe := t.client.Pipeline()
	tokensCmd := pipe.Get(ctx, tokensKey)
	lastRefillCmd := pipe.Get(ctx, lastRefillKey)
	pipe.Exec(ctx)

	var currentTokens float64
	tokensStr, err := tokensCmd.Result()

	if err == redis.Nil {
		currentTokens = t.capacity // new client gets a full bucket
	} else if err != nil {
		return false, 0, fmt.Errorf("redis error: %w", err)
	} else {
		currentTokens, _ = strconv.ParseFloat(tokensStr, 64)
	}

	var lastRefill int64
	lastRefillStr, err := lastRefillCmd.Result()
	
	if err == redis.Nil {
		lastRefill = now
	} else if err != nil {
		return false, 0, fmt.Errorf("redis error: %w", err)
	} else {
		lastRefill, _ = strconv.ParseInt(lastRefillStr, 10, 64)
	}

	elapsedSeconds := float64(now-lastRefill) / 1_000_000.0
	tokensToAdd := elapsedSeconds * t.refillRate
	currentTokens = math.Min(t.capacity, currentTokens+tokensToAdd)

	allowed := currentTokens >= 1.0
	if allowed {
		currentTokens -= 1.0
	}

	pipe2 := t.client.Pipeline()
	pipe2.Set(ctx, tokensKey, strconv.FormatFloat(currentTokens, 'f', 6, 64), t.windowSize)
	pipe2.Set(ctx, lastRefillKey, strconv.FormatInt(now, 10), t.windowSize)
	pipe2.Exec(ctx)

	return allowed, currentTokens, nil
}