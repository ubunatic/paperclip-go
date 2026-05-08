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
		var body struct {
			CompanyID  string          `json:"companyId"`
			ActorType  string          `json:"actorType"`
			ActorID    string          `json:"actorId"`
			Action     string          `json:"action"`
			EntityType string          `json:"entityType"`
			EntityID   string          `json:"entityId"`
			MetaJSON   json.RawMessage `json:"metaJson"`
		}
		if !respond.DecodeJSON(w, r, &body) {
			return
		}

		// Validate required fields
		if body.CompanyID == "" || body.ActorType == "" || body.ActorID == "" || body.Action == "" || body.EntityType == "" || body.EntityID == "" {
			respond.Error(w, http.StatusUnprocessableEntity, "validation_error", "companyId, actorType, actorId, action, entityType, and entityId are required")
			return
		}

		// Validate metaJson is valid JSON if provided
		if len(body.MetaJSON) > 0 && !json.Valid(body.MetaJSON) {
			respond.Error(w, http.StatusUnprocessableEntity, "validation_error", "metaJson must be valid JSON")
			return
		}

		activity, err := s.Record(r.Context(), body.CompanyID, body.ActorType, body.ActorID, body.Action, body.EntityType, body.EntityID, string(body.MetaJSON))
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
