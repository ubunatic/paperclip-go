// Package agents provides CRUD operations for Agent entities.
package agents

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

// ErrNotFound is returned when a requested agent does not exist.
var ErrNotFound = errors.New("agent not found")

// ErrHasActiveCheckout is returned when attempting to delete an agent with active checkouts.
var ErrHasActiveCheckout = errors.New("agent has active checkouts")

// Service provides agent CRUD backed by the store.
type Service struct {
	store *store.Store
}

// New returns a Service using the given store.
func New(s *store.Store) *Service {
	return &Service{store: s}
}

// Create inserts a new agent and returns the created entity.
func (s *Service) Create(ctx context.Context, companyID, shortname, displayName, role string, reportsTo *string, adapter string) (*domain.Agent, error) {
	now := time.Now().UTC().Truncate(time.Second)
	ts := now.Format(time.RFC3339)
	a := &domain.Agent{
		ID:          ids.NewUUID(),
		CompanyID:   companyID,
		Shortname:   shortname,
		DisplayName: displayName,
		Role:        role,
		ReportsTo:   reportsTo,
		Adapter:     adapter,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	_, err := s.store.DB.ExecContext(ctx,
		`INSERT INTO agents(id, company_id, shortname, display_name, role, reports_to, adapter, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		a.ID, a.CompanyID, a.Shortname, a.DisplayName, a.Role, a.ReportsTo, a.Adapter, ts, ts,
	)
	if err != nil {
		return nil, fmt.Errorf("inserting agent: %w", err)
	}
	return a, nil
}

// Get returns the agent with the given ID, or ErrNotFound if it doesn't exist.
func (s *Service) Get(ctx context.Context, id string) (*domain.Agent, error) {
	row := s.store.DB.QueryRowContext(ctx,
		`SELECT id, company_id, shortname, display_name, role, reports_to, adapter, created_at, updated_at
		 FROM agents WHERE id = ?`, id,
	)
	a, err := scanAgent(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return a, err
}

// List returns all agents ordered by creation time.
func (s *Service) List(ctx context.Context) ([]*domain.Agent, error) {
	rows, err := s.store.DB.QueryContext(ctx,
		`SELECT id, company_id, shortname, display_name, role, reports_to, adapter, created_at, updated_at
		 FROM agents ORDER BY created_at`,
	)
	if err != nil {
		return nil, fmt.Errorf("listing agents: %w", err)
	}
	defer rows.Close()

	out := make([]*domain.Agent, 0)
	for rows.Next() {
		a, err := scanAgent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating agents: %w", err)
	}
	return out, nil
}

// ListByCompany returns all agents for a given company, ordered by creation time.
func (s *Service) ListByCompany(ctx context.Context, companyID string) ([]*domain.Agent, error) {
	rows, err := s.store.DB.QueryContext(ctx,
		`SELECT id, company_id, shortname, display_name, role, reports_to, adapter, created_at, updated_at
		 FROM agents WHERE company_id = ? ORDER BY created_at`,
		companyID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing agents by company: %w", err)
	}
	defer rows.Close()

	out := make([]*domain.Agent, 0)
	for rows.Next() {
		a, err := scanAgent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating agents: %w", err)
	}
	return out, nil
}

// GetByShortname returns the agent with the given company ID and shortname, or ErrNotFound if it doesn't exist.
func (s *Service) GetByShortname(ctx context.Context, companyID, shortname string) (*domain.Agent, error) {
	row := s.store.DB.QueryRowContext(ctx,
		`SELECT id, company_id, shortname, display_name, role, reports_to, adapter, created_at, updated_at
		 FROM agents WHERE company_id = ? AND shortname = ?`, companyID, shortname,
	)
	a, err := scanAgent(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return a, err
}

// Delete deletes an agent if it has no active checkouts.
// Returns ErrNotFound if the agent does not exist.
// Returns ErrHasActiveCheckout if the agent has issues checked out with in_progress status.
func (s *Service) Delete(ctx context.Context, id string) error {
	// Check if agent has active checkouts
	var count int
	err := s.store.DB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM issues WHERE assignee_id = ? AND status = 'in_progress' AND checked_out_by IS NOT NULL`,
		id,
	).Scan(&count)
	if err != nil {
		return fmt.Errorf("counting active checkouts: %w", err)
	}

	if count > 0 {
		return ErrHasActiveCheckout
	}

	// Check if agent exists
	var exists sql.NullString
	err = s.store.DB.QueryRowContext(ctx,
		`SELECT id FROM agents WHERE id = ? LIMIT 1`,
		id,
	).Scan(&exists)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("checking agent exists: %w", err)
	}

	// Delete the agent
	if _, err := s.store.DB.ExecContext(ctx,
		`DELETE FROM agents WHERE id = ?`,
		id,
	); err != nil {
		return fmt.Errorf("deleting agent: %w", err)
	}

	return nil
}

// scanner is satisfied by both *sql.Row and *sql.Rows.
type scanner interface {
	Scan(dest ...any) error
}

func scanAgent(s scanner) (*domain.Agent, error) {
	var a domain.Agent
	var createdAt, updatedAt string
	if err := s.Scan(&a.ID, &a.CompanyID, &a.Shortname, &a.DisplayName, &a.Role, &a.ReportsTo, &a.Adapter, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	var err error
	a.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return nil, fmt.Errorf("parsing created_at %q: %w", createdAt, err)
	}
	a.UpdatedAt, err = time.Parse(time.RFC3339, updatedAt)
	if err != nil {
		return nil, fmt.Errorf("parsing updated_at %q: %w", updatedAt, err)
	}
	return &a, nil
}
