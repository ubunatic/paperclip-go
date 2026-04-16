// Package skills provides HTTP handlers for the /api/skills routes.
package skills

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/ubunatic/paperclip-go/internal/domain"
	"github.com/ubunatic/paperclip-go/internal/respond"
)

// Handler returns an http.Handler for the /api/skills sub-router.
func Handler(skills []domain.Skill) http.Handler {
	r := chi.NewRouter()
	r.Get("/", list(skills))
	return r
}

func list(skills []domain.Skill) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		respond.JSON(w, http.StatusOK, map[string]any{"items": skills})
	}
}
