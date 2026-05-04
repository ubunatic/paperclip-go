// Package routines provides CRUD operations for Routine entities.
package routines

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

var (
	ErrNotFound     = errors.New("routine not found")
	ErrInvalidCron  = errors.New("invalid cron expression")
	ErrNameConflict = errors.New("routine name already exists for this company")
)

// Service provides routine CRUD backed by the store.
type Service struct {
	store *store.Store
	now   func() time.Time
}

// New returns a Service using the given store.
func New(s *store.Store) *Service {
	return NewWithClock(s, func() time.Time { return time.Now().UTC() })
}

// NewWithClock returns a Service with a custom clock function for testing.
func NewWithClock(s *store.Store, now func() time.Time) *Service {
	return &Service{store: s, now: now}
}

// Create inserts a new routine and returns the created entity.
func (s *Service) Create(ctx context.Context, companyID, agentID, name, cronExpr string) (*domain.Routine, error) {
	// Validate cronExpr
	if err := s.validateCronExpr(cronExpr); err != nil {
		return nil, err
	}

	now := s.now()
	ts := now.Format(time.RFC3339)

	routine := &domain.Routine{
		ID:        ids.NewUUID(),
		CompanyID: companyID,
		AgentID:   agentID,
		Name:      name,
		CronExpr:  cronExpr,
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	_, err := s.store.DB.ExecContext(ctx,
		`INSERT INTO routines(id, company_id, agent_id, name, cron_expr, enabled, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		routine.ID, routine.CompanyID, routine.AgentID, routine.Name, routine.CronExpr, routine.Enabled, ts, ts,
	)
	if err != nil {
		// Check for UNIQUE constraint violation on (company_id, name)
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return nil, ErrNameConflict
		}
		return nil, fmt.Errorf("create routine: %w", err)
	}

	return routine, nil
}

// GetByID retrieves a routine by its ID.
func (s *Service) GetByID(ctx context.Context, id string) (*domain.Routine, error) {
	row := s.store.DB.QueryRowContext(ctx,
		`SELECT id, company_id, agent_id, name, cron_expr, enabled, last_run_at, dispatch_fingerprint, created_at, updated_at
		 FROM routines WHERE id = ?`,
		id,
	)
	routine, err := scanRoutine(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get routine: %w", err)
	}
	return routine, nil
}

// ListByCompany lists all routines for a company.
func (s *Service) ListByCompany(ctx context.Context, companyID string) ([]*domain.Routine, error) {
	rows, err := s.store.DB.QueryContext(ctx,
		`SELECT id, company_id, agent_id, name, cron_expr, enabled, last_run_at, dispatch_fingerprint, created_at, updated_at
		 FROM routines WHERE company_id = ? ORDER BY created_at`,
		companyID,
	)
	if err != nil {
		return nil, fmt.Errorf("list routines: %w", err)
	}
	defer rows.Close()

	var routines []*domain.Routine
	for rows.Next() {
		r, err := scanRoutine(rows)
		if err != nil {
			return nil, fmt.Errorf("scan routine: %w", err)
		}
		routines = append(routines, r)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return routines, nil
}

type UpdateInput struct {
	Name     *string
	CronExpr *string
	Enabled  *bool
}

// Update patches a routine (name, cronExpr, enabled).
func (s *Service) Update(ctx context.Context, id string, patch UpdateInput) (*domain.Routine, error) {
	// Guard: if patch is empty, return current routine unchanged
	if patch.Name == nil && patch.CronExpr == nil && patch.Enabled == nil {
		return s.GetByID(ctx, id)
	}

	// Get current routine
	current, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Prepare update fields
	now := s.now()
	ts := now.Format(time.RFC3339)

	// Apply patches
	if patch.Name != nil {
		current.Name = *patch.Name
	}
	if patch.CronExpr != nil {
		// Validate new cron expression
		if err := s.validateCronExpr(*patch.CronExpr); err != nil {
			return nil, err
		}
		current.CronExpr = *patch.CronExpr
	}
	if patch.Enabled != nil {
		current.Enabled = *patch.Enabled
	}
	current.UpdatedAt = now

	// Build dynamic update query
	query := "UPDATE routines SET "
	var args []interface{}

	if patch.Name != nil {
		query += "name = ?, "
		args = append(args, *patch.Name)
	}
	if patch.CronExpr != nil {
		query += "cron_expr = ?, "
		args = append(args, *patch.CronExpr)
	}
	if patch.Enabled != nil {
		enabled := 0
		if *patch.Enabled {
			enabled = 1
		}
		query += "enabled = ?, "
		args = append(args, enabled)
	}

	query += "updated_at = ? WHERE id = ?"
	args = append(args, ts, id)

	_, err = s.store.DB.ExecContext(ctx, query, args...)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return nil, ErrNameConflict
		}
		return nil, fmt.Errorf("update routine: %w", err)
	}

	return current, nil
}

// Delete removes a routine.
func (s *Service) Delete(ctx context.Context, id string) error {
	result, err := s.store.DB.ExecContext(ctx,
		`DELETE FROM routines WHERE id = ?`,
		id,
	)
	if err != nil {
		return fmt.Errorf("delete routine: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// Trigger manually fires the routine (sets last_run_at = now).
func (s *Service) Trigger(ctx context.Context, id string) (*domain.Routine, error) {
	now := s.now()
	ts := now.Format(time.RFC3339)

	result, err := s.store.DB.ExecContext(ctx,
		`UPDATE routines SET last_run_at = ?, updated_at = ? WHERE id = ?`,
		ts, ts, id,
	)
	if err != nil {
		return nil, fmt.Errorf("trigger routine: %w", err)
	}

	// Check rows affected first
	if n, _ := result.RowsAffected(); n == 0 {
		return nil, ErrNotFound
	}

	// Now fetch the updated routine
	return s.GetByID(ctx, id)
}

// DueRoutines returns routines that are enabled and due to run.
func (s *Service) DueRoutines(ctx context.Context, now time.Time) ([]*domain.Routine, error) {
	rows, err := s.store.DB.QueryContext(ctx,
		`SELECT id, company_id, agent_id, name, cron_expr, enabled, last_run_at, dispatch_fingerprint, created_at, updated_at
		 FROM routines
		 WHERE enabled = 1 AND (last_run_at IS NULL OR last_run_at < ?)
		 ORDER BY created_at`,
		now.Format(time.RFC3339),
	)
	if err != nil {
		return nil, fmt.Errorf("due routines: %w", err)
	}
	defer rows.Close()

	var routines []*domain.Routine
	truncatedNow := now.Truncate(time.Minute)

	for rows.Next() {
		r, err := scanRoutine(rows)
		if err != nil {
			return nil, fmt.Errorf("scan routine: %w", err)
		}

		// Check if routine is due
		due, err := IsDue(r.CronExpr, truncatedNow)
		if err != nil {
			// Skip routines with invalid cron expressions (should not happen in production).
			continue
		}

		if due {
			routines = append(routines, r)
		}
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return routines, nil
}

// MarkDispatched atomically sets dispatch_fingerprint and last_run_at.
// Returns true if this routine was marked (not already dispatched), false if another process beat us.
func (s *Service) MarkDispatched(ctx context.Context, id, fingerprint string, runAt time.Time) (bool, error) {
	ts := runAt.Format(time.RFC3339)

	result, err := s.store.DB.ExecContext(ctx,
		`UPDATE routines SET dispatch_fingerprint = ?, last_run_at = ?, updated_at = ?
		 WHERE id = ? AND dispatch_fingerprint IS NULL`,
		fingerprint, ts, ts, id,
	)
	if err != nil {
		return false, fmt.Errorf("mark dispatched: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("rows affected: %w", err)
	}

	return rowsAffected == 1, nil
}

// ClearDispatched resets the dispatch fingerprint to allow the routine to fire again.
func (s *Service) ClearDispatched(ctx context.Context, id string) error {
	result, err := s.store.DB.ExecContext(ctx,
		`UPDATE routines SET dispatch_fingerprint = NULL WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// validateCronExpr validates a single routine's cron expression.
func (s *Service) validateCronExpr(expr string) error {
	_, err := IsDue(expr, s.now())
	if err != nil {
		return ErrInvalidCron
	}
	return nil
}

// scanner is an interface for *sql.Row or *sql.Rows.
type scanner interface {
	Scan(dest ...interface{}) error
}

// scanRoutine scans a routine from a row or rows.
func scanRoutine(s scanner) (*domain.Routine, error) {
	var routine domain.Routine
	var enabled int
	var lastRunAt, createdAt, updatedAt *string

	if err := s.Scan(
		&routine.ID, &routine.CompanyID, &routine.AgentID, &routine.Name, &routine.CronExpr,
		&enabled, &lastRunAt, &routine.DispatchFingerprint, &createdAt, &updatedAt,
	); err != nil {
		return nil, err
	}

	routine.Enabled = enabled != 0

	// Parse created_at
	if createdAt == nil {
		return nil, fmt.Errorf("created_at is required")
	}
	var err error
	routine.CreatedAt, err = time.Parse(time.RFC3339, *createdAt)
	if err != nil {
		return nil, fmt.Errorf("parsing created_at %q: %w", *createdAt, err)
	}

	// Parse updated_at
	if updatedAt == nil {
		return nil, fmt.Errorf("updated_at is required")
	}
	routine.UpdatedAt, err = time.Parse(time.RFC3339, *updatedAt)
	if err != nil {
		return nil, fmt.Errorf("parsing updated_at %q: %w", *updatedAt, err)
	}

	// Parse last_run_at if present
	if lastRunAt != nil {
		routine.LastRunAt, err = stringToTimePtr(*lastRunAt)
		if err != nil {
			return nil, fmt.Errorf("parsing last_run_at %q: %w", *lastRunAt, err)
		}
	}

	return &routine, nil
}

// stringToTimePtr parses an RFC3339 string and returns a pointer to the parsed time.
func stringToTimePtr(s string) (*time.Time, error) {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil, err
	}
	return &t, nil
}
