// Package issues provides HTTP handlers for the /api/issues routes.
package issues

import (
	"errors"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	asvc "github.com/ubunatic/paperclip-go/internal/activity"
	"github.com/ubunatic/paperclip-go/internal/comments"
	"github.com/ubunatic/paperclip-go/internal/domain"
	intesvc "github.com/ubunatic/paperclip-go/internal/interactions"
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
func Handler(issueSvc *isvc.Service, commentSvc *comments.Service, labelSvc *lsvc.Service, activityLog *asvc.Log, interactionSvc *intesvc.Service) http.Handler {
	r := chi.NewRouter()
	r.Get("/", list(issueSvc))
	r.Post("/", create(issueSvc))
	r.Get("/{id}", get(issueSvc, labelSvc))
	r.Patch("/{id}", update(issueSvc))
	r.Delete("/{id}", delete(issueSvc))
	r.Post("/{id}/checkout", checkout(issueSvc))
	r.Post("/{id}/release", release(issueSvc))
	r.Post("/{id}/archive", archive(issueSvc))
	r.Post("/{id}/unarchive", unarchive(issueSvc))
	r.Get("/{id}/comments", listComments(commentSvc))
	r.Post("/{id}/comments", createComment(commentSvc))
	r.Get("/{id}/activity", listActivity(activityLog))
	r.Post("/{id}/labels", addLabel(labelSvc))
	r.Delete("/{id}/labels/{labelId}", removeLabel(labelSvc))
	r.Get("/{id}/interactions", listInteractions(interactionSvc))
	r.Post("/{id}/interactions", createInteraction(interactionSvc))
	r.Post("/{id}/interactions/{iid}/resolve", resolveInteraction(interactionSvc))
	return r
}

func list(s *isvc.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		companyID := r.URL.Query().Get("companyId")
		status := r.URL.Query().Get("status")
		assigneeID := r.URL.Query().Get("assigneeId")
		includeArchived := r.URL.Query().Get("includeArchived") == "true"

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
			items, err = s.ListWithFilters(r.Context(), companyID, status, aid, includeArchived)
		} else {
			items, err = s.ListByCompany(r.Context(), companyID, includeArchived)
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
		var body struct {
			CompanyID         string  `json:"companyId"`
			Title             string  `json:"title"`
			Body              string  `json:"body"`
			Status            string  `json:"status"`
			OriginFingerprint string  `json:"originFingerprint"`
			AssigneeID        *string `json:"assigneeId"`
		}
		if !respond.DecodeJSON(w, r, &body) {
			return
		}
		if body.CompanyID == "" || body.Title == "" {
			respond.Error(w, http.StatusUnprocessableEntity, "validation_error", "companyId and title are required")
			return
		}
		issue, err := s.Create(r.Context(), body.CompanyID, body.Title, body.Body, body.OriginFingerprint, body.Status, body.AssigneeID)
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
		var body struct {
			Status       string  `json:"status"`
			AssigneeID   *string `json:"assigneeId"`
			Documents    *[]any  `json:"documents"`
			WorkProducts *[]any  `json:"workProducts"`
		}
		if !respond.DecodeJSON(w, r, &body) {
			return
		}
		// At least one field must be provided
		if body.Status == "" && body.AssigneeID == nil && body.Documents == nil && body.WorkProducts == nil {
			respond.Error(w, http.StatusUnprocessableEntity, "validation_error", "at least one of status, assigneeId, documents, or workProducts is required")
			return
		}
		issue, err := s.Update(r.Context(), id, body.Status, body.AssigneeID, body.Documents, body.WorkProducts)
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
		var body struct {
			AgentID string `json:"agentId"`
		}
		if !respond.DecodeJSON(w, r, &body) {
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
		var body struct {
			AgentID string `json:"agentId"`
		}
		if !respond.DecodeJSON(w, r, &body) {
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

func archive(s *isvc.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		err := s.Archive(r.Context(), id)
		if err != nil {
			if errors.Is(err, isvc.ErrNotFound) {
				respond.Error(w, http.StatusNotFound, "not_found", "issue not found")
				return
			}
			log.Printf("issues: error: %v", err)
			respond.Error(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
			return
		}
		respond.JSON(w, http.StatusOK, map[string]string{"status": "archived"})
	}
}

func unarchive(s *isvc.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		err := s.Unarchive(r.Context(), id)
		if err != nil {
			if errors.Is(err, isvc.ErrNotFound) {
				respond.Error(w, http.StatusNotFound, "not_found", "issue not found")
				return
			}
			log.Printf("issues: error: %v", err)
			respond.Error(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
			return
		}
		respond.JSON(w, http.StatusOK, map[string]string{"status": "unarchived"})
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

func listActivity(a *asvc.Log) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		issueID := chi.URLParam(r, "id")
		items, err := a.ListByEntity(r.Context(), "issue", issueID, 500)
		if err != nil {
			log.Printf("activity: error: %v", err)
			respond.Error(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
			return
		}
		respond.JSON(w, http.StatusOK, map[string]any{"items": items})
	}
}

func createComment(s *comments.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		issueID := chi.URLParam(r, "id")
		var body struct {
			Body          string  `json:"body"`
			AuthorAgentID *string `json:"authorAgentId"`
			AuthorKind    string  `json:"authorKind"`
		}
		if !respond.DecodeJSON(w, r, &body) {
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
		var body struct {
			LabelID string `json:"labelId"`
		}
		if !respond.DecodeJSON(w, r, &body) {
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
				respond.Error(w, http.StatusNotFound, "issue_not_found", "issue not found")
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
			if errors.Is(err, lsvc.ErrAssociationNotFound) {
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

func listInteractions(s *intesvc.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		issueID := chi.URLParam(r, "id")
		items, err := s.ListByIssue(r.Context(), issueID)
		if err != nil {
			log.Printf("interactions: error: %v", err)
			respond.Error(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
			return
		}
		respond.JSON(w, http.StatusOK, map[string]any{"items": items})
	}
}

func createInteraction(s *intesvc.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		issueID := chi.URLParam(r, "id")
		var body struct {
			CompanyID      string  `json:"companyId"`
			AgentID        *string `json:"agentId"`
			CommentID      *string `json:"commentId"`
			RunID          *string `json:"runId"`
			Kind           string  `json:"kind"`
			IdempotencyKey string  `json:"idempotencyKey"`
		}
		if !respond.DecodeJSON(w, r, &body) {
			return
		}
		if body.Kind == "" {
			respond.Error(w, http.StatusUnprocessableEntity, "validation_error", "kind is required")
			return
		}
		input := intesvc.CreateInput{
			CompanyID:      body.CompanyID,
			IssueID:        issueID,
			AgentID:        body.AgentID,
			CommentID:      body.CommentID,
			RunID:          body.RunID,
			Kind:           body.Kind,
			IdempotencyKey: body.IdempotencyKey,
		}
		interaction, err := s.Create(r.Context(), input)
		if err != nil {
			log.Printf("interactions: error: %v", err)
			respond.Error(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
			return
		}
		// Return 201 for new, 200 for idempotency dedup
		// Check if this is a dedup by checking if the returned interaction was just created
		// (can't easily distinguish without tracking, so we return 201 always)
		respond.JSON(w, http.StatusCreated, interaction)
	}
}

func resolveInteraction(s *intesvc.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		iid := chi.URLParam(r, "iid")
		var body struct {
			ResolvedByAgentID string  `json:"resolvedByAgentId"`
			Result            *string `json:"result"`
		}
		if !respond.DecodeJSON(w, r, &body) {
			return
		}
		if body.ResolvedByAgentID == "" {
			respond.Error(w, http.StatusUnprocessableEntity, "validation_error", "resolvedByAgentId is required")
			return
		}
		interaction, err := s.Resolve(r.Context(), iid, body.ResolvedByAgentID, body.Result)
		if err != nil {
			if errors.Is(err, intesvc.ErrNotFound) {
				respond.Error(w, http.StatusNotFound, "not_found", "interaction not found")
				return
			}
			if errors.Is(err, intesvc.ErrAlreadyResolved) {
				respond.Error(w, http.StatusConflict, "conflict", "interaction is already resolved")
				return
			}
			log.Printf("interactions: error: %v", err)
			respond.Error(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
			return
		}
		respond.JSON(w, http.StatusOK, interaction)
	}
}
