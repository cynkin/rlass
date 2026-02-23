package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// How many CheckLimit calls were made, broken down by rule and result
	RequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rlaas_requests_total",
			Help: "Total number of rate limit checks",
		},
		[]string{"rule_id", "algorithm", "result"}, // result = "allowed" or "blocked"
	)

	// How long each CheckLimit call took — gives us p50/p95/p99
	RequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "rlaas_request_duration_seconds",
			Help:    "Duration of rate limit check in seconds",
			Buckets: prometheus.DefBuckets, // 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10
		},
		[]string{"rule_id", "algorithm"},
	)

	// How long Redis operations take specifically
	RedisDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "rlaas_redis_duration_seconds",
			Help:    "Duration of Redis operations",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation"}, // operation = "fixed_window", "sliding_window", "token_bucket"
	)

	// Current number of active gRPC connections
	ActiveConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "rlaas_active_connections",
			Help: "Number of active gRPC connections",
		},
	)

	// Cache hit vs miss for rule lookups
	RuleCacheHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rlaas_rule_cache_total",
			Help: "Rule cache hits and misses",
		},
		[]string{"result"}, // result = "hit" or "miss"
	)
)