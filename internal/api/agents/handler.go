// Package agents provides HTTP handlers for the /api/agents routes.
package agents

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	svc "github.com/ubunatic/paperclip-go/internal/agents"
	"github.com/ubunatic/paperclip-go/internal/respond"
)

// Handler returns an http.Handler for the /api/agents sub-router.
func Handler(s *svc.Service) http.Handler {
	r := chi.NewRouter()
	r.Get("/", list(s))
	r.Post("/", create(s))
	r.Get("/me", getMe(s))
	r.Get("/{id}", get(s))
	return r
}

func list(s *svc.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		companyID := r.URL.Query().Get("companyId")

		var items interface{}
		var err error

		if companyID != "" {
			items, err = s.ListByCompany(r.Context(), companyID)
		} else {
			items, err = s.List(r.Context())
		}

		if err != nil {
			log.Printf("agents: error: %v", err)
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
			CompanyID   string  `json:"companyId"`
			Shortname   string  `json:"shortname"`
			DisplayName string  `json:"displayName"`
			Role        string  `json:"role"`
			ReportsTo   *string `json:"reportsTo"`
			Adapter     string  `json:"adapter"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			respond.Error(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
			return
		}
		if body.CompanyID == "" || body.Shortname == "" || body.DisplayName == "" {
			respond.Error(w, http.StatusUnprocessableEntity, "validation_error", "companyId, shortname, and displayName are required")
			return
		}
		if body.Adapter == "" {
			body.Adapter = "stub"
		}
		agent, err := s.Create(r.Context(), body.CompanyID, body.Shortname, body.DisplayName, body.Role, body.ReportsTo, body.Adapter)
		if err != nil {
			log.Printf("agents: error: %v", err)
			respond.Error(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
			return
		}
		respond.JSON(w, http.StatusCreated, agent)
	}
}

func get(s *svc.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		agent, err := s.Get(r.Context(), id)
		if err != nil {
			if errors.Is(err, svc.ErrNotFound) {
				respond.Error(w, http.StatusNotFound, "not_found", "agent not found")
				return
			}
			log.Printf("agents: error: %v", err)
			respond.Error(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
			return
		}
		respond.JSON(w, http.StatusOK, agent)
	}
}

func getMe(s *svc.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		agentID := r.Header.Get("X-Agent-Id")
		if agentID == "" {
			respond.Error(w, http.StatusBadRequest, "bad_request", "X-Agent-Id header is required")
			return
		}
		agent, err := s.Get(r.Context(), agentID)
		if err != nil {
			if errors.Is(err, svc.ErrNotFound) {
				respond.Error(w, http.StatusNotFound, "not_found", "agent not found")
				return
			}
			log.Printf("agents: error: %v", err)
			respond.Error(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
			return
		}
		respond.JSON(w, http.StatusOK, agent)
	}
}
