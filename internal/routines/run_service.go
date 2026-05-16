package routines

import (
	"context"
	"fmt"
	"time"

	"github.com/ubunatic/paperclip-go/internal/domain"
	"github.com/ubunatic/paperclip-go/internal/ids"
	"github.com/ubunatic/paperclip-go/internal/store"
)

// RunService records and retrieves routine run history.
type RunService struct {
	store *store.Store
}

// NewRunService returns a RunService backed by the given store.
func NewRunService(s *store.Store) *RunService {
	return &RunService{store: s}
}

// Record inserts a new routine_run row with status="dispatched" and started_at=now.
// Routines are fire-and-forget so "dispatched" is the terminal state for this implementation.
func (rs *RunService) Record(ctx context.Context, routineID, agentID string) (*domain.RoutineRun, error) {
	run := &domain.RoutineRun{
		ID:        ids.NewUUID(),
		RoutineID: routineID,
		AgentID:   agentID,
		Status:    "dispatched",
		StartedAt: time.Now().UTC(),
	}

	_, err := rs.store.DB.ExecContext(ctx,
		`INSERT INTO routine_runs(id, routine_id, agent_id, status, started_at)
		 VALUES (?, ?, ?, ?, ?)`,
		run.ID, run.RoutineID, run.AgentID, run.Status, run.StartedAt.Format(time.RFC3339),
	)
	if err != nil {
		return nil, fmt.Errorf("record routine run: %w", err)
	}

	return run, nil
}

// ListByRoutine returns all runs for a given routine, ordered by started_at DESC.
func (rs *RunService) ListByRoutine(ctx context.Context, routineID string) ([]*domain.RoutineRun, error) {
	rows, err := rs.store.DB.QueryContext(ctx,
		`SELECT id, routine_id, agent_id, status, started_at, finished_at, error
		 FROM routine_runs WHERE routine_id = ? ORDER BY started_at DESC`,
		routineID,
	)
	if err != nil {
		return nil, fmt.Errorf("list routine runs: %w", err)
	}
	defer rows.Close()

	runs := make([]*domain.RoutineRun, 0)
	for rows.Next() {
		r, err := scanRoutineRun(rows)
		if err != nil {
			return nil, fmt.Errorf("scan routine run: %w", err)
		}
		runs = append(runs, r)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return runs, nil
}

// scanRoutineRun reads a RoutineRun from a sql.Row or sql.Rows.
func scanRoutineRun(s scanner) (*domain.RoutineRun, error) {
	var run domain.RoutineRun
	var startedAt string
	var finishedAt, runErr *string

	if err := s.Scan(&run.ID, &run.RoutineID, &run.AgentID, &run.Status, &startedAt, &finishedAt, &runErr); err != nil {
		return nil, err
	}

	var err error
	run.StartedAt, err = time.Parse(time.RFC3339, startedAt)
	if err != nil {
		return nil, fmt.Errorf("parsing started_at %q: %w", startedAt, err)
	}

	if finishedAt != nil {
		t, err := time.Parse(time.RFC3339, *finishedAt)
		if err != nil {
			return nil, fmt.Errorf("parsing finished_at %q: %w", *finishedAt, err)
		}
		run.FinishedAt = &t
	}

	run.Error = runErr

	return &run, nil
}
