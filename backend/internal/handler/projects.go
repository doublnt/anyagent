package handler

import (
	"encoding/json"
	"net/http"
)

func ListProjects(w http.ResponseWriter, r *http.Request) {
	// TODO: Get user from auth, query database
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode([]map[string]interface{}{})
}

func CreateProject(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name    string `json:"name"`
		RepoURL string `json:"repo_url,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// TODO: Create in database
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"id":   "new-project-id",
		"name": req.Name,
	})
}

func GetProject(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	// TODO: Query database
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"id":   id,
		"name": "My Project",
	})
}
