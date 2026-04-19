// Package health provides the health check endpoint.
package health

import (
	"net/http"

	"github.com/ubunatic/paperclip-go/internal/respond"
)

// Handler returns an HTTP handler for GET /api/health requests with the given version.
func Handler(version string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		respond.JSON(w, http.StatusOK, map[string]any{
			"status":                "ok",
			"version":               version,
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
}
