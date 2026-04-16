// Package activity provides HTTP handlers for the /api/activity routes.
package activity

import (
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	svc "github.com/ubunatic/paperclip-go/internal/activity"
	"github.com/ubunatic/paperclip-go/internal/respond"
)

// Handler returns an http.Handler for the /api/activity sub-router.
func Handler(s *svc.Log) http.Handler {
	r := chi.NewRouter()
	r.Get("/", list(s))
	return r
}

func list(s *svc.Log) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		companyID := r.URL.Query().Get("companyId")
		if companyID == "" {
			respond.Error(w, http.StatusBadRequest, "bad_request", "companyId query parameter is required")
			return
		}

		limit := 50
		if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
			if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
				limit = l
			}
		}

		items, err := s.List(r.Context(), companyID, limit)
		if err != nil {
			log.Printf("activity: error: %v", err)
			respond.Error(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
			return
		}

		respond.JSON(w, http.StatusOK, map[string]any{"items": items})
	}
}
