package db

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Entitlement struct {
	ID          string    `json:"id"`
	BuyerID     string    `json:"buyer_id"`
	AgentID     string    `json:"agent_id"`
	Status      string    `json:"status"`
	PeriodStart time.Time `json:"period_start"`
	PeriodEnd   time.Time `json:"period_end"`
	QuotaCalls  int       `json:"quota_calls"`
	QuotaTokens int       `json:"quota_tokens"`
	UsedCalls   int       `json:"used_calls"`
	UsedTokens  int       `json:"used_tokens"`
	CreatedAt   time.Time `json:"created_at"`
}

type UsageEvent struct {
	ID             string    `json:"id"`
	EntitlementID  string    `json:"entitlement_id"`
	AgentID        string    `json:"agent_id"`
	BuyerID        string    `json:"buyer_id"`
	TraceID        *string   `json:"trace_id,omitempty"`
	CallSequence   int       `json:"call_sequence"`
	InputTokens    int       `json:"input_tokens"`
	OutputTokens   int       `json:"output_tokens"`
	CostMicros     int64     `json:"cost_micros"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
}

// GetActiveEntitlement returns the buyer's active entitlement for an agent.
// Returns nil if none exists or if the entitlement is expired/suspended.
func GetActiveEntitlement(ctx context.Context, buyerID, agentID string) (*Entitlement, error) {
	var e Entitlement
	err := Pool.QueryRow(ctx,
		`SELECT id::text, buyer_id::text, agent_id::text, status,
		        period_start, period_end, quota_calls, quota_tokens,
		        used_calls, used_tokens, created_at
		 FROM entitlements
		 WHERE buyer_id = $1 AND agent_id = $2 AND status = 'active' AND period_end > NOW()
		 ORDER BY created_at DESC LIMIT 1`,
		buyerID, agentID,
	).Scan(&e.ID, &e.BuyerID, &e.AgentID, &e.Status,
		&e.PeriodStart, &e.PeriodEnd, &e.QuotaCalls, &e.QuotaTokens,
		&e.UsedCalls, &e.UsedTokens, &e.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

// CreateEntitlement creates a new subscription entitlement.
func CreateEntitlement(ctx context.Context, buyerID, agentID string, periodDays, quotaCalls, quotaTokens int) (*Entitlement, error) {
	id := uuid.New().String()
	now := time.Now()
	periodEnd := now.AddDate(0, 0, periodDays)
	var e Entitlement
	err := Pool.QueryRow(ctx,
		`INSERT INTO entitlements (id, buyer_id, agent_id, period_start, period_end, quota_calls, quota_tokens)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)
		 RETURNING id::text, buyer_id::text, agent_id::text, status,
		           period_start, period_end, quota_calls, quota_tokens, used_calls, used_tokens, created_at`,
		id, buyerID, agentID, now, periodEnd, quotaCalls, quotaTokens,
	).Scan(&e.ID, &e.BuyerID, &e.AgentID, &e.Status,
		&e.PeriodStart, &e.PeriodEnd, &e.QuotaCalls, &e.QuotaTokens, &e.UsedCalls, &e.UsedTokens, &e.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

// IncrementUsage records that a call was made against an entitlement and returns
// whether the call is allowed (quota not exhausted).
func IncrementUsage(ctx context.Context, entitlementID string, inputTokens, outputTokens, costMicros int64) (*UsageEvent, bool, error) {
	id := uuid.New().String()

	// Check and increment atomically
	var allowed bool
	var usedCalls int
	var usedTokens int64
	err := Pool.QueryRow(ctx,
		`UPDATE entitlements
		   SET used_calls = used_calls + 1,
		       used_tokens = used_tokens + $2,
		       updated_at = NOW()
		   WHERE id = $1
		     AND used_calls < quota_calls
		     AND used_tokens + $2 <= quota_tokens
		   RETURNING used_calls < quota_calls, used_calls, used_tokens`,
		entitlementID, inputTokens+outputTokens,
	).Scan(&allowed, &usedCalls, &usedTokens)
	if err != nil {
		// Either no rows (quota exceeded) or DB error
		if !allowed { // quota exceeded case — insert rejected event
			insertRejectedUsage(ctx, id, entitlementID, inputTokens, outputTokens, costMicros)
			return nil, false, nil
		}
		return nil, false, err
	}

	status := "ok"
	if !allowed {
		status = "rejected_quota"
	}

	var traceID *string
	seq, err := getNextCallSequence(ctx, entitlementID)
	if err != nil {
		seq = 0
	}

	var e UsageEvent
	err = Pool.QueryRow(ctx,
		`INSERT INTO usage_events (id, entitlement_id, agent_id, buyer_id, call_sequence, input_tokens, output_tokens, cost_micros, status)
		 SELECT $1, $2, agent_id, buyer_id, $3, $4, $5, $6, $7
		 FROM entitlements WHERE id = $2
		 RETURNING id::text, entitlement_id::text, agent_id::text, buyer_id::text, trace_id::text,
		           call_sequence, input_tokens, output_tokens, cost_micros, status, created_at`,
		id, entitlementID, seq, inputTokens, outputTokens, costMicros, status,
	).Scan(&e.ID, &e.EntitlementID, &e.AgentID, &e.BuyerID, &traceID,
		&e.CallSequence, &e.InputTokens, &e.OutputTokens, &e.CostMicros, &e.Status, &e.CreatedAt)
	if err != nil {
		return nil, false, err
	}
	e.TraceID = traceID
	return &e, allowed, nil
}

func insertRejectedUsage(ctx context.Context, id, entitlementID string, inputTokens, outputTokens, costMicros int64) {
	Pool.Exec(ctx,
		`INSERT INTO usage_events (id, entitlement_id, agent_id, buyer_id, call_sequence, input_tokens, output_tokens, cost_micros, status)
		 SELECT $1, $2, agent_id, buyer_id, 0, $3, $4, $5, 'rejected_quota'
		 FROM entitlements WHERE id = $2`,
		id, entitlementID, inputTokens, outputTokens, costMicros,
	)
}

func getNextCallSequence(ctx context.Context, entitlementID string) (int, error) {
	var maxSeq *int
	err := Pool.QueryRow(ctx,
		`SELECT MAX(call_sequence) FROM usage_events WHERE entitlement_id = $1`,
		entitlementID,
	).Scan(&maxSeq)
	if err != nil {
		return 0, err
	}
	if maxSeq == nil {
		return 1, nil
	}
	return *maxSeq + 1, nil
}

// RecordUsageEvent writes a completed usage event (used when the call already succeeded
// and we just need to book the cost).
func RecordUsageEvent(ctx context.Context, entitlementID, agentID, buyerID string,
	inputTokens, outputTokens, costMicros int64, status string) (*UsageEvent, error) {
	id := uuid.New().String()
	seq, _ := getNextCallSequence(ctx, entitlementID)

	var e UsageEvent
	var traceID *string
	err := Pool.QueryRow(ctx,
		`INSERT INTO usage_events (id, entitlement_id, agent_id, buyer_id, call_sequence, input_tokens, output_tokens, cost_micros, status)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		 RETURNING id::text, entitlement_id::text, agent_id::text, buyer_id::text, trace_id::text,
		           call_sequence, input_tokens, output_tokens, cost_micros, status, created_at`,
		id, entitlementID, agentID, buyerID, seq, inputTokens, outputTokens, costMicros, status,
	).Scan(&e.ID, &e.EntitlementID, &e.AgentID, &e.BuyerID, &traceID,
		&e.CallSequence, &e.InputTokens, &e.OutputTokens, &e.CostMicros, &e.Status, &e.CreatedAt)
	if err != nil {
		return nil, err
	}
	e.TraceID = traceID
	return &e, nil
}
