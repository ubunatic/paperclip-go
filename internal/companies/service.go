// Package companies provides CRUD operations for Company entities.
package companies

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

// ErrNotFound is returned when a requested company does not exist.
var ErrNotFound = errors.New("company not found")

// ErrHasDependents is returned when attempting to delete a company with agents or issues.
var ErrHasDependents = errors.New("company has dependent agents or issues")

// Service provides company CRUD backed by the store.
type Service struct {
	store *store.Store
}

// New returns a Service using the given store.
func New(s *store.Store) *Service {
	return &Service{store: s}
}

// Create inserts a new company and returns the created entity.
func (s *Service) Create(ctx context.Context, name, shortname, description string) (*domain.Company, error) {
	now := time.Now().UTC().Truncate(time.Second)
	ts := now.Format(time.RFC3339)
	c := &domain.Company{
		ID:          ids.NewUUID(),
		Name:        name,
		Shortname:   shortname,
		Description: description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	_, err := s.store.DB.ExecContext(ctx,
		`INSERT INTO companies(id, name, shortname, description, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		c.ID, c.Name, c.Shortname, c.Description, ts, ts,
	)
	if err != nil {
		return nil, fmt.Errorf("inserting company: %w", err)
	}
	return c, nil
}

// Get returns the company with the given ID, or ErrNotFound if it doesn't exist.
func (s *Service) Get(ctx context.Context, id string) (*domain.Company, error) {
	row := s.store.DB.QueryRowContext(ctx,
		`SELECT id, name, shortname, description, created_at, updated_at
		 FROM companies WHERE id = ?`, id,
	)
	c, err := scanCompany(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return c, err
}

// List returns all companies ordered by creation time.
func (s *Service) List(ctx context.Context) ([]*domain.Company, error) {
	rows, err := s.store.DB.QueryContext(ctx,
		`SELECT id, name, shortname, description, created_at, updated_at
		 FROM companies ORDER BY created_at`,
	)
	if err != nil {
		return nil, fmt.Errorf("listing companies: %w", err)
	}
	defer rows.Close()

	out := make([]*domain.Company, 0)
	for rows.Next() {
		c, err := scanCompany(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating companies: %w", err)
	}
	return out, nil
}

// GetByShortname returns the company with the given shortname, or ErrNotFound if it doesn't exist.
func (s *Service) GetByShortname(ctx context.Context, shortname string) (*domain.Company, error) {
	row := s.store.DB.QueryRowContext(ctx,
		`SELECT id, name, shortname, description, created_at, updated_at
		 FROM companies WHERE shortname = ?`, shortname,
	)
	c, err := scanCompany(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return c, err
}

// Update updates name and/or description of a company.
// Returns ErrNotFound if the company does not exist.
func (s *Service) Update(ctx context.Context, id string, name, description *string) (*domain.Company, error) {
	now := time.Now().UTC().Truncate(time.Second)
	ts := now.Format(time.RFC3339)

	// Build the UPDATE query dynamically
	query := `UPDATE companies SET updated_at = ?`
	args := []interface{}{ts}

	if name != nil {
		query += `, name = ?`
		args = append(args, *name)
	}

	if description != nil {
		query += `, description = ?`
		args = append(args, *description)
	}

	query += ` WHERE id = ?`
	args = append(args, id)

	result, err := s.store.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("updating company: %w", err)
	}

	// Check if the company exists via RowsAffected
	n, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("getting rows affected for company update: %w", err)
	}
	if n == 0 {
		return nil, ErrNotFound
	}

	// Fetch and return the updated company
	return s.Get(ctx, id)
}

// Delete deletes a company if it has no dependent agents or issues.
// Returns ErrNotFound if the company does not exist.
// Returns ErrHasDependents if the company has agents or issues.
func (s *Service) Delete(ctx context.Context, id string) error {
	// Wrap in transaction for atomicity
	return s.store.WithTx(ctx, func(tx *sql.Tx) error {
		// Check if company exists first (404 before 409)
		var exists sql.NullString
		err := tx.QueryRowContext(ctx,
			`SELECT id FROM companies WHERE id = ? LIMIT 1`,
			id,
		).Scan(&exists)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrNotFound
			}
			return fmt.Errorf("checking company exists: %w", err)
		}

		// Check if company has any dependent agents, issues, or activity logs in a single query
		var agentCount, issueCount, activityCount int
		err = tx.QueryRowContext(ctx,
			`SELECT (SELECT COUNT(*) FROM agents WHERE company_id = ?) as agent_count, (SELECT COUNT(*) FROM issues WHERE company_id = ?) as issue_count, (SELECT COUNT(*) FROM activity_log WHERE company_id = ?) as activity_count`,
			id, id, id,
		).Scan(&agentCount, &issueCount, &activityCount)
		if err != nil {
			return fmt.Errorf("counting dependents: %w", err)
		}

		if agentCount > 0 || issueCount > 0 || activityCount > 0 {
			return ErrHasDependents
		}

		// Delete the company
		if _, err := tx.ExecContext(ctx,
			`DELETE FROM companies WHERE id = ?`,
			id,
		); err != nil {
			return fmt.Errorf("deleting company: %w", err)
		}

		return nil
	})
}

// scanner is satisfied by both *sql.Row and *sql.Rows.
type scanner interface {
	Scan(dest ...any) error
}

func scanCompany(s scanner) (*domain.Company, error) {
	var c domain.Company
	var createdAt, updatedAt string
	if err := s.Scan(&c.ID, &c.Name, &c.Shortname, &c.Description, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	var err error
	c.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return nil, fmt.Errorf("parsing created_at %q: %w", createdAt, err)
	}
	c.UpdatedAt, err = time.Parse(time.RFC3339, updatedAt)
	if err != nil {
		return nil, fmt.Errorf("parsing updated_at %q: %w", updatedAt, err)
	}
	return &c, nil
}
