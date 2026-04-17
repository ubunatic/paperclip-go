// Package skills provides the HTTP API for skills.
package skills

import (
	"net/http"

	"github.com/ubunatic/paperclip-go/internal/domain"
	"github.com/ubunatic/paperclip-go/internal/respond"
)

// Handler returns an http.HandlerFunc for the /api/skills endpoint.
// It accepts a slice of loaded skills and returns them as JSON.
func Handler(skills []domain.Skill) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		respond.JSON(w, http.StatusOK, map[string]any{"items": skills})
	}
}
