// Package issues provides CRUD operations for Issue entities.
package issues

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

// ErrNotFound is returned when a requested issue does not exist.
var ErrNotFound = errors.New("issue not found")

// ErrCheckoutConflict is returned when attempting to checkout an already checked-out issue.
var ErrCheckoutConflict = errors.New("issue already checked out")

// ErrNotCheckedOut is returned when attempting to release an issue not held by the agent.
var ErrNotCheckedOut = errors.New("issue not checked out by this agent")

// Service provides issue CRUD backed by the store.
type Service struct {
	store *store.Store
}

// New returns a Service using the given store.
func New(s *store.Store) *Service {
	return &Service{store: s}
}

// Create inserts a new issue and returns the created entity.
func (s *Service) Create(ctx context.Context, companyID, title, body string, assigneeID *string) (*domain.Issue, error) {
	now := time.Now().UTC().Truncate(time.Second)
	ts := now.Format(time.RFC3339)
	i := &domain.Issue{
		ID:        ids.NewUUID(),
		CompanyID: companyID,
		Title:     title,
		Body:      body,
		Status:    "open",
		AssigneeID: assigneeID,
		CreatedAt: now,
		UpdatedAt: now,
	}
	_, err := s.store.DB.ExecContext(ctx,
		`INSERT INTO issues(id, company_id, title, body, status, assignee_id, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		i.ID, i.CompanyID, i.Title, i.Body, i.Status, i.AssigneeID, ts, ts,
	)
	if err != nil {
		return nil, fmt.Errorf("inserting issue: %w", err)
	}
	return i, nil
}

// Get returns the issue with the given ID, or ErrNotFound if it doesn't exist.
func (s *Service) Get(ctx context.Context, id string) (*domain.Issue, error) {
	row := s.store.DB.QueryRowContext(ctx,
		`SELECT id, company_id, title, body, status, assignee_id, checked_out_by, checked_out_at, parent_issue_id, created_at, updated_at
		 FROM issues WHERE id = ?`, id,
	)
	i, err := scanIssue(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return i, err
}

// ListByCompany returns all issues for a given company, ordered by creation time descending.
func (s *Service) ListByCompany(ctx context.Context, companyID string) ([]*domain.Issue, error) {
	rows, err := s.store.DB.QueryContext(ctx,
		`SELECT id, company_id, title, body, status, assignee_id, checked_out_by, checked_out_at, parent_issue_id, created_at, updated_at
		 FROM issues WHERE company_id = ? ORDER BY created_at DESC`,
		companyID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing issues by company: %w", err)
	}
	defer rows.Close()

	out := make([]*domain.Issue, 0)
	for rows.Next() {
		i, err := scanIssue(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating issues: %w", err)
	}
	return out, nil
}

// ListWithFilters returns issues for a company with optional status and assignee filters, ordered by creation time descending.
func (s *Service) ListWithFilters(ctx context.Context, companyID, status string, assigneeID *string) ([]*domain.Issue, error) {
	query := `SELECT id, company_id, title, body, status, assignee_id, checked_out_by, checked_out_at, parent_issue_id, created_at, updated_at
	          FROM issues WHERE company_id = ?`
	args := []interface{}{companyID}

	if status != "" {
		query += ` AND status = ?`
		args = append(args, status)
	}

	if assigneeID != nil {
		query += ` AND assignee_id = ?`
		args = append(args, *assigneeID)
	}

	query += ` ORDER BY created_at DESC`

	rows, err := s.store.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing issues with filters: %w", err)
	}
	defer rows.Close()

	out := make([]*domain.Issue, 0)
	for rows.Next() {
		i, err := scanIssue(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating issues: %w", err)
	}
	return out, nil
}

// Update updates the status and/or assignee of an issue.
func (s *Service) Update(ctx context.Context, id, status string, assigneeID *string) (*domain.Issue, error) {
	now := time.Now().UTC().Truncate(time.Second)
	ts := now.Format(time.RFC3339)

	// Build the UPDATE query dynamically
	query := `UPDATE issues SET updated_at = ?`
	args := []interface{}{ts}

	if status != "" {
		query += `, status = ?`
		args = append(args, status)
	}

	if assigneeID != nil {
		query += `, assignee_id = ?`
		args = append(args, *assigneeID)
	}

	query += ` WHERE id = ?`
	args = append(args, id)

	_, err := s.store.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("updating issue: %w", err)
	}

	// Fetch and return the updated issue
	return s.Get(ctx, id)
}

// Checkout atomically checks out an issue for an agent.
// Returns nil if the agent already holds the issue (idempotent).
// Returns ErrNotFound if the issue does not exist.
// Returns ErrCheckoutConflict if the issue is checked out by a different agent.
func (s *Service) Checkout(ctx context.Context, issueID, agentID string) error {
	now := time.Now().UTC().Truncate(time.Second)
	ts := now.Format(time.RFC3339)

	// Try atomic update: only succeeds if issue exists and is not checked out
	result, err := s.store.DB.ExecContext(ctx,
		`UPDATE issues SET checked_out_by = ?, checked_out_at = ?, updated_at = ?, status = 'in_progress'
		 WHERE id = ? AND checked_out_by IS NULL`,
		agentID, ts, ts, issueID,
	)
	if err != nil {
		return fmt.Errorf("checking out issue: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("getting rows affected: %w", err)
	}

	if rowsAffected == 1 {
		return nil
	}

	// UPDATE affected 0 rows: either issue doesn't exist or is already checked out
	// Query to distinguish the two cases
	var checkedOutBy sql.NullString
	err = s.store.DB.QueryRowContext(ctx,
		`SELECT checked_out_by FROM issues WHERE id = ?`,
		issueID,
	).Scan(&checkedOutBy)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("checking issue state: %w", err)
	}

	// Issue exists; check if same agent or different agent
	if checkedOutBy.Valid && checkedOutBy.String == agentID {
		// Same agent already holds this issue (idempotent success)
		return nil
	}

	// Different agent holds it (or it's held by someone)
	return ErrCheckoutConflict
}

// Release releases an issue that was checked out by an agent.
// Returns ErrNotFound if the issue does not exist.
// Returns ErrNotCheckedOut if the issue is not held by the specified agent.
func (s *Service) Release(ctx context.Context, issueID, agentID string) error {
	now := time.Now().UTC().Truncate(time.Second)
	ts := now.Format(time.RFC3339)

	// Try atomic update: only succeeds if issue exists and is held by this agent
	result, err := s.store.DB.ExecContext(ctx,
		`UPDATE issues SET checked_out_by = NULL, checked_out_at = NULL, updated_at = ?
		 WHERE id = ? AND checked_out_by = ?`,
		ts, issueID, agentID,
	)
	if err != nil {
		return fmt.Errorf("releasing issue: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("getting rows affected: %w", err)
	}

	if rowsAffected == 1 {
		return nil
	}

	// UPDATE affected 0 rows: either issue doesn't exist or is not held by this agent
	// Query to check if issue exists at all
	var exists sql.NullString
	err = s.store.DB.QueryRowContext(ctx,
		`SELECT id FROM issues WHERE id = ? LIMIT 1`,
		issueID,
	).Scan(&exists)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("checking issue exists: %w", err)
	}

	// Issue exists but not held by this agent
	return ErrNotCheckedOut
}

// scanner is satisfied by both *sql.Row and *sql.Rows.
type scanner interface {
	Scan(dest ...any) error
}

func scanIssue(s scanner) (*domain.Issue, error) {
	var i domain.Issue
	var createdAt, updatedAt string
	var checkedOutAt *string
	var checkedOutBy, assigneeID, parentIssueID *string

	if err := s.Scan(&i.ID, &i.CompanyID, &i.Title, &i.Body, &i.Status, &assigneeID, &checkedOutBy, &checkedOutAt, &parentIssueID, &createdAt, &updatedAt); err != nil {
		return nil, err
	}

	i.AssigneeID = assigneeID
	i.CheckedOutBy = checkedOutBy
	i.ParentIssueID = parentIssueID

	var err error
	i.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return nil, fmt.Errorf("parsing created_at %q: %w", createdAt, err)
	}
	i.UpdatedAt, err = time.Parse(time.RFC3339, updatedAt)
	if err != nil {
		return nil, fmt.Errorf("parsing updated_at %q: %w", updatedAt, err)
	}

	if checkedOutAt != nil {
		parsedTime, err := time.Parse(time.RFC3339, *checkedOutAt)
		if err != nil {
			return nil, fmt.Errorf("parsing checked_out_at %q: %w", *checkedOutAt, err)
		}
		i.CheckedOutAt = &parsedTime
	}

	return &i, nil
}
