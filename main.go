package main

import (
	"context"
	"fmt"
	"time"

	"github.com/cynkin/rlaas/limiter"
	"github.com/redis/go-redis/v9"
)

func main() {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	ctx := context.Background()

	if err := client.Ping(ctx).Err(); err != nil {
		fmt.Printf("Redis NOT CONNECTED: %v\n", err)
	}
	fmt.Println("Redis connected")

	// FIXED WINDOW
	fmt.Println("=== Fixed Window (limit: 5 per 10s) ===")
	fw := limiter.NewFixedWindowLimiter(client, 5, time.Second*10)

	for i := 1; i <= 8; i++ {
		allowed, remaining, err := fw.Allow(ctx, "fw-client1")
		if err != nil {
			fmt.Printf("Request %d: ERROR - %v\n", i, err)
			continue
		}

		status := "✓ ALLOWED"
		if !allowed {
			status = "✗ BLOCKED"
		}
		fmt.Printf("Request %d: %s (remaining: %d)\n", i, status, remaining)
	}
	fmt.Println()

	// SLIDING WINDOW
	fmt.Println("=== Sliding Window (limit: 5 per 10s) ===")
	sw := limiter.NewSlidingWindowLimiter(client, 5, 10*time.Second)
	for i := 1; i <= 8; i++ {
		allowed, remaining, err := sw.Allow(ctx, "sw-client1")
		if err != nil {
			fmt.Printf("Request %d: ERROR - %v\n", i, err)
			continue
		}
		status := "✓ ALLOWED"
		if !allowed {
			status = "✗ BLOCKED"
		}
		fmt.Printf("Request %d: %s (remaining: %d)\n", i, status, remaining)
	}

	// TOKEN BUCKET
	fmt.Println("=== Token Bucket (capacity: 5, refill: 1/sec) ===")
	tb := limiter.NewTokenBucketLimiter(client, 5, 1.0)

	fmt.Println("-- Burst: 8 rapid requests --")
	for i := 1; i <= 8; i++ {
		allowed, remaining, err := tb.Allow(ctx, "tb-client1")
		if err != nil {
			fmt.Printf("Request %d: ERROR - %v\n", i, err)
			continue
		}
		status := "✓ ALLOWED"
		if !allowed {
			status = "✗ BLOCKED"
		}
		fmt.Printf("Request %d: %s (tokens remaining: %.2f)\n", i, status, remaining)
	}

	// Wait 3 seconds — bucket should refill 3 tokens
	fmt.Println("\n-- Waiting 3 seconds for token refill --")
	time.Sleep(3 * time.Second)

	fmt.Println("-- 3 requests after refill --")
	for i := 1; i <= 3; i++ {
		allowed, remaining, err := tb.Allow(ctx, "tb-client1")
		if err != nil {
			fmt.Printf("Request %d: ERROR - %v\n", i, err)
			continue
		}
		status := "✓ ALLOWED"
		if !allowed {
			status = "✗ BLOCKED"
		}
		fmt.Printf("Request %d: %s (tokens remaining: %.2f)\n", i, status, remaining)
	}
}
