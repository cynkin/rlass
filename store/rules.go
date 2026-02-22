package store

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Rule struct {
	ID         string
	RuleID     string
	ClientID   string // empty means applies to all clients
	Algorithm  string
	Limit      int
	WindowSecs int
	Enabled    bool
}

type RuleStore struct {
	db         *pgxpool.Pool
	cache      map[string]Rule
	cacheMu    sync.RWMutex
	cacheUntil time.Time
	cacheTTL   time.Duration
}

func NewRuleStore(db *pgxpool.Pool) *RuleStore {
	return &RuleStore{
		db:       db,
		cache:    make(map[string]Rule), // start with empty cache map
		cacheTTL: 30 * time.Second, // rules refresh every 30 seconds
	}
}

func (r *RuleStore) GetRule(ctx context.Context, ruleID string) (Rule, error) {
	// Serve from cache if still fresh
	r.cacheMu.RLock()
	if time.Now().Before(r.cacheUntil) {
		rule, ok := r.cache[ruleID]
		r.cacheMu.RUnlock()
		if ok {
			return rule, nil
		}
		// Rule not in cache — fall through to DB
	} else {
		r.cacheMu.RUnlock()
	}

	// Cache is stale — reload all rules from DB
	if err := r.refreshCache(ctx); err != nil {
		return Rule{}, fmt.Errorf("failed to refresh rule cache: %w", err)
	}

	r.cacheMu.RLock()
	defer r.cacheMu.RUnlock()

	rule, ok := r.cache[ruleID]
	if !ok {
		// Fall back to default rule
		rule, ok = r.cache["default"]
		if !ok {
			return Rule{}, fmt.Errorf("no rule found for: %s", ruleID)
		}
	}

	return rule, nil
}

func (r *RuleStore) refreshCache(ctx context.Context) error {
	rows, err := r.db.Query(ctx, `
		SELECT rule_id, COALESCE(client_id, ''), algorithm, "limit", window_secs, enabled
		FROM rules
		WHERE enabled = true
	`)
	if err != nil {
		return fmt.Errorf("query error: %w", err)
	}
	defer rows.Close()

	newCache := make(map[string]Rule)
	for rows.Next() {
		var rule Rule
		err := rows.Scan(
			&rule.RuleID,
			&rule.ClientID,
			&rule.Algorithm,
			&rule.Limit,
			&rule.WindowSecs,
			&rule.Enabled,
		)
		if err != nil {
			return fmt.Errorf("scan error: %w", err)
		}
		newCache[rule.RuleID] = rule
	}

	r.cacheMu.Lock()
	r.cache = newCache
	r.cacheUntil = time.Now().Add(r.cacheTTL)
	r.cacheMu.Unlock()

	fmt.Printf("Rule cache refreshed — %d rules loaded\n", len(newCache))
	return nil
}

func (r *RuleStore) SeedDefaultRules(ctx context.Context) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO rules (rule_id, algorithm, "limit", window_secs)
		VALUES
			('default', 'fixed_window',   10, 60),
			('login',   'fixed_window',    5, 60),
			('search',  'sliding_window', 30, 10),
			('upload',  'fixed_window',    3, 60)
		ON CONFLICT (rule_id) DO NOTHING
	`)
	return err
}

func (r *RuleStore) InvalidateCache() {
	r.cacheMu.Lock()
	r.cacheUntil = time.Time{} // zero time forces refresh on next request
	r.cacheMu.Unlock()
}