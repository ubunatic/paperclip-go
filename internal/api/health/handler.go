// Package health provides the health check endpoint.
package health

import (
	"encoding/json"
	"net/http"
)

// Handler handles GET /api/health requests.
func Handler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"version": "dev",
	})
}
