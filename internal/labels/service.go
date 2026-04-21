// Package labels provides CRUD operations for Label entities.
package labels

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

// ErrNotFound is returned when a requested label does not exist.
var ErrNotFound = errors.New("label not found")

// ErrDuplicate is returned when attempting to create a label with a name that already exists for the company.
var ErrDuplicate = errors.New("label with this name already exists for the company")

// ErrIssueNotFound is returned when a referenced issue does not exist.
var ErrIssueNotFound = errors.New("issue not found")

// Service provides label CRUD backed by the store.
type Service struct {
	store *store.Store
}

// New returns a Service using the given store.
func New(s *store.Store) *Service {
	return &Service{store: s}
}

// Create inserts a new label and returns the created entity.
// Returns ErrDuplicate if a label with the same name already exists for the company.
func (s *Service) Create(ctx context.Context, companyID, name, color string) (*domain.Label, error) {
	now := time.Now().UTC().Truncate(time.Second)
	ts := now.Format(time.RFC3339)
	label := &domain.Label{
		ID:        ids.NewUUID(),
		CompanyID: companyID,
		Name:      name,
		Color:     color,
		CreatedAt: now,
		UpdatedAt: now,
	}
	_, err := s.store.DB.ExecContext(ctx,
		`INSERT INTO labels(id, company_id, name, color, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		label.ID, label.CompanyID, label.Name, label.Color, ts, ts,
	)
	if err != nil {
		return nil, fmt.Errorf("inserting label: %w", err)
	}
	return label, nil
}

// Get returns the label with the given ID, or ErrNotFound if it doesn't exist.
func (s *Service) Get(ctx context.Context, id string) (*domain.Label, error) {
	row := s.store.DB.QueryRowContext(ctx,
		`SELECT id, company_id, name, color, created_at, updated_at
		 FROM labels WHERE id = ?`, id,
	)
	label, err := scanLabel(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return label, err
}

// ListByCompany returns all labels for a given company, ordered by name.
func (s *Service) ListByCompany(ctx context.Context, companyID string) ([]*domain.Label, error) {
	rows, err := s.store.DB.QueryContext(ctx,
		`SELECT id, company_id, name, color, created_at, updated_at
		 FROM labels WHERE company_id = ? ORDER BY name ASC`,
		companyID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing labels by company: %w", err)
	}
	defer rows.Close()

	out := make([]*domain.Label, 0)
	for rows.Next() {
		label, err := scanLabel(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, label)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating labels: %w", err)
	}
	return out, nil
}

// Delete deletes a label.
// Returns ErrNotFound if the label does not exist.
func (s *Service) Delete(ctx context.Context, id string) error {
	result, err := s.store.DB.ExecContext(ctx,
		`DELETE FROM labels WHERE id = ?`,
		id,
	)
	if err != nil {
		return fmt.Errorf("deleting label: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("getting rows affected: %w", err)
	}
	if rowsAffected != 1 {
		return ErrNotFound
	}
	return nil
}

// GetByNameAndCompany returns the label with the given name for the company, or nil if not found.
func (s *Service) GetByNameAndCompany(ctx context.Context, companyID, name string) (*domain.Label, error) {
	row := s.store.DB.QueryRowContext(ctx,
		`SELECT id, company_id, name, color, created_at, updated_at
		 FROM labels WHERE company_id = ? AND name = ?`,
		companyID, name,
	)
	label, err := scanLabel(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return label, err
}

// LinkToIssue adds a label to an issue (idempotent).
// Returns ErrIssueNotFound if the issue does not exist.
func (s *Service) LinkToIssue(ctx context.Context, issueID, labelID string) error {
	// Verify the issue exists
	if err := s.issueExists(ctx, issueID); err != nil {
		return err
	}

	now := time.Now().UTC().Truncate(time.Second)
	ts := now.Format(time.RFC3339)

	// Insert with IGNORE or check for existing, then insert if not exists
	// For SQLite, we use INSERT OR IGNORE
	_, err := s.store.DB.ExecContext(ctx,
		`INSERT OR IGNORE INTO issue_labels(issue_id, label_id, created_at)
		 VALUES (?, ?, ?)`,
		issueID, labelID, ts,
	)
	if err != nil {
		return fmt.Errorf("linking label to issue: %w", err)
	}
	return nil
}

// UnlinkFromIssue removes a label from an issue (idempotent).
func (s *Service) UnlinkFromIssue(ctx context.Context, issueID, labelID string) error {
	_, err := s.store.DB.ExecContext(ctx,
		`DELETE FROM issue_labels WHERE issue_id = ? AND label_id = ?`,
		issueID, labelID,
	)
	if err != nil {
		return fmt.Errorf("unlinking label from issue: %w", err)
	}
	return nil
}

// GetLabelsForIssue returns all labels for an issue, ordered by name.
func (s *Service) GetLabelsForIssue(ctx context.Context, issueID string) ([]*domain.Label, error) {
	rows, err := s.store.DB.QueryContext(ctx,
		`SELECT l.id, l.company_id, l.name, l.color, l.created_at, l.updated_at
		 FROM labels l
		 INNER JOIN issue_labels il ON l.id = il.label_id
		 WHERE il.issue_id = ? ORDER BY l.name ASC`,
		issueID,
	)
	if err != nil {
		return nil, fmt.Errorf("getting labels for issue: %w", err)
	}
	defer rows.Close()

	out := make([]*domain.Label, 0)
	for rows.Next() {
		label, err := scanLabel(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, label)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating labels: %w", err)
	}
	return out, nil
}

// issueExists checks if an issue exists.
func (s *Service) issueExists(ctx context.Context, issueID string) error {
	var exists sql.NullString
	err := s.store.DB.QueryRowContext(ctx,
		`SELECT id FROM issues WHERE id = ? LIMIT 1`,
		issueID,
	).Scan(&exists)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrIssueNotFound
		}
		return fmt.Errorf("checking issue exists: %w", err)
	}
	return nil
}

// scanner is satisfied by both *sql.Row and *sql.Rows.
type scanner interface {
	Scan(dest ...any) error
}

func scanLabel(s scanner) (*domain.Label, error) {
	var label domain.Label
	var createdAt, updatedAt string

	if err := s.Scan(&label.ID, &label.CompanyID, &label.Name, &label.Color, &createdAt, &updatedAt); err != nil {
		return nil, err
	}

	var err error
	label.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return nil, fmt.Errorf("parsing created_at %q: %w", createdAt, err)
	}
	label.UpdatedAt, err = time.Parse(time.RFC3339, updatedAt)
	if err != nil {
		return nil, fmt.Errorf("parsing updated_at %q: %w", updatedAt, err)
	}

	return &label, nil
}

