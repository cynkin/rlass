package grpcserver

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	pb "github.com/cynkin/rlaas/proto"
	"github.com/cynkin/rlaas/store"
)

// Rule defines the rate limiting configuration for a specific use case
type Rule struct {
	ID         string
	Algorithm  string
	Limit      int
	WindowSize time.Duration
}

// DefaultRules — hardcoded for now, will move to PostgreSQL in Phase 4
var DefaultRules = map[string]Rule{
	"default": {
		ID:         "default",
		Algorithm:  "fixed_window",
		Limit:      10,
		WindowSize: 60 * time.Second,
	},
	"login": {
		ID:         "login",
		Algorithm:  "fixed_window",
		Limit:      5,
		WindowSize: 60 * time.Second,
	},
	"search": {
		ID:         "search",
		Algorithm:  "sliding_window",
		Limit:      30,
		WindowSize: 10 * time.Second,
	},
}

type RateLimiterServer struct {
	pb.UnimplementedRateLimiterServer	// embedding for forward compatibility
	redisClient *redis.Client
	ruleStore  *store.RuleStore
}

func NewRateLimiterServer(redisClient *redis.Client, ruleStore *store.RuleStore) *RateLimiterServer {
	return &RateLimiterServer{
		redisClient: redisClient,
		ruleStore:   ruleStore,
	}
}

func (s *RateLimiterServer) CheckLimit(ctx context.Context, req *pb.CheckLimitRequest) (*pb.CheckLimitResponse, error) {
	// Look up the rule
	rule, err := s.ruleStore.GetRule(ctx, req.RuleId)
	if err != nil {
		return nil, fmt.Errorf("rule lookup failed: %w", err)
	}

	// Build a composite key: rule + client so different rules don't interfere
	clientKey := fmt.Sprintf("%s:%s", rule.RuleID, req.ClientId)
	windowSize := time.Duration(rule.WindowSecs) * time.Second

	var allowed bool
	var remaining int
	
	switch rule.Algorithm {
	case "sliding_window":
		sw := store.NewAtomicSlidingWindow(s.redisClient, rule.Limit, windowSize)
		allowed, remaining, err = sw.Allow(ctx, clientKey)
	default: // fixed_window
		fw := store.NewAtomicFixedWindow(s.redisClient, rule.Limit, windowSize)
		allowed, remaining, err = fw.Allow(ctx, clientKey)
	}

	if err != nil {
		return nil, fmt.Errorf("rate limiter error: %w", err)
	}

	retryAfterMs := int64(0)
	if !allowed {
		retryAfterMs = windowSize.Milliseconds()
	}

	return &pb.CheckLimitResponse{
		Allowed:      allowed,
		Remaining:    int32(remaining),
		RetryAfterMs: retryAfterMs,
		Algorithm:    rule.Algorithm,
	}, nil
}