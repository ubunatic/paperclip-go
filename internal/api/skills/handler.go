// Package skills provides the HTTP API for skills.
package skills

import (
	"net/http"

	"github.com/ubunatic/paperclip-go/internal/domain"
	"github.com/ubunatic/paperclip-go/internal/respond"
)

// Handler returns an http.HandlerFunc for the /api/skills endpoint.
// It accepts a slice of loaded skills and returns them as JSON.
// Nil slices are normalized to empty arrays for consistent API response shape.
func Handler(skills []domain.Skill) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if skills == nil {
			skills = []domain.Skill{}
		}
		respond.JSON(w, http.StatusOK, map[string]any{"items": skills})
	}
}
