package main

import (
	"context"
	"fmt"
	"time"

	pb "github.com/cynkin/rlaas/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// Connect to the gRPC server
	conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		fmt.Printf("Failed to connect: %v\n", err)
		return
	}
	defer conn.Close()

	client := pb.NewRateLimiterClient(conn)
	ctx := context.Background()

	// Test 1: Default rule — limit 10 per 60s
	fmt.Println("=== Testing default rule (limit: 10/60s) ===")
	for i := 1; i <= 12; i++ {
		resp, err := client.CheckLimit(ctx, &pb.CheckLimitRequest{
			ClientId: "alice",
			RuleId:   "default",
		})
		if err != nil {
			fmt.Printf("Request %d: ERROR - %v\n", i, err)
			continue
		}
		status := "✓ ALLOWED"
		if !resp.Allowed {
			status = "✗ BLOCKED"
		}
		fmt.Printf("Request %d: %s (remaining: %d, algorithm: %s)\n",
			i, status, resp.Remaining, resp.Algorithm)
	}

	fmt.Println()

	// Test 2: Login rule — stricter, limit 5 per 60s
	fmt.Println("=== Testing login rule (limit: 5/60s) ===")
	for i := 1; i <= 7; i++ {
		resp, err := client.CheckLimit(ctx, &pb.CheckLimitRequest{
			ClientId: "alice",
			RuleId:   "login",
		})
		if err != nil {
			fmt.Printf("Request %d: ERROR - %v\n", i, err)
			continue
		}
		status := "✓ ALLOWED"
		if !resp.Allowed {
			status = "✗ BLOCKED"
		}
		retryMsg := ""
		if !resp.Allowed {
			retryMsg = fmt.Sprintf(", retry after: %dms", resp.RetryAfterMs)
		}
		fmt.Printf("Request %d: %s (remaining: %d%s)\n",
			i, status, resp.Remaining, retryMsg)
	}

	fmt.Println()

	// Test 3: Different clients are isolated
	fmt.Println("=== Testing client isolation ===")
	for _, clientID := range []string{"bob", "charlie", "diana"} {
		resp, err := client.CheckLimit(ctx, &pb.CheckLimitRequest{
			ClientId: clientID,
			RuleId:   "login",
		})
		if err != nil {
			fmt.Printf("%s: ERROR\n", clientID)
			continue
		}
		status := "✓ ALLOWED"
		if !resp.Allowed {
			status = "✗ BLOCKED"
		}
		fmt.Printf("%s: %s (remaining: %d)\n", clientID, status, resp.Remaining)
	}

	fmt.Println()

	// Test 4: Show retry_after_ms on blocked request
	fmt.Println("=== Testing retry_after_ms ===")
	// Exhaust alice's search limit
	for i := 0; i < 35; i++ {
		client.CheckLimit(ctx, &pb.CheckLimitRequest{
			ClientId: "alice",
			RuleId:   "search",
		})
	}
	// Now one more — should be blocked with retry info
	resp, _ := client.CheckLimit(ctx, &pb.CheckLimitRequest{
		ClientId: "alice",
		RuleId:   "search",
	})
	fmt.Printf("Blocked response: allowed=%v, retry_after=%dms\n",
		resp.Allowed, resp.RetryAfterMs)

	_ = time.Second // just to use the time import
}