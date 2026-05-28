package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/anyagent/anyagent/backend/internal/db"
	"github.com/anyagent/anyagent/backend/internal/middleware"
)

// SubscribeRequest is the body of POST /api/v1/entitlements
type SubscribeRequest struct {
	AgentID    string `json:"agent_id"`
	PeriodDays int    `json:"period_days"` // subscription period in days
}

// EntitlementResponse is the JSON shape returned to clients.
type EntitlementResponse struct {
	ID          string    `json:"id"`
	AgentID     string    `json:"agent_id"`
	Status      string    `json:"status"`
	PeriodStart time.Time `json:"period_start"`
	PeriodEnd   time.Time `json:"period_end"`
	QuotaCalls  int       `json:"quota_calls"`
	QuotaTokens int       `json:"quota_tokens"`
	UsedCalls   int       `json:"used_calls"`
	UsedTokens  int       `json:"used_tokens"`
}

// Subscribe creates or returns an existing entitlement for the buyer.
func Subscribe(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	buyerID := middleware.GetUserID(ctx)
	if buyerID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var req SubscribeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.AgentID == "" {
		http.Error(w, `{"error":"agent_id is required"}`, http.StatusBadRequest)
		return
	}
	if req.PeriodDays <= 0 {
		req.PeriodDays = 30 // default monthly
	}

	// Check if agent exists and is hosted
	a, err := db.GetAgentByID(ctx, req.AgentID)
	if err != nil {
		http.Error(w, "agent not found", http.StatusNotFound)
		return
	}
	if !a.IsHosted {
		http.Error(w, "agent is not available as a hosted service", http.StatusBadRequest)
		return
	}

	// Check for existing active entitlement
	existing, _ := db.GetActiveEntitlement(ctx, buyerID, req.AgentID)
	if existing != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(EntitlementResponse{
			ID:          existing.ID,
			AgentID:     existing.AgentID,
			Status:      existing.Status,
			PeriodStart: existing.PeriodStart,
			PeriodEnd:   existing.PeriodEnd,
			QuotaCalls:  existing.QuotaCalls,
			QuotaTokens: existing.QuotaTokens,
			UsedCalls:   existing.UsedCalls,
			UsedTokens:  existing.UsedTokens,
		})
		return
	}

	// Create new entitlement with base quotas (MVP: flat 100 calls / 500k tokens/month)
	quotaCalls := 100
	quotaTokens := 500_000
	if req.PeriodDays > 30 {
		quotaCalls = req.PeriodDays * 4       // ~4 calls/day
		quotaTokens = req.PeriodDays * 20_000 // ~20k tokens/day
	}

	ent, err := db.CreateEntitlement(ctx, buyerID, req.AgentID, req.PeriodDays, quotaCalls, quotaTokens)
	if err != nil {
		http.Error(w, "failed to create subscription: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(EntitlementResponse{
		ID:          ent.ID,
		AgentID:     ent.AgentID,
		Status:      ent.Status,
		PeriodStart: ent.PeriodStart,
		PeriodEnd:   ent.PeriodEnd,
		QuotaCalls:  ent.QuotaCalls,
		QuotaTokens: ent.QuotaTokens,
		UsedCalls:   ent.UsedCalls,
		UsedTokens:  ent.UsedTokens,
	})
}

// MintToken issues a short-lived entitlement token for the MCP gateway.
// POST /api/v1/entitlements/{id}/token
func MintToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	buyerID := middleware.GetUserID(ctx)
	entitlementID := r.PathValue("id")

	if buyerID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	// Verify the entitlement belongs to this buyer
	ent, err := db.GetActiveEntitlement(ctx, buyerID, "")
	if err != nil || ent == nil || ent.ID != entitlementID {
		http.Error(w, "entitlement not found", http.StatusNotFound)
		return
	}

	// Get the agent ID for this entitlement
	a, err := db.GetAgentByID(ctx, ent.AgentID)
	if err != nil {
		http.Error(w, "agent not found", http.StatusInternalServerError)
		return
	}

	// Mint a short-lived token scoped to this entitlement
	token, err := middleware.GenerateEntitlementToken(buyerID, entitlementID, a.ID)
	if err != nil {
		http.Error(w, "failed to mint token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"token":         token,
		"entitlement_id": entitlementID,
		"agent_id":      a.ID,
		"expires_in":    "86400", // 24h in seconds
	})
}

// CheckEntitlement is used by the MCP gateway to validate an entitlement token.
// GET /api/v1/entitlements/check?agent_id=&entitlement_id=
func CheckEntitlement(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	buyerID := middleware.GetUserID(ctx)
	agentID := r.URL.Query().Get("agent_id")
	entitlementID := r.URL.Query().Get("entitlement_id")

	if buyerID == "" || agentID == "" {
		http.Error(w, `{"error":"missing parameters"}`, http.StatusBadRequest)
		return
	}

	ent, err := db.GetActiveEntitlement(ctx, buyerID, agentID)
	if err != nil || ent == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"allowed":      false,
			"reason":       "no_active_subscription",
		})
		return
	}

	// If entitlementID is provided, verify it matches
	if entitlementID != "" && ent.ID != entitlementID {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"allowed": false,
			"reason":  "entitlement_id_mismatch",
		})
		return
	}

	quotaAvailable := ent.UsedCalls < ent.QuotaCalls && ent.UsedTokens < ent.QuotaTokens

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"allowed":          quotaAvailable,
		"entitlement_id":   ent.ID,
		"used_calls":       ent.UsedCalls,
		"quota_calls":      ent.QuotaCalls,
		"used_tokens":      ent.UsedTokens,
		"quota_tokens":     ent.QuotaTokens,
		"period_end":       ent.PeriodEnd,
		"reason":           "ok",
	})
}

// RecordUsage handles POST /api/v1/usage from the MCP gateway after a call completes.
func RecordUsage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		EntitlementID string `json:"entitlement_id"`
		AgentID       string `json:"agent_id"`
		BuyerID       string `json:"buyer_id"`
		InputTokens   int    `json:"input_tokens"`
		OutputTokens  int    `json:"output_tokens"`
		CostMicros    int64  `json:"cost_micros"`
		Status        string `json:"status"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.EntitlementID == "" || req.AgentID == "" || req.BuyerID == "" {
		http.Error(w, `{"error":"missing required fields"}`, http.StatusBadRequest)
		return
	}

	if req.Status == "" {
		req.Status = "ok"
	}

	event, err := db.RecordUsageEvent(ctx, req.EntitlementID, req.AgentID, req.BuyerID,
		int64(req.InputTokens), int64(req.OutputTokens), req.CostMicros, req.Status)
	if err != nil {
		http.Error(w, "failed to record usage: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(event)
}

// GetSubscription returns the caller's active entitlements (their "subscription" view).
func GetSubscription(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	buyerID := middleware.GetUserID(ctx)
	if buyerID == "" {
		// Unauthenticated: return empty
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"plan":          "open-source",
			"entitlements": []interface{}{},
		})
		return
	}

	// List all entitlements for this buyer (MVP: query all; later add ListEntitlements DB func)
	// For now, return a stub based on the DB — we'll add a proper ListEntitlements in follow-up
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"plan":          "marketplace",
		"entitlements": []interface{}{},
	})
}
