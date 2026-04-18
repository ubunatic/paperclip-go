// Package health provides the health check endpoint.
package health

import (
	"net/http"

	"github.com/ubunatic/paperclip-go/internal/respond"
)

// Handler handles GET /api/health requests.
func Handler(w http.ResponseWriter, r *http.Request) {
	respond.JSON(w, http.StatusOK, map[string]any{
		"status":                "ok",
		"version":               "dev",
		"deploymentMode":        "local_trusted",
		"deploymentExposure":    "private",
		"authReady":             true,
		"bootstrapStatus":       "ready",
		"bootstrapInviteActive": false,
		"features": map[string]any{
			"companyDeletionEnabled": true,
		},
	})
}
