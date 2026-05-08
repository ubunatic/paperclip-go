// Package labels provides HTTP handlers for the /api/labels routes.
package labels

import (
	"errors"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	labelssvc "github.com/ubunatic/paperclip-go/internal/labels"
	"github.com/ubunatic/paperclip-go/internal/respond"
)

// Handler returns an http.Handler for the /api/labels sub-router.
func Handler(svc *labelssvc.Service) http.Handler {
	r := chi.NewRouter()
	r.Get("/", list(svc))
	r.Post("/", create(svc))
	r.Get("/{id}", get(svc))
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

		respond.JSON(w, http.StatusOK, map[string]any{"items": items})
	}
}

func create(svc *labelssvc.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			CompanyID string `json:"companyId"`
			Name      string `json:"name"`
			Color     string `json:"color"`
		}
		if !respond.DecodeJSON(w, r, &body) {
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

func get(svc *labelssvc.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		label, err := svc.Get(r.Context(), id)
		if err != nil {
			if errors.Is(err, labelssvc.ErrNotFound) {
				respond.Error(w, http.StatusNotFound, "not_found", "label not found")
				return
			}
			log.Printf("labels: error getting label: %v", err)
			respond.Error(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
			return
		}
		respond.JSON(w, http.StatusOK, label)
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
