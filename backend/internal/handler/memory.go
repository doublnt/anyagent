package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/anyagent/anyagent/backend/internal/db"
	"github.com/anyagent/anyagent/backend/internal/middleware"
)

func ListMemories(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	if projectID == "" {
		http.Error(w, `{"error":"project id required"}`, http.StatusBadRequest)
		return
	}

	limit := 50
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

	memories, err := db.ListMemories(r.Context(), projectID, limit, offset)
	if err != nil {
		http.Error(w, `{"error":"failed to list memories"}`, http.StatusInternalServerError)
		return
	}

	if memories == nil {
		memories = []db.Memory{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(memories)
}

func CreateMemory(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	if projectID == "" {
		http.Error(w, `{"error":"project id required"}`, http.StatusBadRequest)
		return
	}

	var req struct {
		Kind    string `json:"kind"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}

	if req.Kind == "" || req.Content == "" {
		http.Error(w, `{"error":"kind and content are required"}`, http.StatusBadRequest)
		return
	}

	userID := middleware.GetUserID(r.Context())
	source := "user"
	if userID == "" {
		source = "cli"
	}

	memory, err := db.CreateMemory(r.Context(), projectID, req.Kind, req.Content, source)
	if err != nil {
		http.Error(w, `{"error":"failed to create memory"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(memory)
}

func SearchMemories(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	if projectID == "" {
		http.Error(w, `{"error":"project id required"}`, http.StatusBadRequest)
		return
	}

	var req struct {
		Query string `json:"query"`
		Limit int    `json:"limit,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}

	if req.Query == "" {
		http.Error(w, `{"error":"query is required"}`, http.StatusBadRequest)
		return
	}

	if req.Limit <= 0 {
		req.Limit = 5
	}

	memories, err := db.SearchMemories(r.Context(), projectID, req.Query, req.Limit)
	if err != nil {
		http.Error(w, `{"error":"failed to search memories"}`, http.StatusInternalServerError)
		return
	}

	if memories == nil {
		memories = []db.Memory{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(memories)
}
