// Package secrets provides HTTP handlers for the /api/secrets routes.
package secrets

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	secretssvc "github.com/ubunatic/paperclip-go/internal/secrets"
	"github.com/ubunatic/paperclip-go/internal/domain"
	"github.com/ubunatic/paperclip-go/internal/respond"
)

// Handler returns an http.Handler for the /api/secrets sub-router.
func Handler(svc *secretssvc.Service) http.Handler {
	r := chi.NewRouter()
	r.Get("/", list(svc))
	r.Post("/", create(svc))
	r.Get("/{id}", get(svc))
	r.Patch("/{id}", update(svc))
	r.Delete("/{id}", delete(svc))
	return r
}

func list(svc *secretssvc.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		companyID := r.URL.Query().Get("companyId")
		if strings.TrimSpace(companyID) == "" {
			respond.Error(w, http.StatusUnprocessableEntity, "validation_error", "companyId query parameter is required and must not be blank")
			return
		}

		items, err := svc.ListByCompany(r.Context(), companyID)
		if err != nil {
			log.Printf("secrets: error listing by company: %v", err)
			respond.Error(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
			return
		}

		// Ensure items is never nil (return empty array, not null)
		if items == nil {
			items = make([]*domain.SecretSummary, 0)
		}
		respond.JSON(w, http.StatusOK, map[string]any{"items": items})
	}
}

func create(svc *secretssvc.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			CompanyID string `json:"companyId"`
			Name      string `json:"name"`
			Value     string `json:"value"`
		}
		if !respond.DecodeJSON(w, r, &body) {
			return
		}
		if strings.TrimSpace(body.CompanyID) == "" || strings.TrimSpace(body.Name) == "" || strings.TrimSpace(body.Value) == "" {
			respond.Error(w, http.StatusUnprocessableEntity, "validation_error", "companyId, name, and value are required and must be non-empty")
			return
		}

		secret, err := svc.Create(r.Context(), body.CompanyID, body.Name, body.Value)
		if err != nil {
			if errors.Is(err, secretssvc.ErrDuplicate) {
				respond.Error(w, http.StatusConflict, "duplicate_secret", "secret name already exists in this company")
				return
			}
			log.Printf("secrets: error creating: %v", err)
			respond.Error(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
			return
		}
		respond.JSON(w, http.StatusCreated, secret)
	}
}

func get(svc *secretssvc.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		secret, err := svc.GetByID(r.Context(), id)
		if err != nil {
			if errors.Is(err, secretssvc.ErrNotFound) {
				respond.Error(w, http.StatusNotFound, "not_found", "secret not found")
				return
			}
			log.Printf("secrets: error getting by ID: %v", err)
			respond.Error(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
			return
		}
		respond.JSON(w, http.StatusOK, secret)
	}
}

func update(svc *secretssvc.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		var body struct {
			Name  *string `json:"name"`
			Value *string `json:"value"`
		}
		if !respond.DecodeJSON(w, r, &body) {
			return
		}
		// At least one field must be provided
		if body.Name == nil && body.Value == nil {
			respond.Error(w, http.StatusUnprocessableEntity, "validation_error", "at least one of name or value is required")
			return
		}
		// Validate name if provided
		if body.Name != nil && strings.TrimSpace(*body.Name) == "" {
			respond.Error(w, http.StatusUnprocessableEntity, "validation_error", "name cannot be empty or only whitespace")
			return
		}
		// Validate value if provided
		if body.Value != nil && strings.TrimSpace(*body.Value) == "" {
			respond.Error(w, http.StatusUnprocessableEntity, "validation_error", "value cannot be empty or only whitespace")
			return
		}

		secret, err := svc.Update(r.Context(), id, body.Name, body.Value)
		if err != nil {
			if errors.Is(err, secretssvc.ErrNotFound) {
				respond.Error(w, http.StatusNotFound, "not_found", "secret not found")
				return
			}
			if errors.Is(err, secretssvc.ErrDuplicate) {
				respond.Error(w, http.StatusConflict, "duplicate_secret", "secret name already exists in this company")
				return
			}
			log.Printf("secrets: error updating: %v", err)
			respond.Error(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
			return
		}
		respond.JSON(w, http.StatusOK, secret)
	}
}

func delete(svc *secretssvc.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		err := svc.Delete(r.Context(), id)
		if err != nil {
			if errors.Is(err, secretssvc.ErrNotFound) {
				respond.Error(w, http.StatusNotFound, "not_found", "secret not found")
				return
			}
			log.Printf("secrets: error deleting: %v", err)
			respond.Error(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
