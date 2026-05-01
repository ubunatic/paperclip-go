// Package secrets provides CRUD operations for Secret entities.
package secrets

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ubunatic/paperclip-go/internal/domain"
	"github.com/ubunatic/paperclip-go/internal/ids"
	"github.com/ubunatic/paperclip-go/internal/store"
	"modernc.org/sqlite"
)

// ErrNotFound is returned when a requested secret does not exist.
var ErrNotFound = errors.New("secret not found")

// ErrDuplicate is returned when attempting to create a secret with a duplicate name within a company.
var ErrDuplicate = errors.New("secret name already exists in this company")

// SQLite extended error code for unique constraint violations.
const (
	sqliteConstraintUnique = 2067 // SQLITE_CONSTRAINT_UNIQUE
)

// Service provides secret CRUD backed by the store.
type Service struct {
	store *store.Store
}

// New returns a Service using the given store.
func New(s *store.Store) *Service {
	return &Service{store: s}
}

// Create inserts a new secret and returns the created entity.
// Returns ErrDuplicate if a secret with the same name already exists for this company.
func (s *Service) Create(ctx context.Context, companyID, name, value string) (*domain.Secret, error) {
	now := time.Now().UTC().Truncate(time.Second)
	ts := now.Format(time.RFC3339)
	secret := &domain.Secret{
		ID:        ids.NewUUID(),
		CompanyID: companyID,
		Name:      name,
		Value:     value,
		CreatedAt: now,
		UpdatedAt: now,
	}
	_, err := s.store.DB.ExecContext(ctx,
		`INSERT INTO secrets(id, company_id, name, value, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		secret.ID, secret.CompanyID, secret.Name, secret.Value, ts, ts,
	)
	if err != nil {
		var sqliteErr *sqlite.Error
		if errors.As(err, &sqliteErr) && sqliteErr.Code() == sqliteConstraintUnique {
			return nil, ErrDuplicate
		}
		return nil, fmt.Errorf("inserting secret: %w", err)
	}
	return secret, nil
}

// GetByID returns the secret with the given ID, or ErrNotFound if it doesn't exist.
func (s *Service) GetByID(ctx context.Context, id string) (*domain.Secret, error) {
	row := s.store.DB.QueryRowContext(ctx,
		`SELECT id, company_id, name, value, created_at, updated_at
		 FROM secrets WHERE id = ?`, id,
	)
	secret, err := scanSecret(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return secret, err
}

// ListByCompany returns all secrets for a given company (without values), ordered by creation time.
func (s *Service) ListByCompany(ctx context.Context, companyID string) ([]*domain.SecretSummary, error) {
	rows, err := s.store.DB.QueryContext(ctx,
		`SELECT id, company_id, name, created_at, updated_at
		 FROM secrets WHERE company_id = ? ORDER BY created_at ASC`,
		companyID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing secrets by company: %w", err)
	}
	defer rows.Close()

	out := make([]*domain.SecretSummary, 0)
	for rows.Next() {
		summary, err := scanSecretSummary(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, summary)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating secrets: %w", err)
	}
	return out, nil
}

// Update updates name and/or value of a secret.
// Returns ErrNotFound if the secret does not exist.
// Returns ErrDuplicate if the new name already exists for this company.
func (s *Service) Update(ctx context.Context, id string, name, value *string) (*domain.Secret, error) {
	// Validate that at least one field is being updated
	if name == nil && value == nil {
		return nil, fmt.Errorf("update requires at least one field to be provided")
	}

	now := time.Now().UTC().Truncate(time.Second)
	ts := now.Format(time.RFC3339)

	// Build the UPDATE query dynamically
	query := `UPDATE secrets SET updated_at = ?`
	args := []interface{}{ts}

	if name != nil {
		query += `, name = ?`
		args = append(args, *name)
	}

	if value != nil {
		query += `, value = ?`
		args = append(args, *value)
	}

	query += ` WHERE id = ?`
	args = append(args, id)

	result, err := s.store.DB.ExecContext(ctx, query, args...)
	if err != nil {
		var sqliteErr *sqlite.Error
		if errors.As(err, &sqliteErr) && sqliteErr.Code() == sqliteConstraintUnique {
			return nil, ErrDuplicate
		}
		return nil, fmt.Errorf("updating secret: %w", err)
	}

	// Check if the secret exists via RowsAffected
	n, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("getting rows affected for secret update: %w", err)
	}
	if n == 0 {
		// RowsAffected may be 0 for a no-op update in SQLite even when the row exists.
		// Run a lightweight existence check to distinguish not-found from no-op.
		var exists int
		err := s.store.DB.QueryRowContext(ctx,
			`SELECT 1 FROM secrets WHERE id = ? LIMIT 1`,
			id,
		).Scan(&exists)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, ErrNotFound
			}
			return nil, fmt.Errorf("checking secret exists after update: %w", err)
		}
	}

	// Fetch and return the updated secret
	return s.GetByID(ctx, id)
}

// Delete deletes a secret.
// Returns ErrNotFound if the secret does not exist.
func (s *Service) Delete(ctx context.Context, id string) error {
	result, err := s.store.DB.ExecContext(ctx,
		`DELETE FROM secrets WHERE id = ?`,
		id,
	)
	if err != nil {
		return fmt.Errorf("deleting secret: %w", err)
	}

	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("getting rows affected for secret delete: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// scanner is satisfied by both *sql.Row and *sql.Rows.
type scanner interface {
	Scan(dest ...any) error
}

func scanSecret(s scanner) (*domain.Secret, error) {
	var secret domain.Secret
	var createdAt, updatedAt string
	if err := s.Scan(&secret.ID, &secret.CompanyID, &secret.Name, &secret.Value, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	var err error
	secret.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return nil, fmt.Errorf("parsing created_at %q: %w", createdAt, err)
	}
	secret.UpdatedAt, err = time.Parse(time.RFC3339, updatedAt)
	if err != nil {
		return nil, fmt.Errorf("parsing updated_at %q: %w", updatedAt, err)
	}
	return &secret, nil
}

func scanSecretSummary(s scanner) (*domain.SecretSummary, error) {
	var summary domain.SecretSummary
	var createdAt, updatedAt string
	if err := s.Scan(&summary.ID, &summary.CompanyID, &summary.Name, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	var err error
	summary.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return nil, fmt.Errorf("parsing created_at %q: %w", createdAt, err)
	}
	summary.UpdatedAt, err = time.Parse(time.RFC3339, updatedAt)
	if err != nil {
		return nil, fmt.Errorf("parsing updated_at %q: %w", updatedAt, err)
	}
	return &summary, nil
}
