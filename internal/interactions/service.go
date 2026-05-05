// Package interactions provides CRUD operations for Interaction entities.
package interactions

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ubunatic/paperclip-go/internal/domain"
	"github.com/ubunatic/paperclip-go/internal/ids"
	"github.com/ubunatic/paperclip-go/internal/store"
)

// ErrNotFound is returned when a requested interaction does not exist.
var ErrNotFound = errors.New("interaction not found")

// ErrAlreadyResolved is returned when attempting to resolve an already resolved interaction.
var ErrAlreadyResolved = errors.New("interaction already resolved")

// CreateInput holds the input for creating an interaction.
type CreateInput struct {
	CompanyID      string
	IssueID        string
	AgentID        *string
	CommentID      *string
	RunID          *string
	Kind           string
	IdempotencyKey string
}

// Service provides interaction CRUD backed by the store.
type Service struct {
	store *store.Store
}

// New returns a Service using the given store.
func New(s *store.Store) *Service {
	return &Service{store: s}
}

// Create inserts a new interaction and returns the created entity.
// If an interaction with the same issue_id and idempotency_key already exists,
// returns that existing interaction (idempotency dedup) with no error.
func (s *Service) Create(ctx context.Context, input CreateInput) (*domain.Interaction, error) {
	now := time.Now().UTC().Truncate(time.Second)
	ts := now.Format(time.RFC3339)

	// Try to create the interaction
	id := ids.NewUUID()
	_, err := s.store.DB.ExecContext(ctx,
		`INSERT INTO issue_thread_interactions(id, company_id, issue_id, agent_id, comment_id, run_id, kind, status, idempotency_key, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, input.CompanyID, input.IssueID, input.AgentID, input.CommentID, input.RunID, input.Kind, domain.InteractionStatusPending, input.IdempotencyKey, ts, ts,
	)
	if err != nil {
		// Check if this is a UNIQUE constraint violation on (issue_id, idempotency_key)
		if strings.Contains(err.Error(), "UNIQUE constraint failed") && strings.Contains(err.Error(), "idempotency_key") {
			// Idempotency dedup: return the existing interaction
			return s.GetByIdempotencyKey(ctx, input.IssueID, input.IdempotencyKey)
		}
		return nil, fmt.Errorf("inserting interaction: %w", err)
	}

	return s.GetByID(ctx, id)
}

// GetByID returns the interaction with the given ID, or ErrNotFound if it doesn't exist.
func (s *Service) GetByID(ctx context.Context, id string) (*domain.Interaction, error) {
	row := s.store.DB.QueryRowContext(ctx,
		`SELECT id, company_id, issue_id, agent_id, comment_id, run_id, kind, status, idempotency_key, result, resolved_at, resolved_by_agent_id, created_at, updated_at
		 FROM issue_thread_interactions WHERE id = ?`, id,
	)
	i, err := scanInteraction(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return i, err
}

// GetByIdempotencyKey returns the interaction with the given issue_id and idempotency_key,
// or ErrNotFound if it doesn't exist.
func (s *Service) GetByIdempotencyKey(ctx context.Context, issueID, idempotencyKey string) (*domain.Interaction, error) {
	row := s.store.DB.QueryRowContext(ctx,
		`SELECT id, company_id, issue_id, agent_id, comment_id, run_id, kind, status, idempotency_key, result, resolved_at, resolved_by_agent_id, created_at, updated_at
		 FROM issue_thread_interactions WHERE issue_id = ? AND idempotency_key = ?`, issueID, idempotencyKey,
	)
	i, err := scanInteraction(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return i, err
}

// ListByIssue returns all interactions for a given issue, ordered by creation time descending.
// Returns an empty slice (not nil) if no interactions exist.
func (s *Service) ListByIssue(ctx context.Context, issueID string) ([]*domain.Interaction, error) {
	rows, err := s.store.DB.QueryContext(ctx,
		`SELECT id, company_id, issue_id, agent_id, comment_id, run_id, kind, status, idempotency_key, result, resolved_at, resolved_by_agent_id, created_at, updated_at
		 FROM issue_thread_interactions WHERE issue_id = ? ORDER BY created_at DESC`,
		issueID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing interactions by issue: %w", err)
	}
	defer rows.Close()

	out := make([]*domain.Interaction, 0)
	for rows.Next() {
		i, err := scanInteraction(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating interactions: %w", err)
	}
	return out, nil
}

// Resolve atomically resolves an interaction.
// Returns ErrNotFound if the interaction does not exist.
// Returns ErrAlreadyResolved if the interaction is already resolved.
func (s *Service) Resolve(ctx context.Context, id, resolvedByAgentID string, result *string) (*domain.Interaction, error) {
	now := time.Now().UTC().Truncate(time.Second)
	ts := now.Format(time.RFC3339)

	// Try atomic update: only succeeds if interaction exists and is not already resolved
	resultUpd := result
	result_, err := s.store.DB.ExecContext(ctx,
		`UPDATE issue_thread_interactions SET status = ?, resolved_at = ?, resolved_by_agent_id = ?, result = ?, updated_at = ?
		 WHERE id = ? AND status = ?`,
		domain.InteractionStatusResolved, ts, resolvedByAgentID, resultUpd, ts, id, domain.InteractionStatusPending,
	)
	if err != nil {
		return nil, fmt.Errorf("resolving interaction: %w", err)
	}

	// Check how many rows were affected
	rowsAffected, err := result_.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("getting rows affected: %w", err)
	}

	// If 0 rows were affected, either interaction doesn't exist or is already resolved
	if rowsAffected == 0 {
		// Check if interaction exists
		_, err2 := s.GetByID(ctx, id)
		if errors.Is(err2, ErrNotFound) {
			return nil, ErrNotFound
		}
		if err2 != nil {
			return nil, err2
		}
		// Interaction exists, so it must be already resolved
		return nil, ErrAlreadyResolved
	}

	// Update succeeded, fetch and return the interaction
	interaction, err := s.GetByID(ctx, id)
	if errors.Is(err, ErrNotFound) {
		// Should not happen, but handle gracefully
		return nil, ErrNotFound
	}
	return interaction, err
}

// scanner is satisfied by both *sql.Row and *sql.Rows.
type scanner interface {
	Scan(dest ...any) error
}

func scanInteraction(s scanner) (*domain.Interaction, error) {
	var i domain.Interaction
	var createdAt, updatedAt string
	var resolvedAt *string

	if err := s.Scan(&i.ID, &i.CompanyID, &i.IssueID, &i.AgentID, &i.CommentID, &i.RunID, &i.Kind, &i.Status, &i.IdempotencyKey, &i.Result, &resolvedAt, &i.ResolvedByAgentID, &createdAt, &updatedAt); err != nil {
		return nil, err
	}

	var err error
	i.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return nil, fmt.Errorf("parsing created_at %q: %w", createdAt, err)
	}
	i.UpdatedAt, err = time.Parse(time.RFC3339, updatedAt)
	if err != nil {
		return nil, fmt.Errorf("parsing updated_at %q: %w", updatedAt, err)
	}

	if resolvedAt != nil {
		parsedTime, err := time.Parse(time.RFC3339, *resolvedAt)
		if err != nil {
			return nil, fmt.Errorf("parsing resolved_at %q: %w", *resolvedAt, err)
		}
		i.ResolvedAt = &parsedTime
	}

	return &i, nil
}
