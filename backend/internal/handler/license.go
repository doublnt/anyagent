package handler

import (
	"encoding/json"
	"net/http"
)

// GetSubscription returns a simple active status (no paid plans in open source)
func GetSubscription(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"plan":   "open-source",
		"status": "active",
	})
}

// VerifyLicense always returns valid (open source, no license needed)
func VerifyLicense(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"valid": true,
		"plan":  "open-source",
	})
}
