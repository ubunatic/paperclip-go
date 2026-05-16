// Package apikeys provides HTTP handlers for the /api/apikeys routes.
package apikeys

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	apikeyssvc "github.com/ubunatic/paperclip-go/internal/apikeys"
	"github.com/ubunatic/paperclip-go/internal/respond"
)

// Handler returns an http.Handler for the /api/apikeys sub-router.
func Handler(svc *apikeyssvc.Service) http.Handler {
	r := chi.NewRouter()
	r.Post("/", create(svc))
	r.Get("/", list(svc))
	r.Delete("/{id}", revoke(svc))
	return r
}

func create(svc *apikeyssvc.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			CompanyID string `json:"companyId"`
			Name      string `json:"name"`
		}
		if !respond.DecodeJSON(w, r, &body) {
			return
		}
		if strings.TrimSpace(body.CompanyID) == "" || strings.TrimSpace(body.Name) == "" {
			respond.Error(w, http.StatusUnprocessableEntity, "validation_error", "companyId and name are required and must be non-empty")
			return
		}

		key, rawKey, err := svc.Create(r.Context(), body.CompanyID, body.Name)
		if err != nil {
			log.Printf("apikeys: error creating: %v", err)
			respond.Error(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
			return
		}

		respond.JSON(w, http.StatusCreated, map[string]any{
			"id":        key.ID,
			"companyId": key.CompanyID,
			"name":      key.Name,
			"createdAt": key.CreatedAt,
			"key":       rawKey,
		})
	}
}

func list(svc *apikeyssvc.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		companyID := r.URL.Query().Get("companyId")
		if strings.TrimSpace(companyID) == "" {
			respond.Error(w, http.StatusUnprocessableEntity, "validation_error", "companyId query parameter is required and must not be blank")
			return
		}

		items, err := svc.ListByCompany(r.Context(), companyID)
		if err != nil {
			log.Printf("apikeys: error listing by company: %v", err)
			respond.Error(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
			return
		}

		respond.JSON(w, http.StatusOK, map[string]any{"items": items})
	}
}

func revoke(svc *apikeyssvc.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		err := svc.Revoke(r.Context(), id)
		if err != nil {
			if errors.Is(err, apikeyssvc.ErrNotFound) {
				respond.Error(w, http.StatusNotFound, "not_found", "api key not found")
				return
			}
			log.Printf("apikeys: error revoking: %v", err)
			respond.Error(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
