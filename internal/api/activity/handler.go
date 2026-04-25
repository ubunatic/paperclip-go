// Package activity provides HTTP handlers for the /api/activity routes.
package activity

import (
	"encoding/json"
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
	r.Post("/", create(s))
	return r
}

const maxLimit = 500

func create(s *svc.Log) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MiB
		var body struct {
			CompanyID  string `json:"companyId"`
			ActorKind  string `json:"actorKind"`
			ActorID    string `json:"actorId"`
			Action     string `json:"action"`
			EntityKind string `json:"entityKind"`
			EntityID   string `json:"entityId"`
			MetaJSON   string `json:"metaJson"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			respond.Error(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
			return
		}

		// Validate required fields
		if body.CompanyID == "" || body.ActorKind == "" || body.ActorID == "" || body.Action == "" || body.EntityKind == "" || body.EntityID == "" {
			respond.Error(w, http.StatusUnprocessableEntity, "validation_error", "companyId, actorKind, actorId, action, entityKind, and entityId are required")
			return
		}

		// Validate metaJson is valid JSON if provided
		if body.MetaJSON != "" && !json.Valid([]byte(body.MetaJSON)) {
			respond.Error(w, http.StatusUnprocessableEntity, "validation_error", "metaJson must be valid JSON")
			return
		}

		activity, err := s.Record(r.Context(), body.CompanyID, body.ActorKind, body.ActorID, body.Action, body.EntityKind, body.EntityID, body.MetaJSON)
		if err != nil {
			log.Printf("activity: error: %v", err)
			respond.Error(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
			return
		}

		respond.JSON(w, http.StatusCreated, activity)
	}
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
		if limit > maxLimit {
			limit = maxLimit
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
