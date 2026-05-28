package handler

import (
	"encoding/json"
	"net/http"
)

// VerifyLicense always returns valid (open source, no license needed)
func VerifyLicense(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"valid": true,
		"plan":  "open-source",
	})
}
