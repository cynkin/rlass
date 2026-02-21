package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/cynkin/rlaas/limiter"
)

type Response struct {
	Allowed   bool   `json:"allowed"`
	Remaining int    `json:"remaining"`
	ClientID  string `json:"client_id"`
	Server    string `json:"server"` // so we know which instance responded
}

func startServer(port string, redisAddr string) {
	client := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		fmt.Printf("Redis connection failed: %v\n", err)
		return
	}
	fmt.Printf("Connected to Redis at %s\n", redisAddr)

	hostname, _ := os.Hostname()
	fw := limiter.NewFixedWindowLimiter(client, 10, 60*time.Second)

	http.HandleFunc("/check", func(w http.ResponseWriter, r *http.Request) {
		clientID := r.URL.Query().Get("client_id")
		if clientID == "" {
			clientID = "default"
		}

		allowed, remaining, err := fw.Allow(r.Context(), clientID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if !allowed {
			w.WriteHeader(http.StatusTooManyRequests)
		}
		json.NewEncoder(w).Encode(Response{
			Allowed:   allowed,
			Remaining: remaining,
			ClientID:  clientID,
			Server:    hostname,
		})
	})

	fmt.Printf("Server listening on port %s\n", port)
	http.ListenAndServe(":"+port, nil)
}