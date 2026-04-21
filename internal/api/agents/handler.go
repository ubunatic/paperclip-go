// Package agents provides HTTP handlers for the /api/agents routes.
package agents

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	svc "github.com/ubunatic/paperclip-go/internal/agents"
	"github.com/ubunatic/paperclip-go/internal/domain"
	"github.com/ubunatic/paperclip-go/internal/respond"
)

// Handler returns an http.Handler for the /api/agents sub-router.
func Handler(s *svc.Service) http.Handler {
	r := chi.NewRouter()
	r.Get("/", list(s))
	r.Post("/", create(s))
	r.Get("/me", getMe(s))
	r.Get("/{id}", get(s))
	r.Patch("/{id}", update(s))
	r.Post("/{id}/pause", pause(s))
	r.Post("/{id}/resume", resume(s))
	r.Post("/{id}/terminate", terminate(s))
	r.Delete("/{id}", delete(s))
	return r
}

func list(s *svc.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		companyID := r.URL.Query().Get("companyId")

		var items []*domain.Agent
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

func delete(s *svc.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		err := s.Delete(r.Context(), id)
		if err != nil {
			if errors.Is(err, svc.ErrNotFound) {
				respond.Error(w, http.StatusNotFound, "not_found", "agent not found")
				return
			}
			if errors.Is(err, svc.ErrHasActiveCheckout) {
				respond.Error(w, http.StatusConflict, "has_active_checkout", "agent has active checkouts")
				return
			}
			log.Printf("agents: error: %v", err)
			respond.Error(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func update(s *svc.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MiB
		var body struct {
			DisplayName   *string         `json:"displayName"`
			Role          *string         `json:"role"`
			RuntimeState  *string         `json:"runtimeState"`
			Configuration json.RawMessage `json:"configuration"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			respond.Error(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
			return
		}
		// At least one field must be provided
		if body.DisplayName == nil && body.Role == nil && body.RuntimeState == nil && body.Configuration == nil {
			respond.Error(w, http.StatusUnprocessableEntity, "validation_error", "at least one of displayName, role, runtimeState, or configuration is required")
			return
		}

		// Parse configuration if provided
		var configuration map[string]any
		if body.Configuration != nil {
			// Unmarshal configuration
			if err := json.Unmarshal(body.Configuration, &configuration); err != nil {
				respond.Error(w, http.StatusBadRequest, "bad_request", "invalid configuration JSON")
				return
			}
			// Check if it's null (after unmarshal, which properly handles whitespace)
			if configuration == nil {
				respond.Error(w, http.StatusUnprocessableEntity, "validation_error", "configuration cannot be null")
				return
			}
		}

		agent, err := s.Update(r.Context(), id, body.DisplayName, body.Role, body.RuntimeState, configuration)
		if err != nil {
			if errors.Is(err, svc.ErrNotFound) {
				respond.Error(w, http.StatusNotFound, "not_found", "agent not found")
				return
			}
			if errors.Is(err, svc.ErrInvalidRuntimeState) {
				respond.Error(w, http.StatusUnprocessableEntity, "invalid_runtime_state", "invalid runtime state")
				return
			}
			log.Printf("agents: error: %v", err)
			respond.Error(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
			return
		}
		respond.JSON(w, http.StatusOK, agent)
	}
}

func pause(s *svc.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		agent, err := s.Pause(r.Context(), id)
		if err != nil {
			if errors.Is(err, svc.ErrNotFound) {
				respond.Error(w, http.StatusNotFound, "not_found", "agent not found")
				return
			}
			if errors.Is(err, svc.ErrInvalidTransition) {
				respond.Error(w, http.StatusUnprocessableEntity, "invalid_transition", "invalid state transition")
				return
			}
			log.Printf("agents: error: %v", err)
			respond.Error(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
			return
		}
		respond.JSON(w, http.StatusOK, agent)
	}
}

func resume(s *svc.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		agent, err := s.Resume(r.Context(), id)
		if err != nil {
			if errors.Is(err, svc.ErrNotFound) {
				respond.Error(w, http.StatusNotFound, "not_found", "agent not found")
				return
			}
			if errors.Is(err, svc.ErrInvalidTransition) {
				respond.Error(w, http.StatusUnprocessableEntity, "invalid_transition", "invalid state transition")
				return
			}
			log.Printf("agents: error: %v", err)
			respond.Error(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
			return
		}
		respond.JSON(w, http.StatusOK, agent)
	}
}

func terminate(s *svc.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		agent, err := s.Terminate(r.Context(), id)
		if err != nil {
			if errors.Is(err, svc.ErrNotFound) {
				respond.Error(w, http.StatusNotFound, "not_found", "agent not found")
				return
			}
			if errors.Is(err, svc.ErrInvalidTransition) {
				respond.Error(w, http.StatusUnprocessableEntity, "invalid_transition", "invalid state transition")
				return
			}
			log.Printf("agents: error: %v", err)
			respond.Error(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
			return
		}
		respond.JSON(w, http.StatusOK, agent)
	}
}
