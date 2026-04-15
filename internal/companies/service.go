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
	now := time.Now().UTC()
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
	return out, rows.Err()
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
