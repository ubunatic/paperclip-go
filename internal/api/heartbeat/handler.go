// Package heartbeat provides HTTP handlers for the /api/heartbeat routes.
package heartbeat

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/ubunatic/paperclip-go/internal/agents"
	svc "github.com/ubunatic/paperclip-go/internal/heartbeat"
	"github.com/ubunatic/paperclip-go/internal/respond"
)

// Handler returns an http.Handler for the /api/heartbeat sub-router.
func Handler(r *svc.Runner) http.Handler {
	router := chi.NewRouter()
	router.Post("/runs", create(r))
	router.Get("/runs", list(r))
	router.Get("/runs/{id}", get(r))
	router.Post("/runs/{id}/cancel", cancel(r))
	return router
}

func create(r *svc.Runner) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		req.Body = http.MaxBytesReader(w, req.Body, 1<<20) // 1 MiB
		var body struct {
			AgentID string `json:"agentId"`
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			respond.Error(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
			return
		}
		if body.AgentID == "" {
			respond.Error(w, http.StatusUnprocessableEntity, "validation_error", "agentId is required")
			return
		}

		run, err := r.Run(req.Context(), body.AgentID)
		if err != nil {
			// Check if it's an agent not found error (errors.Is works with wrapped errors)
			if errors.Is(err, agents.ErrNotFound) {
				respond.Error(w, http.StatusNotFound, "not_found", "agent not found")
				return
			}
			log.Printf("heartbeat: error running: %v", err)
			respond.Error(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
			return
		}
		respond.JSON(w, http.StatusCreated, run)
	}
}

func list(r *svc.Runner) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		agentID := req.URL.Query().Get("agentId")
		if agentID == "" {
			respond.Error(w, http.StatusBadRequest, "bad_request", "agentId query parameter is required")
			return
		}

		items, err := r.ListByAgent(req.Context(), agentID)
		if err != nil {
			log.Printf("heartbeat: error listing: %v", err)
			respond.Error(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
			return
		}

		respond.JSON(w, http.StatusOK, map[string]any{"items": items})
	}
}

func get(r *svc.Runner) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		id := chi.URLParam(req, "id")
		if id == "" {
			respond.Error(w, http.StatusBadRequest, "bad_request", "run id is required")
			return
		}

		run, err := r.GetByID(req.Context(), id)
		if err != nil {
			if errors.Is(err, svc.ErrNotFound) {
				respond.Error(w, http.StatusNotFound, "not_found", "heartbeat run not found")
				return
			}
			log.Printf("heartbeat: error getting run: %v", err)
			respond.Error(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
			return
		}

		respond.JSON(w, http.StatusOK, run)
	}
}

func cancel(r *svc.Runner) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		id := chi.URLParam(req, "id")
		if id == "" {
			respond.Error(w, http.StatusBadRequest, "bad_request", "run id is required")
			return
		}

		run, err := r.Cancel(req.Context(), id)
		if err != nil {
			if errors.Is(err, svc.ErrNotFound) {
				respond.Error(w, http.StatusNotFound, "not_found", "heartbeat run not found")
				return
			}
			if errors.Is(err, svc.ErrTerminalStatus) {
				respond.Error(w, http.StatusConflict, "terminal_status", "heartbeat run is not running")
				return
			}
			log.Printf("heartbeat: error cancelling run: %v", err)
			respond.Error(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
			return
		}

		respond.JSON(w, http.StatusOK, run)
	}
}
