// Package heartbeat provides heartbeat execution and adapter infrastructure.
package heartbeat

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/ubunatic/paperclip-go/internal/activity"
	"github.com/ubunatic/paperclip-go/internal/agents"
	"github.com/ubunatic/paperclip-go/internal/comments"
	"github.com/ubunatic/paperclip-go/internal/domain"
	"github.com/ubunatic/paperclip-go/internal/ids"
	"github.com/ubunatic/paperclip-go/internal/issues"
	"github.com/ubunatic/paperclip-go/internal/store"
)

// ErrNotFound is returned when a heartbeat run is not found.
var ErrNotFound = errors.New("heartbeat run not found")

// Runner provides heartbeat run operations backed by the store.
type Runner struct {
	store       *store.Store
	agents      *agents.Service
	issues      *issues.Service
	comments    *comments.Service
	actLog      *activity.Log
	registry    *Registry
}

// New returns a Runner using the given dependencies.
func New(
	s *store.Store,
	agentSvc *agents.Service,
	issueSvc *issues.Service,
	commentSvc *comments.Service,
	actLog *activity.Log,
	registry *Registry,
) *Runner {
	return &Runner{
		store:    s,
		agents:   agentSvc,
		issues:   issueSvc,
		comments: commentSvc,
		actLog:   actLog,
		registry: registry,
	}
}

// Run executes a heartbeat for the given agent.
// It performs the following:
// 1. Fetches the agent by ID
// 2. Selects an issue to work on (if any are available)
// 3. Creates a HeartbeatRun record with status "running"
// 4. Looks up the agent's adapter in the registry
// 5. Calls the adapter's Run() method
// 6. Updates the HeartbeatRun with the result
// 7. Records activity for the successful run
// 8. Returns the completed HeartbeatRun
//
// If the agent is not found, returns an error.
func (r *Runner) Run(ctx context.Context, agentID string) (*domain.HeartbeatRun, error) {
	// Fetch the agent
	agent, err := r.agents.Get(ctx, agentID)
	if err != nil {
		if errors.Is(err, agents.ErrNotFound) {
			return nil, fmt.Errorf("agent not found: %w", err)
		}
		return nil, fmt.Errorf("fetching agent: %w", err)
	}

	// Select an issue to work on (first open issue)
	// This is a simple heuristic; heartbeat may pass nil issue
	var selectedIssue *domain.Issue
	issues, err := r.issues.ListWithFilters(ctx, agent.CompanyID, "open", nil, false)
	if err != nil {
		return nil, fmt.Errorf("listing issues: %w", err)
	}
	if len(issues) > 0 {
		selectedIssue = issues[0]
	}

	// Create the HeartbeatRun record with status "running"
	now := time.Now().UTC().Truncate(time.Second)
	startedAtStr := now.Format(time.RFC3339)

	run := &domain.HeartbeatRun{
		ID:        ids.NewUUID(),
		AgentID:   agentID,
		IssueID:   nil,
		Status:    "running",
		StartedAt: now,
	}

	if selectedIssue != nil {
		run.IssueID = &selectedIssue.ID
	}

	_, err = r.store.DB.ExecContext(ctx,
		`INSERT INTO heartbeat_runs(id, agent_id, issue_id, status, started_at)
		 VALUES (?, ?, ?, ?, ?)`,
		run.ID, run.AgentID, run.IssueID, run.Status, startedAtStr,
	)
	if err != nil {
		return nil, fmt.Errorf("creating heartbeat run: %w", err)
	}

	// Get the adapter from the registry
	adapterName := agent.Adapter
	if adapterName == "" {
		adapterName = "stub"
	}
	adapter := r.registry.Get(adapterName)
	if adapter == nil {
		adapter = r.registry.Get("stub")
	}

	// Check if adapter was found
	if adapter == nil {
		// Create run record with status="error"
		finishedAt := time.Now().UTC().Truncate(time.Second)
		finishedAtStr := finishedAt.Format(time.RFC3339)
		errMsg := fmt.Sprintf("heartbeat adapter %q not found", adapterName)

		_, err := r.store.DB.ExecContext(ctx,
			`UPDATE heartbeat_runs SET status = 'error', finished_at = ?, error = ?
			 WHERE id = ?`,
			finishedAtStr, errMsg, run.ID,
		)
		if err != nil {
			return nil, fmt.Errorf("updating heartbeat run with error: %w", err)
		}

		return nil, fmt.Errorf("heartbeat adapter %q not found", adapterName)
	}

	// Call the adapter's Run method
	result, err := adapter.Run(ctx, agent, selectedIssue)
	if err != nil {
		// Adapter returned an error; mark the run as failed
		finishedAt := time.Now().UTC().Truncate(time.Second)
		finishedAtStr := finishedAt.Format(time.RFC3339)
		errMsg := err.Error()

		_, err2 := r.store.DB.ExecContext(ctx,
			`UPDATE heartbeat_runs SET status = 'error', finished_at = ?, error = ?
			 WHERE id = ?`,
			finishedAtStr, errMsg, run.ID,
		)
		if err2 != nil {
			return nil, fmt.Errorf("updating heartbeat run with error: %w", err2)
		}

		return nil, err
	}

	// Success: update the run with the result
	finishedAt := time.Now().UTC().Truncate(time.Second)
	finishedAtStr := finishedAt.Format(time.RFC3339)

	_, err = r.store.DB.ExecContext(ctx,
		`UPDATE heartbeat_runs SET status = ?, finished_at = ?, summary = ?
		 WHERE id = ?`,
		result.Status, finishedAtStr, result.Summary, run.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("updating heartbeat run: %w", err)
	}

	// Record activity: heartbeat_run
	actErr := r.actLog.Record(ctx,
		agent.CompanyID, "system", agentID, "heartbeat_run", "heartbeat_run", run.ID,
		"{}",
	)
	if actErr != nil {
		// Log but don't fail; activity recording is best-effort
		log.Printf("warning: failed to record activity: %v", actErr)
	}

	// Post a comment on the issue if one was selected
	if r.comments != nil && selectedIssue != nil {
		commentErr := r.postHeartbeatComment(ctx, selectedIssue.ID, agent.ID, result.Summary)
		if commentErr != nil {
			// Log but don't fail; comment posting is best-effort
			log.Printf("warning: failed to post heartbeat comment: %v", commentErr)
		}
	}

	run.Status = result.Status
	run.FinishedAt = &finishedAt
	run.Summary = &result.Summary
	return run, nil
}

// postHeartbeatComment posts a comment on an issue from the heartbeat adapter.
func (r *Runner) postHeartbeatComment(ctx context.Context, issueID, agentID, summary string) error {
	_, err := r.comments.Create(ctx, issueID, &agentID, "agent", summary)
	if err != nil {
		return fmt.Errorf("posting heartbeat comment: %w", err)
	}
	return nil
}

// Create inserts a new heartbeat run and returns it.
// Only used in testing; production uses Run().
func (r *Runner) Create(ctx context.Context, agentID string, issueID *string, status string) (*domain.HeartbeatRun, error) {
	now := time.Now().UTC().Truncate(time.Second)
	ts := now.Format(time.RFC3339)

	run := &domain.HeartbeatRun{
		ID:        ids.NewUUID(),
		AgentID:   agentID,
		IssueID:   issueID,
		Status:    status,
		StartedAt: now,
	}

	_, err := r.store.DB.ExecContext(ctx,
		`INSERT INTO heartbeat_runs(id, agent_id, issue_id, status, started_at)
		 VALUES (?, ?, ?, ?, ?)`,
		run.ID, run.AgentID, run.IssueID, run.Status, ts,
	)
	if err != nil {
		return nil, fmt.Errorf("creating heartbeat run: %w", err)
	}

	return run, nil
}

// Update updates the status, summary, and error of a heartbeat run.
func (r *Runner) Update(ctx context.Context, id, status string, summary, errMsg *string) (*domain.HeartbeatRun, error) {
	finishedAt := time.Now().UTC().Truncate(time.Second)
	finishedAtStr := finishedAt.Format(time.RFC3339)

	_, err := r.store.DB.ExecContext(ctx,
		`UPDATE heartbeat_runs SET status = ?, finished_at = ?, summary = ?, error = ?
		 WHERE id = ?`,
		status, finishedAtStr, summary, errMsg, id,
	)
	if err != nil {
		return nil, fmt.Errorf("updating heartbeat run: %w", err)
	}

	return r.GetByID(ctx, id)
}

// GetByID returns the heartbeat run with the given ID, or ErrNotFound if it doesn't exist.
func (r *Runner) GetByID(ctx context.Context, id string) (*domain.HeartbeatRun, error) {
	row := r.store.DB.QueryRowContext(ctx,
		`SELECT id, agent_id, issue_id, status, started_at, finished_at, summary, error
		 FROM heartbeat_runs WHERE id = ?`,
		id,
	)

	run, err := scanHeartbeatRun(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return run, err
}

// ListByAgent returns all heartbeat runs for a given agent, ordered by started_at descending.
func (r *Runner) ListByAgent(ctx context.Context, agentID string) ([]*domain.HeartbeatRun, error) {
	rows, err := r.store.DB.QueryContext(ctx,
		`SELECT id, agent_id, issue_id, status, started_at, finished_at, summary, error
		 FROM heartbeat_runs WHERE agent_id = ? ORDER BY started_at DESC`,
		agentID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing heartbeat runs: %w", err)
	}
	defer rows.Close()

	out := make([]*domain.HeartbeatRun, 0)
	for rows.Next() {
		run, err := scanHeartbeatRun(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, run)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating heartbeat runs: %w", err)
	}
	return out, nil
}

// scanner is satisfied by both *sql.Row and *sql.Rows.
type scanner interface {
	Scan(dest ...any) error
}

func scanHeartbeatRun(s scanner) (*domain.HeartbeatRun, error) {
	var run domain.HeartbeatRun
	var startedAtStr string
	var finishedAtStr *string
	var issueID, summary, errMsg *string

	if err := s.Scan(&run.ID, &run.AgentID, &issueID, &run.Status, &startedAtStr, &finishedAtStr, &summary, &errMsg); err != nil {
		return nil, err
	}

	run.IssueID = issueID
	run.Summary = summary

	var err error
	run.StartedAt, err = time.Parse(time.RFC3339, startedAtStr)
	if err != nil {
		return nil, fmt.Errorf("parsing started_at %q: %w", startedAtStr, err)
	}

	if finishedAtStr != nil {
		parsedTime, err := time.Parse(time.RFC3339, *finishedAtStr)
		if err != nil {
			return nil, fmt.Errorf("parsing finished_at %q: %w", *finishedAtStr, err)
		}
		run.FinishedAt = &parsedTime
	}

	return &run, nil
}
