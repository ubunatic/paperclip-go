// Package skills provides HTTP handlers for the /api/skills routes.
package skills

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/ubunatic/paperclip-go/internal/domain"
	"github.com/ubunatic/paperclip-go/internal/respond"
)

// skillSummary is the response struct for skill list items, excluding the large Body.
type skillSummary struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Path        string `json:"path"`
}

// Handler returns an http.Handler for the /api/skills sub-router.
func Handler(skills []domain.Skill) http.Handler {
	r := chi.NewRouter()
	r.Get("/", list(skills))
	return r
}

func list(skills []domain.Skill) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		summaries := make([]skillSummary, len(skills))
		for i, s := range skills {
			summaries[i] = skillSummary{
				Name:        s.Name,
				Description: s.Description,
				Path:        s.Path,
			}
		}
		respond.JSON(w, http.StatusOK, map[string]any{"items": summaries})
	}
}
