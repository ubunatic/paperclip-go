// Package labels provides HTTP handlers for the /api/labels routes.
package labels

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	lsvc "github.com/ubunatic/paperclip-go/internal/labels"
	"github.com/ubunatic/paperclip-go/internal/respond"
)

// Handler returns an http.Handler for the /api/labels sub-router.
func Handler(labelSvc *lsvc.Service) http.Handler {
	r := chi.NewRouter()
	r.Get("/", list(labelSvc))
	r.Post("/", create(labelSvc))
	r.Get("/{id}", get(labelSvc))
	r.Delete("/{id}", delete(labelSvc))
	return r
}

func list(s *lsvc.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		companyID := r.URL.Query().Get("companyId")

		if companyID == "" {
			respond.Error(w, http.StatusBadRequest, "validation_error", "companyId is required")
			return
		}

		items, err := s.ListByCompany(r.Context(), companyID)
		if err != nil {
			log.Printf("labels: error: %v", err)
			respond.Error(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
			return
		}

		respond.JSON(w, http.StatusOK, map[string]any{"items": items})
	}
}

func create(s *lsvc.Service) http.HandlerFunc {
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
		label, err := s.Create(r.Context(), body.CompanyID, body.Name, body.Color)
		if err != nil {
			if errors.Is(err, lsvc.ErrDuplicate) {
				respond.Error(w, http.StatusConflict, "duplicate_error", "label with this name already exists for the company")
				return
			}
			log.Printf("labels: error: %v", err)
			respond.Error(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
			return
		}
		respond.JSON(w, http.StatusCreated, label)
	}
}

func get(s *lsvc.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		companyID := r.URL.Query().Get("companyId")
		if companyID == "" {
			respond.Error(w, http.StatusBadRequest, "validation_error", "companyId is required")
			return
		}

		id := chi.URLParam(r, "id")
		label, err := s.Get(r.Context(), id)
		if err != nil {
			if errors.Is(err, lsvc.ErrNotFound) {
				respond.Error(w, http.StatusNotFound, "not_found", "label not found")
				return
			}
			log.Printf("labels: error: %v", err)
			respond.Error(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
			return
		}

		// Verify label belongs to the requested company
		if label.CompanyID != companyID {
			respond.Error(w, http.StatusForbidden, "forbidden", "label does not belong to this company")
			return
		}

		respond.JSON(w, http.StatusOK, label)
	}
}

func delete(s *lsvc.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		companyID := r.URL.Query().Get("companyId")
		if companyID == "" {
			respond.Error(w, http.StatusBadRequest, "validation_error", "companyId is required")
			return
		}

		id := chi.URLParam(r, "id")

		// First fetch the label to verify it belongs to this company
		label, err := s.Get(r.Context(), id)
		if err != nil {
			if errors.Is(err, lsvc.ErrNotFound) {
				respond.Error(w, http.StatusNotFound, "not_found", "label not found")
				return
			}
			log.Printf("labels: error: %v", err)
			respond.Error(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
			return
		}

		// Verify label belongs to the requested company
		if label.CompanyID != companyID {
			respond.Error(w, http.StatusForbidden, "forbidden", "label does not belong to this company")
			return
		}

		err = s.Delete(r.Context(), id)
		if err != nil {
			if errors.Is(err, lsvc.ErrNotFound) {
				respond.Error(w, http.StatusNotFound, "not_found", "label not found")
				return
			}
			log.Printf("labels: error: %v", err)
			respond.Error(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
