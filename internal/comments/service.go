// Package comments provides CRUD operations for Comment entities.
package comments

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

// ErrIssueNotFound is returned when a referenced issue does not exist.
var ErrIssueNotFound = errors.New("issue not found")

// Service provides comment CRUD backed by the store.
type Service struct {
	store *store.Store
}

// New returns a Service using the given store.
func New(s *store.Store) *Service {
	return &Service{store: s}
}

// Create inserts a new comment and returns the created entity.
// Returns ErrIssueNotFound if the issue does not exist.
func (s *Service) Create(ctx context.Context, issueID string, authorAgentID *string, authorKind, body string) (*domain.Comment, error) {
	// Verify the issue exists
	if err := s.issueExists(ctx, issueID); err != nil {
		return nil, err
	}

	now := time.Now().UTC().Truncate(time.Second)
	ts := now.Format(time.RFC3339)
	c := &domain.Comment{
		ID:            ids.NewUUID(),
		IssueID:       issueID,
		AuthorAgentID: authorAgentID,
		AuthorKind:    authorKind,
		Body:          body,
		CreatedAt:     now,
	}
	_, err := s.store.DB.ExecContext(ctx,
		`INSERT INTO comments(id, issue_id, author_agent_id, author_kind, body, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		c.ID, c.IssueID, c.AuthorAgentID, c.AuthorKind, c.Body, ts,
	)
	if err != nil {
		return nil, fmt.Errorf("inserting comment: %w", err)
	}
	return c, nil
}

// ListByIssue returns all comments for a given issue, ordered by creation time ascending (with id as tiebreaker for determinism).
func (s *Service) ListByIssue(ctx context.Context, issueID string) ([]*domain.Comment, error) {
	rows, err := s.store.DB.QueryContext(ctx,
		`SELECT id, issue_id, author_agent_id, author_kind, body, created_at
		 FROM comments WHERE issue_id = ? ORDER BY created_at ASC, id ASC`,
		issueID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing comments by issue: %w", err)
	}
	defer rows.Close()

	out := make([]*domain.Comment, 0)
	for rows.Next() {
		c, err := scanComment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating comments: %w", err)
	}
	return out, nil
}

// scanner is satisfied by both *sql.Row and *sql.Rows.
type scanner interface {
	Scan(dest ...any) error
}

func scanComment(s scanner) (*domain.Comment, error) {
	var c domain.Comment
	var createdAt string
	var authorAgentID *string

	if err := s.Scan(&c.ID, &c.IssueID, &authorAgentID, &c.AuthorKind, &c.Body, &createdAt); err != nil {
		return nil, err
	}

	c.AuthorAgentID = authorAgentID

	var err error
	c.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return nil, fmt.Errorf("parsing created_at %q: %w", createdAt, err)
	}

	return &c, nil
}

// issueExists checks if an issue with the given ID exists.
func (s *Service) issueExists(ctx context.Context, issueID string) error {
	var id string
	err := s.store.DB.QueryRowContext(ctx,
		`SELECT id FROM issues WHERE id = ? LIMIT 1`,
		issueID,
	).Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrIssueNotFound
		}
		return fmt.Errorf("checking issue exists: %w", err)
	}
	return nil
}
