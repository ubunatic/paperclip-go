// Package agents provides CRUD operations for Agent entities.
package agents

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/ubunatic/paperclip-go/internal/activity"
	"github.com/ubunatic/paperclip-go/internal/domain"
	"github.com/ubunatic/paperclip-go/internal/ids"
	"github.com/ubunatic/paperclip-go/internal/store"
)

// ErrNotFound is returned when a requested agent does not exist.
var ErrNotFound = errors.New("agent not found")

// ErrHasActiveCheckout is returned when attempting to delete an agent with active checkouts.
var ErrHasActiveCheckout = errors.New("agent has active checkouts")

// ErrInvalidRuntimeState is returned when an invalid runtime state is provided.
var ErrInvalidRuntimeState = errors.New("invalid runtime state")

// ErrInvalidTransition is returned when a runtime state transition is not allowed.
var ErrInvalidTransition = errors.New("invalid state transition")

// Service provides agent CRUD backed by the store.
type Service struct {
	store *store.Store
	log   *activity.Log
}

// New returns a Service using the given store and activity log.
func New(s *store.Store, log *activity.Log) *Service {
	return &Service{store: s, log: log}
}

// Create inserts a new agent and returns the created entity.
func (s *Service) Create(ctx context.Context, companyID, shortname, displayName, role string, reportsTo *string, adapter string) (*domain.Agent, error) {
	now := time.Now().UTC().Truncate(time.Second)
	ts := now.Format(time.RFC3339)
	a := &domain.Agent{
		ID:           ids.NewUUID(),
		CompanyID:    companyID,
		Shortname:    shortname,
		DisplayName:  displayName,
		Role:         role,
		ReportsTo:    reportsTo,
		Adapter:      adapter,
		RuntimeState: "idle",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	_, err := s.store.DB.ExecContext(ctx,
		`INSERT INTO agents(id, company_id, shortname, display_name, role, reports_to, adapter, runtime_state, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		a.ID, a.CompanyID, a.Shortname, a.DisplayName, a.Role, a.ReportsTo, a.Adapter, a.RuntimeState, ts, ts,
	)
	if err != nil {
		return nil, fmt.Errorf("inserting agent: %w", err)
	}
	return a, nil
}

// Get returns the agent with the given ID, or ErrNotFound if it doesn't exist.
func (s *Service) Get(ctx context.Context, id string) (*domain.Agent, error) {
	row := s.store.DB.QueryRowContext(ctx,
		`SELECT id, company_id, shortname, display_name, role, reports_to, adapter, runtime_state, created_at, updated_at
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
		`SELECT id, company_id, shortname, display_name, role, reports_to, adapter, runtime_state, created_at, updated_at
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
		`SELECT id, company_id, shortname, display_name, role, reports_to, adapter, runtime_state, created_at, updated_at
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
		`SELECT id, company_id, shortname, display_name, role, reports_to, adapter, runtime_state, created_at, updated_at
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
	// Wrap in transaction for consistency and atomicity
	return s.store.WithTx(ctx, func(tx *sql.Tx) error {
		// Check if agent exists first (404 before 409)
		var exists sql.NullString
		err := tx.QueryRowContext(ctx,
			`SELECT id FROM agents WHERE id = ? LIMIT 1`,
			id,
		).Scan(&exists)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrNotFound
			}
			return fmt.Errorf("checking agent exists: %w", err)
		}

		// Check if agent has any dependents: issues (assignee or checked_out), comments, or heartbeat runs
		var assigneeCount, checkedOutCount, commentCount, heartbeatCount int
		err = tx.QueryRowContext(ctx,
			`SELECT (SELECT COUNT(*) FROM issues WHERE assignee_id = ?) as assignee_count, (SELECT COUNT(*) FROM issues WHERE checked_out_by = ?) as checked_out_count, (SELECT COUNT(*) FROM comments WHERE author_agent_id = ?) as comment_count, (SELECT COUNT(*) FROM heartbeat_runs WHERE agent_id = ?) as heartbeat_count`,
			id, id, id, id,
		).Scan(&assigneeCount, &checkedOutCount, &commentCount, &heartbeatCount)
		if err != nil {
			return fmt.Errorf("counting dependents: %w", err)
		}

		if assigneeCount > 0 || checkedOutCount > 0 || commentCount > 0 || heartbeatCount > 0 {
			return ErrHasActiveCheckout
		}

		// Delete the agent
		if _, err := tx.ExecContext(ctx,
			`DELETE FROM agents WHERE id = ?`,
			id,
		); err != nil {
			return fmt.Errorf("deleting agent: %w", err)
		}

		return nil
	})
}

// Update updates the displayName, role, and/or runtimeState of an agent.
// NOTE: This is an admin override that bypasses the state machine validation.
func (s *Service) Update(ctx context.Context, id string, displayName, role, runtimeState *string) (*domain.Agent, error) {
	now := time.Now().UTC().Truncate(time.Second)
	ts := now.Format(time.RFC3339)

	// Build the UPDATE query dynamically
	query := `UPDATE agents SET updated_at = ?`
	args := []interface{}{ts}

	if displayName != nil {
		query += `, display_name = ?`
		args = append(args, *displayName)
	}

	if role != nil {
		query += `, role = ?`
		args = append(args, *role)
	}

	if runtimeState != nil {
		if !domain.IsValidRuntimeState(*runtimeState) {
			return nil, ErrInvalidRuntimeState
		}
		query += `, runtime_state = ?`
		args = append(args, *runtimeState)
	}

	query += ` WHERE id = ?`
	args = append(args, id)

	_, err := s.store.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("updating agent: %w", err)
	}

	// RowsAffected can be 0 for legitimate no-op updates (same values, truncated timestamps),
	// so verify existence with a follow-up read instead of treating 0 as not found.
	agent, err := s.Get(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("fetching agent after update: %w", err)
	}
	return agent, nil
}

// Pause transitions an agent from idle or running to paused.
func (s *Service) Pause(ctx context.Context, agentID string) (*domain.Agent, error) {
	agent, err := s.Get(ctx, agentID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC().Truncate(time.Second)
	ts := now.Format(time.RFC3339)

	// Atomic conditional update: only succeeds if current state is idle or running
	result, err := s.store.DB.ExecContext(ctx,
		`UPDATE agents SET runtime_state = ?, updated_at = ? WHERE id = ? AND runtime_state IN ('idle', 'running')`,
		"paused", ts, agentID,
	)
	if err != nil {
		return nil, fmt.Errorf("pausing agent: %w", err)
	}

	n, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("checking rows affected: %w", err)
	}
	if n == 0 {
		// Either not found or invalid transition; check which
		_, err := s.Get(ctx, agentID)
		if err != nil {
			return nil, err
		}
		return nil, ErrInvalidTransition // agent exists but state transition not allowed
	}

	// Log the state transition
	metaJSON, _ := json.Marshal(map[string]string{
		"from": agent.RuntimeState,
		"to":   "paused",
	})
	if err := s.log.Record(ctx, agent.CompanyID, "system", "system", "pause", "agent", agentID, string(metaJSON)); err != nil {
		log.Printf("activity log error: %v\n", err)
	}

	// Fetch and return the updated agent
	return s.Get(ctx, agentID)
}

// Resume transitions an agent from paused to running.
func (s *Service) Resume(ctx context.Context, agentID string) (*domain.Agent, error) {
	agent, err := s.Get(ctx, agentID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC().Truncate(time.Second)
	ts := now.Format(time.RFC3339)

	// Atomic conditional update: only succeeds if current state is paused
	result, err := s.store.DB.ExecContext(ctx,
		`UPDATE agents SET runtime_state = ?, updated_at = ? WHERE id = ? AND runtime_state = 'paused'`,
		"running", ts, agentID,
	)
	if err != nil {
		return nil, fmt.Errorf("resuming agent: %w", err)
	}

	n, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("checking rows affected: %w", err)
	}
	if n == 0 {
		// Either not found or invalid transition; check which
		_, err := s.Get(ctx, agentID)
		if err != nil {
			return nil, err
		}
		return nil, ErrInvalidTransition // agent exists but state transition not allowed
	}

	// Log the state transition
	metaJSON, _ := json.Marshal(map[string]string{
		"from": agent.RuntimeState,
		"to":   "running",
	})
	if err := s.log.Record(ctx, agent.CompanyID, "system", "system", "resume", "agent", agentID, string(metaJSON)); err != nil {
		log.Printf("activity log error: %v\n", err)
	}

	// Fetch and return the updated agent
	return s.Get(ctx, agentID)
}

// Terminate transitions an agent to terminated (from idle, running, or paused states).
func (s *Service) Terminate(ctx context.Context, agentID string) (*domain.Agent, error) {
	agent, err := s.Get(ctx, agentID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC().Truncate(time.Second)
	ts := now.Format(time.RFC3339)

	// Atomic conditional update: only succeeds if current state is idle, running, or paused
	result, err := s.store.DB.ExecContext(ctx,
		`UPDATE agents SET runtime_state = ?, updated_at = ? WHERE id = ? AND runtime_state IN ('idle', 'running', 'paused')`,
		"terminated", ts, agentID,
	)
	if err != nil {
		return nil, fmt.Errorf("terminating agent: %w", err)
	}

	n, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("checking rows affected: %w", err)
	}
	if n == 0 {
		// Either not found or invalid transition; check which
		_, err := s.Get(ctx, agentID)
		if err != nil {
			return nil, err
		}
		return nil, ErrInvalidTransition // agent exists but state transition not allowed
	}

	// Log the state transition
	metaJSON, _ := json.Marshal(map[string]string{
		"from": agent.RuntimeState,
		"to":   "terminated",
	})
	if err := s.log.Record(ctx, agent.CompanyID, "system", "system", "terminate", "agent", agentID, string(metaJSON)); err != nil {
		log.Printf("activity log error: %v\n", err)
	}

	// Fetch and return the updated agent
	return s.Get(ctx, agentID)
}

// scanner is satisfied by both *sql.Row and *sql.Rows.
type scanner interface {
	Scan(dest ...any) error
}

func scanAgent(s scanner) (*domain.Agent, error) {
	var a domain.Agent
	var createdAt, updatedAt string
	if err := s.Scan(&a.ID, &a.CompanyID, &a.Shortname, &a.DisplayName, &a.Role, &a.ReportsTo, &a.Adapter, &a.RuntimeState, &createdAt, &updatedAt); err != nil {
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
