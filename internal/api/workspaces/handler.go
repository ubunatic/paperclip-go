// Package workspaces provides HTTP handlers for the /api/execution-workspaces routes.
package workspaces

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/ubunatic/paperclip-go/internal/domain"
	"github.com/ubunatic/paperclip-go/internal/respond"
	"github.com/ubunatic/paperclip-go/internal/workspaces"
)

// Handler returns an http.Handler for the /api/execution-workspaces sub-router.
func Handler(svc *workspaces.Service) http.Handler {
	r := chi.NewRouter()
	r.Get("/", list(svc))
	r.Post("/", create(svc))
	r.Get("/{id}", get(svc))
	r.Delete("/{id}", delete(svc))
	return r
}

func list(svc *workspaces.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		companyID := r.URL.Query().Get("companyId")
		if strings.TrimSpace(companyID) == "" {
			respond.Error(w, http.StatusUnprocessableEntity, "validation_error", "companyId query parameter is required and must not be blank")
			return
		}

		items, err := svc.ListByCompany(r.Context(), companyID)
		if err != nil {
			log.Printf("workspaces: error listing by company: %v", err)
			respond.Error(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
			return
		}

		// Ensure items is never nil (return empty array, not null)
		if items == nil {
			items = make([]*domain.Workspace, 0)
		}
		respond.JSON(w, http.StatusOK, map[string]any{"items": items})
	}
}

func create(svc *workspaces.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MiB
		var body struct {
			CompanyID string  `json:"companyId"`
			AgentID   string  `json:"agentId"`
			IssueID   *string `json:"issueId"`
			Path      string  `json:"path"`
			Status    string  `json:"status"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			respond.Error(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
			return
		}

		if strings.TrimSpace(body.CompanyID) == "" || strings.TrimSpace(body.AgentID) == "" || strings.TrimSpace(body.Path) == "" {
			respond.Error(w, http.StatusUnprocessableEntity, "validation_error", "companyId, agentId, and path are required and must not be blank")
			return
		}

		status := body.Status
		if strings.TrimSpace(status) == "" {
			status = "active"
		}

		if !domain.IsValidWorkspaceStatus(status) {
			respond.Error(w, http.StatusUnprocessableEntity, "validation_error", "status must be one of: active, inactive, error")
			return
		}

		workspace, err := svc.Create(r.Context(), body.CompanyID, body.AgentID, body.Path, body.IssueID, status)
		if err != nil {
			if errors.Is(err, workspaces.ErrDuplicate) {
				respond.Error(w, http.StatusConflict, "duplicate_error", "workspace with this agent and path already exists")
				return
			}
			log.Printf("workspaces: error creating workspace: %v", err)
			respond.Error(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
			return
		}

		respond.JSON(w, http.StatusCreated, workspace)
	}
}

func get(svc *workspaces.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if strings.TrimSpace(id) == "" {
			respond.Error(w, http.StatusUnprocessableEntity, "validation_error", "id is required and must not be blank")
			return
		}

		workspace, err := svc.GetByID(r.Context(), id)
		if err != nil {
			if errors.Is(err, workspaces.ErrNotFound) {
				respond.Error(w, http.StatusNotFound, "not_found", "workspace not found")
				return
			}
			log.Printf("workspaces: error getting workspace: %v", err)
			respond.Error(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
			return
		}

		respond.JSON(w, http.StatusOK, workspace)
	}
}

func delete(svc *workspaces.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if strings.TrimSpace(id) == "" {
			respond.Error(w, http.StatusUnprocessableEntity, "validation_error", "id is required and must not be blank")
			return
		}

		err := svc.Delete(r.Context(), id)
		if err != nil {
			if errors.Is(err, workspaces.ErrNotFound) {
				respond.Error(w, http.StatusNotFound, "not_found", "workspace not found")
				return
			}
			log.Printf("workspaces: error deleting workspace: %v", err)
			respond.Error(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
