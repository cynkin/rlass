package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/cynkin/rlaas/limiter"
)

func runRaceTest() {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	ctx := context.Background()
	client.FlushAll(ctx)

	// Limit: 10 requests per 60 seconds
	fw := limiter.NewFixedWindowLimiter(client, 10, 60*time.Second)

	totalAllowed := 0
	var mu sync.Mutex
	var wg sync.WaitGroup

	concurrentRequests := 100

	fmt.Printf("Sending %d concurrent requests with limit of 10...\n", concurrentRequests)

	for i := 0; i < concurrentRequests; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			allowed, _, err := fw.Allow(ctx, "race-client")
			if err != nil {
				return
			}
			if allowed {
				mu.Lock()
				totalAllowed++
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()

	fmt.Printf("Limit: 10\n")
	fmt.Printf("Allowed: %d\n", totalAllowed)
	if totalAllowed > 10 {
		fmt.Printf("🚨 RACE CONDITION DETECTED — %d extra requests slipped through\n", totalAllowed-10)
	} else {
		fmt.Printf("✓ No race condition detected this run (try again)\n")
	}
}