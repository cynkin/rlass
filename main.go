package main

import (
	"context"
	"fmt"
	"net"
	"os"

	"github.com/redis/go-redis/v9"
	"github.com/cynkin/rlaas/grpcserver"
	pb "github.com/cynkin/rlaas/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	if len(os.Args) >= 3 {
		startServer(os.Args[1], os.Args[2])
		return
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		fmt.Printf("Redis connection failed: %v\n", err)
		return
	}
	fmt.Println("Redis connected")

	// Start gRPC server
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		fmt.Printf("Failed to listen: %v\n", err)
		return
	}

	grpcServer := grpc.NewServer()
	pb.RegisterRateLimiterServer(grpcServer, grpcserver.NewRateLimiterServer(redisClient))

	// Reflection lets tools like grpcurl inspect your service without the proto file
	reflection.Register(grpcServer)

	fmt.Println("gRPC server listening on :50051")
	if err := grpcServer.Serve(lis); err != nil {
		fmt.Printf("Failed to serve: %v\n", err)
	}
}