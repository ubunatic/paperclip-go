// Package labels provides HTTP handlers for the /api/labels routes.
package labels

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/ubunatic/paperclip-go/internal/domain"
	labelssvc "github.com/ubunatic/paperclip-go/internal/labels"
	"github.com/ubunatic/paperclip-go/internal/respond"
)

// Handler returns an http.Handler for the /api/labels sub-router.
func Handler(svc *labelssvc.Service) http.Handler {
	r := chi.NewRouter()
	r.Get("/", list(svc))
	r.Post("/", create(svc))
	r.Delete("/{id}", delete(svc))
	return r
}

func list(svc *labelssvc.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		companyID := r.URL.Query().Get("companyId")
		if companyID == "" {
			respond.Error(w, http.StatusBadRequest, "validation_error", "companyId is required")
			return
		}

		items, err := svc.ListByCompany(r.Context(), companyID)
		if err != nil {
			log.Printf("labels: error listing: %v", err)
			respond.Error(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
			return
		}

		// Ensure items is never nil, always return an array
		if items == nil {
			items = []*domain.Label{}
		}

		respond.JSON(w, http.StatusOK, map[string]any{"items": items})
	}
}

func create(svc *labelssvc.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MiB
		var body struct {
			CompanyID string `json:"companyId"`
			Name      string `json:"name"`
			Color     string `json:"color"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			respond.Error(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
			return
		}

		if body.CompanyID == "" || body.Name == "" || body.Color == "" {
			respond.Error(w, http.StatusUnprocessableEntity, "validation_error", "companyId, name, and color are required")
			return
		}

		label, err := svc.Create(r.Context(), body.CompanyID, body.Name, body.Color)
		if err != nil {
			if errors.Is(err, labelssvc.ErrDuplicate) {
				respond.Error(w, http.StatusConflict, "duplicate_label", "label name already exists in this company")
				return
			}
			log.Printf("labels: error creating: %v", err)
			respond.Error(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
			return
		}

		respond.JSON(w, http.StatusCreated, label)
	}
}

func delete(svc *labelssvc.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		err := svc.Delete(r.Context(), id)
		if err != nil {
			if errors.Is(err, labelssvc.ErrNotFound) {
				respond.Error(w, http.StatusNotFound, "not_found", "label not found")
				return
			}
			log.Printf("labels: error deleting: %v", err)
			respond.Error(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
