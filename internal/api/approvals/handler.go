// Package approvals provides HTTP handlers for the /api/approvals routes.
package approvals

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/ubunatic/paperclip-go/internal/approvals"
	"github.com/ubunatic/paperclip-go/internal/domain"
	"github.com/ubunatic/paperclip-go/internal/respond"
)

// Handler returns an http.Handler for the /api/approvals sub-router.
func Handler(svc *approvals.Service) http.Handler {
	r := chi.NewRouter()
	r.Get("/", list(svc))
	r.Post("/", create(svc))
	r.Get("/{id}", get(svc))
	r.Post("/{id}/approve", approve(svc))
	r.Post("/{id}/reject", reject(svc))
	return r
}

func list(svc *approvals.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		companyID := r.URL.Query().Get("companyId")
		if companyID == "" || len(strings.TrimSpace(companyID)) == 0 {
			respond.Error(w, http.StatusUnprocessableEntity, "validation_error", "companyId query parameter is required and must not be blank")
			return
		}

		items, err := svc.ListByCompany(r.Context(), companyID)
		if err != nil {
			log.Printf("approvals: error listing by company: %v", err)
			respond.Error(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
			return
		}

		// Ensure items is never nil (return empty array, not null)
		if items == nil {
			items = make([]*domain.Approval, 0)
		}
		respond.JSON(w, http.StatusOK, map[string]any{"items": items})
	}
}

func create(svc *approvals.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MiB
		var body struct {
			CompanyID   string  `json:"companyId"`
			AgentID     string  `json:"agentId"`
			IssueID     string  `json:"issueId"`
			Kind        string  `json:"kind"`
			RequestBody *string `json:"requestBody"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			respond.Error(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
			return
		}

		if body.CompanyID == "" || body.AgentID == "" || body.IssueID == "" || body.Kind == "" {
			respond.Error(w, http.StatusUnprocessableEntity, "validation_error", "companyId, agentId, issueId, and kind are required and must be non-empty")
			return
		}

		// Validate fields are not just whitespace
		if len(strings.TrimSpace(body.CompanyID)) == 0 {
			respond.Error(w, http.StatusUnprocessableEntity, "validation_error", "companyId cannot be only whitespace")
			return
		}
		if len(strings.TrimSpace(body.AgentID)) == 0 {
			respond.Error(w, http.StatusUnprocessableEntity, "validation_error", "agentId cannot be only whitespace")
			return
		}
		if len(strings.TrimSpace(body.IssueID)) == 0 {
			respond.Error(w, http.StatusUnprocessableEntity, "validation_error", "issueId cannot be only whitespace")
			return
		}
		if len(strings.TrimSpace(body.Kind)) == 0 {
			respond.Error(w, http.StatusUnprocessableEntity, "validation_error", "kind cannot be only whitespace")
			return
		}

		approval, err := svc.Create(r.Context(), body.CompanyID, body.AgentID, body.IssueID, body.Kind, body.RequestBody)
		if err != nil {
			log.Printf("approvals: error creating approval: %v", err)
			respond.Error(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
			return
		}

		respond.JSON(w, http.StatusCreated, approval)
	}
}

func get(svc *approvals.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if id == "" || len(strings.TrimSpace(id)) == 0 {
			respond.Error(w, http.StatusUnprocessableEntity, "validation_error", "id is required and must not be blank")
			return
		}

		approval, err := svc.GetByID(r.Context(), id)
		if err != nil {
			if err == approvals.ErrNotFound {
				respond.Error(w, http.StatusNotFound, "not_found", "approval not found")
				return
			}
			log.Printf("approvals: error getting approval: %v", err)
			respond.Error(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
			return
		}

		respond.JSON(w, http.StatusOK, approval)
	}
}

func approve(svc *approvals.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if id == "" || len(strings.TrimSpace(id)) == 0 {
			respond.Error(w, http.StatusUnprocessableEntity, "validation_error", "id is required and must not be blank")
			return
		}

		approval, err := svc.Approve(r.Context(), id)
		if err != nil {
			if err == approvals.ErrNotFound {
				respond.Error(w, http.StatusNotFound, "not_found", "approval not found")
				return
			}
			if err == approvals.ErrAlreadyResolved {
				respond.Error(w, http.StatusConflict, "conflict", "approval is already resolved")
				return
			}
			log.Printf("approvals: error approving: %v", err)
			respond.Error(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
			return
		}

		respond.JSON(w, http.StatusOK, approval)
	}
}

func reject(svc *approvals.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if id == "" || len(strings.TrimSpace(id)) == 0 {
			respond.Error(w, http.StatusUnprocessableEntity, "validation_error", "id is required and must not be blank")
			return
		}

		approval, err := svc.Reject(r.Context(), id)
		if err != nil {
			if err == approvals.ErrNotFound {
				respond.Error(w, http.StatusNotFound, "not_found", "approval not found")
				return
			}
			if err == approvals.ErrAlreadyResolved {
				respond.Error(w, http.StatusConflict, "conflict", "approval is already resolved")
				return
			}
			log.Printf("approvals: error rejecting: %v", err)
			respond.Error(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
			return
		}

		respond.JSON(w, http.StatusOK, approval)
	}
}
