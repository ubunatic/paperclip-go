// Package companies provides HTTP handlers for the /api/companies routes.
package companies

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	svc "github.com/ubunatic/paperclip-go/internal/companies"
	"github.com/ubunatic/paperclip-go/internal/respond"
)

// Handler returns an http.Handler for the /api/companies sub-router.
func Handler(s *svc.Service) http.Handler {
	r := chi.NewRouter()
	r.Get("/", list(s))
	r.Post("/", create(s))
	r.Get("/{id}", get(s))
	return r
}

func list(s *svc.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		items, err := s.List(r.Context())
		if err != nil {
			log.Printf("companies: error: %v", err)
			respond.Error(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
			return
		}
		respond.JSON(w, http.StatusOK, map[string]any{"items": items})
	}
}

func create(s *svc.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MiB
		var body struct {
			Name        string `json:"name"`
			Shortname   string `json:"shortname"`
			Description string `json:"description"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			respond.Error(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
			return
		}
		if body.Name == "" || body.Shortname == "" {
			respond.Error(w, http.StatusUnprocessableEntity, "validation_error", "name and shortname are required")
			return
		}
		company, err := s.Create(r.Context(), body.Name, body.Shortname, body.Description)
		if err != nil {
			log.Printf("companies: error: %v", err)
			respond.Error(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
			return
		}
		respond.JSON(w, http.StatusCreated, company)
	}
}

func get(s *svc.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		company, err := s.Get(r.Context(), id)
		if err != nil {
			if errors.Is(err, svc.ErrNotFound) {
				respond.Error(w, http.StatusNotFound, "not_found", "company not found")
				return
			}
			log.Printf("companies: error: %v", err)
			respond.Error(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
			return
		}
		respond.JSON(w, http.StatusOK, company)
	}
}
