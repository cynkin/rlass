package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/cynkin/rlaas/adminapi"
	"github.com/cynkin/rlaas/grpcserver"
	pb "github.com/cynkin/rlaas/proto"
	"github.com/cynkin/rlaas/store"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func main() {
	// if its run with 2 args, start the http server
	if len(os.Args) >= 3 {
		startServer(os.Args[1], os.Args[2]) // port and redis address
		return
	}
	ctx := context.Background()
	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
	dbURL := getEnv("DATABASE_URL", "postgresql://rlaas:rlaas@localhost:5432/rlaas")

	// Create client (like saving the phone no but not dialing it yet)
	redisClient := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	// Ping Redis with retries to handle K8s DNS resolution delay
	var redisErr error
	for i := 1; i <= 10; i++ {
		redisErr = redisClient.Ping(ctx).Err()
		if redisErr == nil {
			break
		}
		fmt.Printf("Redis connection attempt %d/10 failed: %v\n", i, redisErr)
		if i < 10 {
			time.Sleep(2 * time.Second)
		}
	}
	if redisErr != nil {
		fmt.Printf("✗ Redis connection failed after 10 attempts: %v\n", redisErr)
		return
	}
	fmt.Println("✓ Redis connected")

	db, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		fmt.Printf("PostgreSQL connection failed: %v\n", err)
		return
	}
	defer db.Close() // close connection when main exits

	if err := db.Ping(ctx); err != nil {
		fmt.Printf("PostgreSQL ping failed: %v\n", err)
		return
	}
	fmt.Println("✓ PostgreSQL connected")

	if err := runMigrations(ctx, db); err != nil {
		fmt.Printf("Migration failed: %v\n", err)
		return
	}

	ruleStore := store.NewRuleStore(db)
	if err := ruleStore.SeedDefaultRules(ctx); err != nil {
		fmt.Printf("Failed to seed rules: %v\n", err)
		return
	}

	admin := adminapi.NewAdminServer(db, ruleStore)
	go admin.Start("8090")

	// Opens a TCP port (Claiming this port)
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		fmt.Printf("Failed to listen: %v\n", err)
		return
	}

	// Create gRPC server 
	grpcServer := grpc.NewServer()

	// Register our service implementation with the gRPC server
	pb.RegisterRateLimiterServer(grpcServer, grpcserver.NewRateLimiterServer(redisClient, ruleStore, db))

	// Reflection lets tools like grpcurl inspect your service without the proto file
	reflection.Register(grpcServer)

	fmt.Println("✓ gRPC server listening on :50051")
	if err := grpcServer.Serve(lis); err != nil {
		fmt.Printf("Failed to serve: %v\n", err)
	}
}