package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/cynkin/rlaas/limiter"
	"github.com/cynkin/rlaas/store"
)

func main() {
	if len(os.Args) >= 3 {
		startServer(os.Args[1], os.Args[2])
		return
	}

	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	ctx := context.Background()
	client.FlushAll(ctx)

	fmt.Println("=== Original Fixed Window ===")
	fw := limiter.NewFixedWindowLimiter(client, 5, 10*time.Second)
	for i := 1; i <= 8; i++ {
		allowed, remaining, _ := fw.Allow(ctx, "fw-client")
		status := "✓ ALLOWED"
		if !allowed {
			status = "✗ BLOCKED"
		}
		fmt.Printf("Request %d: %s (remaining: %d)\n", i, status, remaining)
	}

	fmt.Println()
	fmt.Println("=== Atomic Fixed Window (Lua) ===")
	afw := store.NewAtomicFixedWindow(client, 5, 10*time.Second)
	for i := 1; i <= 8; i++ {
		allowed, remaining, _ := afw.Allow(ctx, "afw-client")
		status := "✓ ALLOWED"
		if !allowed {
			status = "✗ BLOCKED"
		}
		fmt.Printf("Request %d: %s (remaining: %d)\n", i, status, remaining)
	}
}