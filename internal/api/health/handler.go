// Package health provides the health check endpoint.
package health

import (
	"net/http"

	"github.com/ubunatic/paperclip-go/internal/respond"
)

// Handler handles GET /api/health requests.
func Handler(w http.ResponseWriter, r *http.Request) {
	respond.JSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"version": "dev",
	})
}
