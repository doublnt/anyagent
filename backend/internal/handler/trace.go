package handler

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/anyagent/anyagent/backend/internal/db"
	"github.com/anyagent/anyagent/backend/internal/middleware"
)

func ListTraces(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	if projectID == "" {
		http.Error(w, `{"error":"project id required"}`, http.StatusBadRequest)
		return
	}

	limit := 20
	offset := 0
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			offset = n
		}
	}

	traces, err := db.ListTraces(r.Context(), projectID, limit, offset)
	if err != nil {
		http.Error(w, `{"error":"failed to list traces"}`, http.StatusInternalServerError)
		return
	}

	if traces == nil {
		traces = []db.Trace{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(traces)
}

func CreateTrace(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	if projectID == "" {
		http.Error(w, `{"error":"project id required"}`, http.StatusBadRequest)
		return
	}

	var req struct {
		Task      string  `json:"task"`
		AgentName *string `json:"agent_name,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}

	if req.Task == "" {
		http.Error(w, `{"error":"task is required"}`, http.StatusBadRequest)
		return
	}

	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		userID = "anonymous"
	}

	trace, err := db.CreateTrace(r.Context(), projectID, userID, req.Task, req.AgentName)
	if err != nil {
		http.Error(w, `{"error":"failed to create trace"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(trace)
}

func GetTrace(w http.ResponseWriter, r *http.Request) {
	traceID := r.PathValue("id")
	if traceID == "" {
		http.Error(w, `{"error":"trace id required"}`, http.StatusBadRequest)
		return
	}

	trace, err := db.GetTrace(r.Context(), traceID)
	if err != nil {
		http.Error(w, `{"error":"trace not found"}`, http.StatusNotFound)
		return
	}

	spans, _ := db.GetTraceSpans(r.Context(), traceID)
	if spans == nil {
		spans = []db.TraceSpan{}
	}

	result := map[string]interface{}{
		"trace": trace,
		"spans": spans,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func AddTraceSpan(w http.ResponseWriter, r *http.Request) {
	traceID := r.PathValue("id")
	if traceID == "" {
		http.Error(w, `{"error":"trace id required"}`, http.StatusBadRequest)
		return
	}

	var req struct {
		ToolName   string  `json:"tool_name"`
		SpanID     string  `json:"span_id"`
		Input      *string `json:"input,omitempty"`
		Output     *string `json:"output,omitempty"`
		DurationMs *int    `json:"duration_ms,omitempty"`
		Status     string  `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}

	if req.ToolName == "" {
		http.Error(w, `{"error":"tool_name is required"}`, http.StatusBadRequest)
		return
	}

	if req.SpanID == "" {
		req.SpanID = generateID()
	}

	if req.Status == "" {
		req.Status = "ok"
	}

	span, err := db.AddTraceSpan(r.Context(), traceID, req.SpanID, req.ToolName, req.Status, req.Input, req.Output, req.DurationMs)
	if err != nil {
		http.Error(w, `{"error":"failed to add span"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(span)
}

func CompleteTrace(w http.ResponseWriter, r *http.Request) {
	traceID := r.PathValue("id")
	if traceID == "" {
		http.Error(w, `{"error":"trace id required"}`, http.StatusBadRequest)
		return
	}

	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}

	if req.Status == "" {
		req.Status = "completed"
	}

	if err := db.CompleteTrace(r.Context(), traceID, req.Status); err != nil {
		http.Error(w, `{"error":"failed to complete trace"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"id":     traceID,
		"status": req.Status,
	})
}

func generateID() string {
	return strconv.FormatInt(time.Now().UnixNano()%1000000, 36) + strconv.Itoa(rand.Intn(1000))
}
