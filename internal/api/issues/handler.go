// Package issues provides HTTP handlers for the /api/issues routes.
package issues

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/ubunatic/paperclip-go/internal/comments"
	"github.com/ubunatic/paperclip-go/internal/domain"
	isvc "github.com/ubunatic/paperclip-go/internal/issues"
	lsvc "github.com/ubunatic/paperclip-go/internal/labels"
	"github.com/ubunatic/paperclip-go/internal/respond"
)

// issueWithLabels is the response struct for issue GET, including associated labels.
type issueWithLabels struct {
	*domain.Issue
	Labels []*domain.Label `json:"labels"`
}

// Handler returns an http.Handler for the /api/issues sub-router.
func Handler(issueSvc *isvc.Service, commentSvc *comments.Service, labelSvc *lsvc.Service) http.Handler {
	r := chi.NewRouter()
	r.Get("/", list(issueSvc))
	r.Post("/", create(issueSvc))
	r.Get("/{id}", get(issueSvc, labelSvc))
	r.Patch("/{id}", update(issueSvc))
	r.Delete("/{id}", delete(issueSvc))
	r.Post("/{id}/checkout", checkout(issueSvc))
	r.Post("/{id}/release", release(issueSvc))
	r.Get("/{id}/comments", listComments(commentSvc))
	r.Post("/{id}/comments", createComment(commentSvc))
	r.Post("/{id}/labels", addLabel(labelSvc))
	r.Delete("/{id}/labels/{labelId}", removeLabel(labelSvc))
	return r
}

func list(s *isvc.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		companyID := r.URL.Query().Get("companyId")
		status := r.URL.Query().Get("status")
		assigneeID := r.URL.Query().Get("assigneeId")

		if companyID == "" {
			respond.Error(w, http.StatusBadRequest, "validation_error", "companyId is required")
			return
		}

		// Validate status if provided
		if status != "" && !domain.IsValidIssueStatus(status) {
			respond.Error(w, http.StatusUnprocessableEntity, "validation_error", "invalid status value")
			return
		}

		var items []*domain.Issue
		var err error

		if status != "" || assigneeID != "" {
			// Use filters if provided
			var aid *string
			if assigneeID != "" {
				aid = &assigneeID
			}
			items, err = s.ListWithFilters(r.Context(), companyID, status, aid)
		} else {
			items, err = s.ListByCompany(r.Context(), companyID)
		}

		if err != nil {
			log.Printf("issues: error: %v", err)
			respond.Error(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
			return
		}

		respond.JSON(w, http.StatusOK, map[string]any{"items": items})
	}
}

func create(s *isvc.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MiB
		var body struct {
			CompanyID  string  `json:"companyId"`
			Title      string  `json:"title"`
			Body       string  `json:"body"`
			Status     string  `json:"status"`
			AssigneeID *string `json:"assigneeId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			respond.Error(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
			return
		}
		if body.CompanyID == "" || body.Title == "" {
			respond.Error(w, http.StatusUnprocessableEntity, "validation_error", "companyId and title are required")
			return
		}
		issue, err := s.Create(r.Context(), body.CompanyID, body.Title, body.Body, body.Status, body.AssigneeID)
		if err != nil {
			if errors.Is(err, isvc.ErrInvalidStatus) {
				respond.Error(w, http.StatusUnprocessableEntity, "validation_error", "invalid status value")
				return
			}
			log.Printf("issues: error: %v", err)
			respond.Error(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
			return
		}
		respond.JSON(w, http.StatusCreated, issue)
	}
}

func get(s *isvc.Service, labelSvc *lsvc.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		issue, err := s.Get(r.Context(), id)
		if err != nil {
			if errors.Is(err, isvc.ErrNotFound) {
				respond.Error(w, http.StatusNotFound, "not_found", "issue not found")
				return
			}
			log.Printf("issues: error: %v", err)
			respond.Error(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
			return
		}

		// Load labels for this issue
		labels, err := labelSvc.GetLabelsForIssue(r.Context(), id)
		if err != nil {
			log.Printf("issues: error loading labels: %v", err)
			respond.Error(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
			return
		}

		respond.JSON(w, http.StatusOK, issueWithLabels{
			Issue:  issue,
			Labels: labels,
		})
	}
}

func update(s *isvc.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MiB
		var body struct {
			Status     string  `json:"status"`
			AssigneeID *string `json:"assigneeId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			respond.Error(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
			return
		}
		// At least one field must be provided
		if body.Status == "" && body.AssigneeID == nil {
			respond.Error(w, http.StatusUnprocessableEntity, "validation_error", "at least one of status or assigneeId is required")
			return
		}
		issue, err := s.Update(r.Context(), id, body.Status, body.AssigneeID)
		if err != nil {
			if errors.Is(err, isvc.ErrNotFound) {
				respond.Error(w, http.StatusNotFound, "not_found", "issue not found")
				return
			}
			if errors.Is(err, isvc.ErrInvalidStatus) {
				respond.Error(w, http.StatusUnprocessableEntity, "validation_error", "invalid status value")
				return
			}
			log.Printf("issues: error: %v", err)
			respond.Error(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
			return
		}
		respond.JSON(w, http.StatusOK, issue)
	}
}

func checkout(s *isvc.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		issueID := chi.URLParam(r, "id")
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MiB
		var body struct {
			AgentID string `json:"agentId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			respond.Error(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
			return
		}
		if body.AgentID == "" {
			respond.Error(w, http.StatusUnprocessableEntity, "validation_error", "agentId is required")
			return
		}
		err := s.Checkout(r.Context(), issueID, body.AgentID)
		if err != nil {
			if errors.Is(err, isvc.ErrNotFound) {
				respond.Error(w, http.StatusNotFound, "not_found", "issue not found")
				return
			}
			if errors.Is(err, isvc.ErrCheckoutConflict) {
				respond.Error(w, http.StatusConflict, "checkout_conflict", "issue is already checked out")
				return
			}
			log.Printf("issues: error: %v", err)
			respond.Error(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
			return
		}
		respond.JSON(w, http.StatusOK, map[string]string{"status": "checked_out"})
	}
}

func release(s *isvc.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		issueID := chi.URLParam(r, "id")
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MiB
		var body struct {
			AgentID string `json:"agentId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			respond.Error(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
			return
		}
		if body.AgentID == "" {
			respond.Error(w, http.StatusUnprocessableEntity, "validation_error", "agentId is required")
			return
		}
		err := s.Release(r.Context(), issueID, body.AgentID)
		if err != nil {
			if errors.Is(err, isvc.ErrNotFound) {
				respond.Error(w, http.StatusNotFound, "not_found", "issue not found")
				return
			}
			if errors.Is(err, isvc.ErrNotCheckedOut) {
				respond.Error(w, http.StatusBadRequest, "not_held_by_agent", "issue is not held by this agent")
				return
			}
			log.Printf("issues: error: %v", err)
			respond.Error(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
			return
		}
		respond.JSON(w, http.StatusOK, map[string]string{"status": "released"})
	}
}

func delete(s *isvc.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		err := s.Delete(r.Context(), id)
		if err != nil {
			if errors.Is(err, isvc.ErrNotFound) {
				respond.Error(w, http.StatusNotFound, "not_found", "issue not found")
				return
			}
			if errors.Is(err, isvc.ErrCheckoutConflictDelete) {
				respond.Error(w, http.StatusConflict, "checkout_conflict", "cannot delete checked-out issue")
				return
			}
			log.Printf("issues: error: %v", err)
			respond.Error(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func listComments(s *comments.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		issueID := chi.URLParam(r, "id")
		items, err := s.ListByIssue(r.Context(), issueID)
		if err != nil {
			log.Printf("comments: error: %v", err)
			respond.Error(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
			return
		}
		respond.JSON(w, http.StatusOK, map[string]any{"items": items})
	}
}

func createComment(s *comments.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		issueID := chi.URLParam(r, "id")
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MiB
		var body struct {
			Body          string  `json:"body"`
			AuthorAgentID *string `json:"authorAgentId"`
			AuthorKind    string  `json:"authorKind"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			respond.Error(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
			return
		}
		if body.Body == "" {
			respond.Error(w, http.StatusUnprocessableEntity, "validation_error", "body is required")
			return
		}
		if body.AuthorKind == "" {
			body.AuthorKind = "system"
		}
		if body.AuthorKind == "agent" && body.AuthorAgentID == nil {
			respond.Error(w, http.StatusUnprocessableEntity, "validation_error", "authorAgentId is required when authorKind is agent")
			return
		}
		if body.AuthorKind != "agent" && body.AuthorAgentID != nil {
			respond.Error(w, http.StatusUnprocessableEntity, "validation_error", "authorAgentId is only allowed when authorKind is agent")
			return
		}
		comment, err := s.Create(r.Context(), issueID, body.AuthorAgentID, body.AuthorKind, body.Body)
		if err != nil {
			if errors.Is(err, comments.ErrIssueNotFound) {
				respond.Error(w, http.StatusNotFound, "issue_not_found", "issue not found")
				return
			}
			log.Printf("comments: error: %v", err)
			respond.Error(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
			return
		}
		respond.JSON(w, http.StatusCreated, comment)
	}
}

func addLabel(labelSvc *lsvc.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		issueID := chi.URLParam(r, "id")
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MiB
		var body struct {
			LabelID string `json:"labelId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			respond.Error(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
			return
		}
		if body.LabelID == "" {
			respond.Error(w, http.StatusUnprocessableEntity, "validation_error", "labelId is required")
			return
		}

		// LinkToIssue validates issue/label existence and company match
		err := labelSvc.LinkToIssue(r.Context(), issueID, body.LabelID)
		if err != nil {
			switch {
			case errors.Is(err, lsvc.ErrIssueNotFound):
				respond.Error(w, http.StatusNotFound, "not_found", "issue not found")
				return
			case errors.Is(err, lsvc.ErrNotFound):
				respond.Error(w, http.StatusNotFound, "label_not_found", "label not found")
				return
			case errors.Is(err, lsvc.ErrCompanyMismatch):
				respond.Error(w, http.StatusConflict, "conflict", "issue and label are in different companies")
				return
			default:
				log.Printf("issues: error linking label to issue: %v", err)
				respond.Error(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
				return
			}
		}

		respond.JSON(w, http.StatusOK, map[string]string{"status": "added"})
	}
}

func removeLabel(labelSvc *lsvc.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		issueID := chi.URLParam(r, "id")
		labelID := chi.URLParam(r, "labelId")

		err := labelSvc.UnlinkFromIssue(r.Context(), issueID, labelID)
		if err != nil {
			if errors.Is(err, lsvc.ErrNotFound) {
				respond.Error(w, http.StatusNotFound, "not_found", "label association not found")
				return
			}
			log.Printf("issues: error unlinking from issue: %v", err)
			respond.Error(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
