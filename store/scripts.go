package store

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)
var fixedWindowScript = redis.NewScript(`
	local key = KEYS[1]
	local limit = tonumber(ARGV[1])
	local window = tonumber(ARGV[2])

	-- Atomically increment
	local count = redis.call('INCR', key)

	-- Set expiry only on first request in this window
	if count == 1 then
		redis.call('EXPIRE', key, window)
	end

	-- Return count so Go can decide allowed/blocked
	return count
`)

var slidingWindowScript = redis.NewScript(`
	local key = KEYS[1]
	local now = tonumber(ARGV[1])
	local window_start = tonumber(ARGV[2])
	local limit = tonumber(ARGV[3])
	local ttl = tonumber(ARGV[4])

	-- Remove entries outside the window
	redis.call('ZREMRANGEBYSCORE', key, '0', window_start)

	-- Count entries in current window
	local count = redis.call('ZCARD', key)

	-- Only add if under limit
	if count < limit then
		redis.call('ZADD', key, now, now)
		redis.call('EXPIRE', key, ttl)
		return count + 1
	end

	return count + 1  -- return over-limit count so caller knows to block
`)

var tokenBucketScript = redis.NewScript(`
	local tokens_key = KEYS[1]
	local last_refill_key = KEYS[2]
	local capacity = tonumber(ARGV[1])
	local refill_rate = tonumber(ARGV[2])
	local now = tonumber(ARGV[3])
	local ttl = tonumber(ARGV[4])

	-- Get current state
	local tokens = tonumber(redis.call('GET', tokens_key))
	local last_refill = tonumber(redis.call('GET', last_refill_key))

	-- Default for new clients
	if tokens == nil then tokens = capacity end
	if last_refill == nil then last_refill = now end

	-- Calculate refill
	local elapsed = (now - last_refill) / 1000000.0
	local new_tokens = elapsed * refill_rate
	tokens = math.min(capacity, tokens + new_tokens)

	-- Check and consume
	local allowed = 0
	if tokens >= 1.0 then
		tokens = tokens - 1.0
		allowed = 1
	end

	-- Save state back atomically
	redis.call('SET', tokens_key, tostring(tokens), 'EX', ttl)
	redis.call('SET', last_refill_key, tostring(now), 'EX', ttl)

	return {allowed, tostring(tokens)}
`)

type AtomicLimiter struct {
	client     *redis.Client
	limit      int
	windowSize time.Duration
}
func NewAtomicFixedWindow(client *redis.Client, limit int, windowSize time.Duration) *AtomicLimiter {
	return &AtomicLimiter{
		client: client, 
		limit: limit, 
		windowSize: windowSize,
	}
}

func (a *AtomicLimiter) Allow(ctx context.Context, clientID string) (bool, int, error) {
	windowStart := time.Now().Truncate(a.windowSize).Unix()
	key := fmt.Sprintf("rate:atomic:fixed:%s:%d", clientID, windowStart)

	result, err := fixedWindowScript.Run(
		ctx,
		a.client,
		[]string{key},
		a.limit,
		int(a.windowSize.Seconds()),
	).Int()

	if err != nil {
		return false, 0, fmt.Errorf("lua script error: %w", err)
	}

	remaining := a.limit - result
	if remaining < 0 {
		remaining = 0
	}

	return result <= a.limit, remaining, nil
}