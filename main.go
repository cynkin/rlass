package main

import (
	"context"
	"fmt"
	"net"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/cynkin/rlaas/adminapi"
	"github.com/cynkin/rlaas/grpcserver"
	pb "github.com/cynkin/rlaas/proto"
	"github.com/cynkin/rlaas/store"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// if its run with 2 args, start the http server
	if len(os.Args) >= 3 {
		startServer(os.Args[1], os.Args[2]) // port and redis address
		return
	}
	ctx := context.Background()

	// Create client (like saving the phone no but not dialing it yet)
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	// Ping Redis to open connection
	if err := redisClient.Ping(ctx).Err(); err != nil {
		fmt.Printf("✓ Redis connection failed: %v\n", err)
		return
	}
	fmt.Println("Redis connected")

	dbURL := "postgresql://rlaas:rlaas@localhost:5432/rlaas"
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

	ruleStore := store.NewRuleStore(db)
	if err := ruleStore.SeedDefaultRules(ctx); err != nil {
		fmt.Printf("Failed to seed rules: %v\n", err)
		return
	}

	// Opens a TCP port (Claiming this port)
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		fmt.Printf("Failed to listen: %v\n", err)
		return
	}

	// Create gRPC server 
	grpcServer := grpc.NewServer()

	// Register our service implementation with the gRPC server
	pb.RegisterRateLimiterServer(grpcServer, grpcserver.NewRateLimiterServer(redisClient, ruleStore))

	// Reflection lets tools like grpcurl inspect your service without the proto file
	reflection.Register(grpcServer)

	admin := adminapi.NewAdminServer(db, ruleStore)
	go admin.Start("8090")

	fmt.Println("✓ gRPC server listening on :50051")
	if err := grpcServer.Serve(lis); err != nil {
		fmt.Printf("Failed to serve: %v\n", err)
	}
}