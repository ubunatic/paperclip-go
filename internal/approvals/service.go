// Package approvals provides CRUD operations for Approval entities.
package approvals

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ubunatic/paperclip-go/internal/domain"
	"github.com/ubunatic/paperclip-go/internal/ids"
	"github.com/ubunatic/paperclip-go/internal/store"
)

// ErrNotFound is returned when a requested approval does not exist.
var ErrNotFound = errors.New("approval not found")

// ErrAlreadyResolved is returned when attempting to resolve an already-resolved approval.
var ErrAlreadyResolved = errors.New("approval is already resolved")

// ErrInvalidStatus is returned when an invalid status value is provided.
var ErrInvalidStatus = errors.New("invalid approval status")

// Service provides approval CRUD backed by the store.
type Service struct {
	store *store.Store
}

// New returns a Service using the given store.
func New(s *store.Store) *Service {
	return &Service{store: s}
}

// Create inserts a new approval and returns the created entity.
func (s *Service) Create(ctx context.Context, companyID, agentID, issueID, kind string, requestBody *string) (*domain.Approval, error) {
	now := time.Now().UTC().Truncate(time.Second)
	ts := now.Format(time.RFC3339)

	approval := &domain.Approval{
		ID:          ids.NewUUID(),
		CompanyID:   companyID,
		AgentID:     agentID,
		IssueID:     issueID,
		Kind:        kind,
		Status:      domain.ApprovalStatusPending,
		RequestBody: requestBody,
		CreatedAt:   now,
	}

	_, err := s.store.DB.ExecContext(ctx,
		`INSERT INTO approvals(id, company_id, agent_id, issue_id, kind, status, request_body, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		approval.ID, approval.CompanyID, approval.AgentID, approval.IssueID, approval.Kind, approval.Status, approval.RequestBody, ts,
	)
	if err != nil {
		return nil, fmt.Errorf("create approval: %w", err)
	}

	return approval, nil
}

// GetByID retrieves an approval by its ID.
func (s *Service) GetByID(ctx context.Context, id string) (*domain.Approval, error) {
	row := s.store.DB.QueryRowContext(ctx,
		`SELECT id, company_id, agent_id, issue_id, kind, status, request_body, response_body, created_at, resolved_at
		 FROM approvals WHERE id = ?`,
		id,
	)
	approval, err := scanApproval(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get approval: %w", err)
	}
	return approval, nil
}

// ListByCompany retrieves all approvals for a given company.
func (s *Service) ListByCompany(ctx context.Context, companyID string) ([]*domain.Approval, error) {
	rows, err := s.store.DB.QueryContext(ctx,
		`SELECT id, company_id, agent_id, issue_id, kind, status, request_body, response_body, created_at, resolved_at
		 FROM approvals WHERE company_id = ? ORDER BY created_at DESC`,
		companyID,
	)
	if err != nil {
		return nil, fmt.Errorf("list approvals: %w", err)
	}
	defer rows.Close()

	var approvals []*domain.Approval
	for rows.Next() {
		a, err := scanApproval(rows)
		if err != nil {
			return nil, fmt.Errorf("scan approval: %w", err)
		}
		approvals = append(approvals, a)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return approvals, nil
}

// Approve transitions an approval to "approved" status.
// Returns ErrNotFound if the approval doesn't exist.
// Returns ErrAlreadyResolved if the approval is already resolved.
func (s *Service) Approve(ctx context.Context, id string) (*domain.Approval, error) {
	return s.setState(ctx, id, domain.ApprovalStatusApproved)
}

// Reject transitions an approval to "rejected" status.
// Returns ErrNotFound if the approval doesn't exist.
// Returns ErrAlreadyResolved if the approval is already resolved.
func (s *Service) Reject(ctx context.Context, id string) (*domain.Approval, error) {
	return s.setState(ctx, id, domain.ApprovalStatusRejected)
}

// setState updates the status of an approval atomically.
// Returns ErrAlreadyResolved if already resolved.
func (s *Service) setState(ctx context.Context, id string, newStatus domain.ApprovalStatus) (*domain.Approval, error) {
	// Validate status
	if newStatus != domain.ApprovalStatusApproved && newStatus != domain.ApprovalStatusRejected {
		return nil, ErrInvalidStatus
	}

	now := time.Now().UTC().Truncate(time.Second)
	ts := now.Format(time.RFC3339)

	// Atomic UPDATE: only update if status is pending
	result, err := s.store.DB.ExecContext(ctx,
		`UPDATE approvals
		 SET status = ?, resolved_at = ?
		 WHERE id = ? AND status = ?`,
		newStatus, ts, id, domain.ApprovalStatusPending,
	)
	if err != nil {
		return nil, fmt.Errorf("set approval state: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("rows affected: %w", err)
	}

	// If 0 rows affected, either the approval doesn't exist or it's already resolved
	if rowsAffected == 0 {
		// Check if it exists at all
		existing, err := s.GetByID(ctx, id)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				return nil, ErrNotFound
			}
			return nil, err
		}

		// It exists, so it must already be resolved
		if existing.ResolvedAt != nil {
			return nil, ErrAlreadyResolved
		}

		// Shouldn't reach here, but be defensive
		return nil, ErrAlreadyResolved
	}

	// Fetch and return the updated approval
	return s.GetByID(ctx, id)
}

// scanner is an interface for *sql.Row or *sql.Rows.
type scanner interface {
	Scan(dest ...interface{}) error
}

// scanApproval scans an approval from a row or rows.
func scanApproval(s scanner) (*domain.Approval, error) {
	var approval domain.Approval
	var createdAt, resolvedAt *string

	if err := s.Scan(
		&approval.ID, &approval.CompanyID, &approval.AgentID, &approval.IssueID,
		&approval.Kind, &approval.Status, &approval.RequestBody, &approval.ResponseBody,
		&createdAt, &resolvedAt,
	); err != nil {
		return nil, err
	}

	// Parse created_at
	if createdAt == nil {
		return nil, fmt.Errorf("created_at is required")
	}
	var err error
	approval.CreatedAt, err = time.Parse(time.RFC3339, *createdAt)
	if err != nil {
		return nil, fmt.Errorf("parsing created_at %q: %w", *createdAt, err)
	}

	// Parse resolved_at if present
	if resolvedAt != nil {
		approval.ResolvedAt, err = stringToTimePtr(*resolvedAt)
		if err != nil {
			return nil, fmt.Errorf("parsing resolved_at %q: %w", *resolvedAt, err)
		}
	}

	return &approval, nil
}

// stringToTimePtr parses an RFC3339 string and returns a pointer to the parsed time.
func stringToTimePtr(s string) (*time.Time, error) {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil, err
	}
	return &t, nil
}
