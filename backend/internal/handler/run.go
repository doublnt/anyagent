package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/anyagent/anyagent/backend/internal/db"
	"github.com/anyagent/anyagent/backend/internal/middleware"
	"github.com/anyagent/anyagent/backend/internal/sandbox"
)

// RunRequest is the body of POST /api/v1/agents/{name}/run
type RunRequest struct {
	Input    json.RawMessage `json:"input"`              // buyer's structured input
	EntitlementID string     `json:"entitlement_id"`    // must match token's entitlement_id
}

// RunResponse is the result returned to the MCP gateway.
type RunResponse struct {
	Result      string `json:"result"`
	InputTokens int64  `json:"input_tokens"`
	OutputTokens int64 `json:"output_tokens"`
	CostMicros  int64  `json:"cost_micros"`
	TraceID     string `json:"trace_id"`
}

// RunAgent handles the internal run dispatch for a hosted agent.
// It checks entitlement+quota, runs the sandbox, and records usage.
// The entitlement_id and agent_id come from the JWT token validated by RequireScope("use").
func RunAgent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	agentName := r.PathValue("name")

	// These are set by RequireScope("use") from the entitlement token claims
	entitlementID := middleware.GetEntitlementID(ctx)
	agentIDFromToken := middleware.GetAgentID(ctx)

	if entitlementID == "" || agentIDFromToken == "" {
		http.Error(w, `{"error":"missing entitlement context"}`, http.StatusForbidden)
		return
	}

	// Parse request — entitlement_id in body is optional (for logging); the token claim is authoritative
	var req RunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Look up the agent
	a, err := db.GetAgentByName(ctx, agentName)
	if err != nil {
		http.Error(w, "agent not found", http.StatusNotFound)
		return
	}

	if !a.IsHosted {
		http.Error(w, "agent is not a hosted agent", http.StatusBadRequest)
		return
	}

	// Verify the entitlement belongs to this buyer and covers this agent
	buyerID := middleware.GetUserID(ctx)
	ent, err := db.GetActiveEntitlement(ctx, buyerID, a.ID)
	if err != nil || ent == nil {
		http.Error(w, "no active subscription for this agent", http.StatusForbidden)
		return
	}
	// Atomically check quota and record usage attempt
	// Use zero cost for now; the runner will fill it in after execution
	_, allowed, err := db.IncrementUsage(ctx, ent.ID, 0, 0, 0)
	if err != nil {
		http.Error(w, "failed to check quota", http.StatusInternalServerError)
		return
	}
	if !allowed {
		http.Error(w, `{"error":"quota exceeded","code":"rejected_quota"}`, http.StatusForbidden)
		return
	}

	// Load the agent pack — for MVP, read from filesystem store
	packDir := filepath.Join(AgentStoreDir, agentName)
	promptText := ""
	knowledgeFiles := []string{}

	// Read the system prompt from the pack
	promptPath := filepath.Join(packDir, "prompts", "review.md") // MVP: hardcoded review prompt
	if data, err := os.ReadFile(promptPath); err == nil {
		promptText = string(data)
	} else {
		// Try generic prompt
		promptPath = filepath.Join(packDir, "prompts", "system.md")
		if data, err := os.ReadFile(promptPath); err == nil {
			promptText = string(data)
		}
	}

	// Read knowledge files if present
	knowledgeDir := filepath.Join(packDir, "knowledge")
	if kdir, err := os.Stat(knowledgeDir); err == nil && kdir.IsDir() {
		entries, _ := os.ReadDir(knowledgeDir)
		for _, e := range entries {
			if !e.IsDir() {
				if data, err := os.ReadFile(filepath.Join(knowledgeDir, e.Name())); err == nil {
					knowledgeFiles = append(knowledgeFiles, string(data))
				}
			}
		}
	}

	// Create a trace for this run
	trace, err := db.CreateTrace(ctx, "", buyerID, fmt.Sprintf("hosted call: %s", agentName), &agentName)
	if err != nil {
		trace = nil // non-fatal
	}

	// Run in sandbox
	runner := &sandbox.Runner{
		BaseImage:   getEnv("SANDBOX_IMAGE", "anyagent/agent-runner:latest"),
		EgressProxy: getEnv("EGRESS_PROXY_URL", "http://egress-proxy:8080"),
		ModelAPIKey: os.Getenv("MODEL_API_KEY"),
		ModelURL:    "https://api.anthropic.com/v1/messages",
		ModelName:   "claude-3-5-sonnet-20241022",
	}

	result, usage, runErr := runner.Run(ctx, packDir, promptText, knowledgeFiles, req.Input,
		[]string{"read_file", "list_files", "run_command"})

	status := "ok"
	if runErr != nil {
		status = "error"
		result = fmt.Sprintf("Execution error: %v", runErr)
		usage = &sandbox.UsageSummary{}
	}

	// Record the actual usage
	if trace != nil {
		db.RecordUsageEvent(ctx, ent.ID, a.ID, buyerID,
			usage.InputTokens, usage.OutputTokens, usage.CostMicros, status)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(RunResponse{
		Result:       result,
		InputTokens:  usage.InputTokens,
		OutputTokens: usage.OutputTokens,
		CostMicros:   usage.CostMicros,
		TraceID:      traceID(trace),
	})
}

func traceID(t *db.Trace) string {
	if t == nil {
		return ""
	}
	return t.ID
}
