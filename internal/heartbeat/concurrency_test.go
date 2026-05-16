package heartbeat_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ubunatic/paperclip-go/internal/activity"
	"github.com/ubunatic/paperclip-go/internal/agents"
	"github.com/ubunatic/paperclip-go/internal/comments"
	"github.com/ubunatic/paperclip-go/internal/companies"
	"github.com/ubunatic/paperclip-go/internal/heartbeat"
	"github.com/ubunatic/paperclip-go/internal/ids"
	"github.com/ubunatic/paperclip-go/internal/issues"
	"github.com/ubunatic/paperclip-go/internal/testutil"
)

func TestConcurrencyLimitBlocksSecondRun(t *testing.T) {
	s := testutil.NewStore(t)
	ctx := context.Background()

	companySvc := companies.New(s)
	company, err := companySvc.Create(ctx, "Test Corp", "test", "Test company")
	if err != nil {
		t.Fatalf("Create company: %v", err)
	}

	agentSvc := agents.New(s, activity.New(s))
	agent, err := agentSvc.Create(ctx, company.ID, "alice", "Alice", "agent", nil, "stub")
	if err != nil {
		t.Fatalf("Create agent: %v", err)
	}

	// Insert a heartbeat run with status="running" directly into the DB.
	now := time.Now().UTC().Truncate(time.Second).Format(time.RFC3339)
	_, err = s.DB.ExecContext(ctx,
		`INSERT INTO heartbeat_runs(id, agent_id, issue_id, status, started_at) VALUES (?, ?, NULL, 'running', ?)`,
		ids.NewUUID(), agent.ID, now,
	)
	if err != nil {
		t.Fatalf("Insert in-flight run: %v", err)
	}

	actLog := activity.New(s)
	commentSvc := comments.New(s)
	issueSvc := issues.New(s)
	registry := heartbeat.NewDefaultRegistry()
	runner := heartbeat.New(s, agentSvc, issueSvc, commentSvc, actLog, registry, nil)

	_, err = runner.Run(ctx, agent.ID)
	if !errors.Is(err, heartbeat.ErrAlreadyRunning) {
		t.Errorf("expected ErrAlreadyRunning, got %v", err)
	}
}

func TestConcurrencyLimitAllowsRunAfterCompletion(t *testing.T) {
	s := testutil.NewStore(t)
	ctx := context.Background()

	companySvc := companies.New(s)
	company, err := companySvc.Create(ctx, "Test Corp", "test", "Test company")
	if err != nil {
		t.Fatalf("Create company: %v", err)
	}

	agentSvc := agents.New(s, activity.New(s))
	agent, err := agentSvc.Create(ctx, company.ID, "alice", "Alice", "agent", nil, "stub")
	if err != nil {
		t.Fatalf("Create agent: %v", err)
	}

	// Insert a heartbeat run with status="success" — not in-flight.
	now := time.Now().UTC().Truncate(time.Second).Format(time.RFC3339)
	_, err = s.DB.ExecContext(ctx,
		`INSERT INTO heartbeat_runs(id, agent_id, issue_id, status, started_at, finished_at) VALUES (?, ?, NULL, 'success', ?, ?)`,
		ids.NewUUID(), agent.ID, now, now,
	)
	if err != nil {
		t.Fatalf("Insert completed run: %v", err)
	}

	actLog := activity.New(s)
	commentSvc := comments.New(s)
	issueSvc := issues.New(s)
	registry := heartbeat.NewDefaultRegistry()
	runner := heartbeat.New(s, agentSvc, issueSvc, commentSvc, actLog, registry, nil)

	run, err := runner.Run(ctx, agent.ID)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if run.Status != "success" {
		t.Errorf("Status = %q, want %q", run.Status, "success")
	}
}
