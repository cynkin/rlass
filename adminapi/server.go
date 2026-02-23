package adminapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/cynkin/rlaas/store"
)

type AdminServer struct {
	db 			*pgxpool.Pool
	ruleStore   *store.RuleStore
}

func NewAdminServer(db *pgxpool.Pool, ruleStore *store.RuleStore) *AdminServer {
	return &AdminServer{
		db: db,
		ruleStore: ruleStore,
	}
}

type CreateRuleRequest struct {
	RuleID 		string 	`json:"rule_id"`
	Algorithm 	string 	`json:"algorithm"`
	Limit 		int 	`json:"limit"`
	WindowSecs	int 	`json:"window_secs"`
}

type UpdateRuleRequest struct {
	Limit 		*int 	`json:"limit"`
	WindowSecs	*int 	`json:"window_secs"`
    Enabled     *bool   `json:"enabled"`
}

func (a *AdminServer) Start(port string) {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /metrics", a.getMetrics)
	mux.HandleFunc("GET /rules", a.listRules)
	mux.HandleFunc("POST /rules", a.createRule)
	mux.HandleFunc("PATCH /rules/{rule_id}", a.updateRule)
	mux.HandleFunc("DELETE /rules/{rule_id}", a.deleteRule)

	fmt.Printf("✓ Admin API listening on port %s\n", port)
	http.ListenAndServe(":"+port, mux)
}

func (a *AdminServer) listRules(w http.ResponseWriter, r *http.Request) {
	rows, err := a.db.Query(r.Context(), `
		SELECT rule_id, COALESCE(client_id, ''), algorithm, "limit", window_secs, enabled, created_at
		FROM rules ORDER BY created_at
	`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type RuleResponse struct {
		RuleID     string    `json:"rule_id"`
		ClientID   string    `json:"client_id,omitempty"`
		Algorithm  string    `json:"algorithm"`
		Limit      int       `json:"limit"`
		WindowSecs int       `json:"window_secs"`
		Enabled    bool      `json:"enabled"`
		CreatedAt  time.Time `json:"created_at"`
	}

	var rules []RuleResponse
	for rows.Next() {
		var rule RuleResponse
		err := rows.Scan(
			&rule.RuleID, 
			&rule.ClientID,
			&rule.Algorithm,
			&rule.Limit, 
			&rule.WindowSecs, 
			&rule.Enabled, 
			&rule.CreatedAt,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		rules = append(rules, rule)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rules)
}

func (a *AdminServer) createRule(w http.ResponseWriter, r *http.Request) {
	var req CreateRuleRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.RuleID == "" || req.Limit == 0 || req.WindowSecs == 0 {
		http.Error(w, "rule_id, limit, and window_secs are required", http.StatusBadRequest)
		return
	}

	if req.Algorithm == "" {
		req.Algorithm = "fixed_window"
	}

	_, err := a.db.Exec(r.Context(), `
		INSERT INTO rules (rule_id, algorithm, "limit", window_secs)
		VALUES ($1, $2, $3, $4)
	`, req.RuleID, req.Algorithm, req.Limit, req.WindowSecs)

	if err != nil {
		http.Error(w, fmt.Sprintf("failed to create rule: %v", err), http.StatusInternalServerError)
		return
	}

	a.ruleStore.InvalidateCache()

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "created", "rule_id": req.RuleID})
}

func (a *AdminServer) updateRule(w http.ResponseWriter, r *http.Request) {
	ruleID := r.PathValue("rule_id")

	var req UpdateRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	_, err := a.db.Exec(r.Context(), `
		UPDATE rules SET
			"limit"     = COALESCE($1, "limit"),
			window_secs = COALESCE($2, window_secs),
			enabled     = COALESCE($3, enabled),
			updated_at  = NOW()
		WHERE rule_id = $4
	`, req.Limit, req.WindowSecs, req.Enabled, ruleID)

	if err != nil {
		http.Error(w, fmt.Sprintf("failed to update rule: %v", err), http.StatusInternalServerError)
		return
	}

	a.ruleStore.InvalidateCache()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "updated", "rule_id": ruleID})
}

func (a *AdminServer) deleteRule(w http.ResponseWriter, r *http.Request) {
	ruleID := r.PathValue("rule_id")

	_, err := a.db.Exec(r.Context(), `
		UPDATE rules SET enabled = false, updated_at = NOW()
		WHERE rule_id = $1
	`, ruleID)

	if err != nil {
		http.Error(w, fmt.Sprintf("failed to disable rule: %v", err), http.StatusInternalServerError)
		return
	}

	a.ruleStore.InvalidateCache()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "disabled", "rule_id": ruleID})
}

func (a *AdminServer) getMetrics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Count total requests in last 60 seconds from request_logs
	var totalAllowed, totalBlocked int
	err := a.db.QueryRow(ctx, `
		SELECT 
			COUNT(*) FILTER (WHERE allowed = true),
			COUNT(*) FILTER (WHERE allowed = false)
		FROM request_logs
		WHERE created_at > NOW() - INTERVAL '60 seconds'
	`).Scan(&totalAllowed, &totalBlocked)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get per-rule breakdown
	rows, err := a.db.Query(ctx, `
		SELECT 
			rule_id,
			COUNT(*) FILTER (WHERE allowed = true) as allowed,
			COUNT(*) FILTER (WHERE allowed = false) as blocked
		FROM request_logs
		WHERE created_at > NOW() - INTERVAL '60 seconds'
		GROUP BY rule_id
		ORDER BY rule_id
	`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type RuleMetric struct {
		RuleID  string `json:"rule_id"`
		Allowed int    `json:"allowed"`
		Blocked int    `json:"blocked"`
	}

	var ruleMetrics []RuleMetric
	for rows.Next() {
		var rm RuleMetric
		rows.Scan(&rm.RuleID, &rm.Allowed, &rm.Blocked)
		ruleMetrics = append(ruleMetrics, rm)
	}

	type MetricsResponse struct {
		TotalAllowed int          `json:"total_allowed"`
		TotalBlocked int          `json:"total_blocked"`
		ByRule       []RuleMetric `json:"by_rule"`
		Timestamp    time.Time    `json:"timestamp"`
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(MetricsResponse{
		TotalAllowed: totalAllowed,
		TotalBlocked: totalBlocked,
		ByRule:       ruleMetrics,
		Timestamp:    time.Now(),
	})
}

// helper for testing
func (a *AdminServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /metrics", a.getMetrics)
	mux.HandleFunc("GET /rules", a.listRules)
	mux.HandleFunc("POST /rules", a.createRule)
	mux.HandleFunc("PATCH /rules/{rule_id}", a.updateRule)
	mux.HandleFunc("DELETE /rules/{rule_id}", a.deleteRule)
	mux.ServeHTTP(w, r)
}

var _ context.Context