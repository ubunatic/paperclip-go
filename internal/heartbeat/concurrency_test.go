package heartbeat_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ubunatic/paperclip-go/internal/activity"
	"github.com/ubunatic/paperclip-go/internal/agents"
	"github.com/ubunatic/paperclip-go/internal/comments"
	"github.com/ubunatic/paperclip-go/internal/companies"
	"github.com/ubunatic/paperclip-go/internal/domain"
	"github.com/ubunatic/paperclip-go/internal/heartbeat"
	"github.com/ubunatic/paperclip-go/internal/ids"
	"github.com/ubunatic/paperclip-go/internal/issues"
	"github.com/ubunatic/paperclip-go/internal/testutil"
)

// TestConcurrencyLimitRaceCondition fires two Run() calls simultaneously for the
// same agent and verifies that exactly one succeeds and one returns ErrAlreadyRunning.
// The race detector will catch any data race in the in-flight check.
func TestConcurrencyLimitRaceCondition(t *testing.T) {
	s := testutil.NewStore(t)
	ctx := context.Background()

	companySvc := companies.New(s)
	company, err := companySvc.Create(ctx, "Race Corp", "race-corp", "Race test")
	if err != nil {
		t.Fatalf("Create company: %v", err)
	}

	agentSvc := agents.New(s, activity.New(s))
	agent, err := agentSvc.Create(ctx, company.ID, "racer", "Racer", "agent", nil, "slow-stub")
	if err != nil {
		t.Fatalf("Create agent: %v", err)
	}

	// A slow adapter that holds the run in "running" state briefly so the second
	// goroutine can see it in the DB.
	slowAdapter := heartbeat.NewMockAdapter(func(a *domain.Agent, i *domain.Issue) (*domain.RunResult, error) {
		time.Sleep(50 * time.Millisecond)
		return &domain.RunResult{Status: "success", Summary: "done"}, nil
	})
	registry := heartbeat.NewRegistry()
	registry.Register("slow-stub", slowAdapter)

	issueSvc := issues.New(s)
	commentSvc := comments.New(s)
	actLog := activity.New(s)
	runner := heartbeat.New(s, agentSvc, issueSvc, commentSvc, actLog, registry, nil)

	var (
		wg       sync.WaitGroup
		successes atomic.Int32
		conflicts atomic.Int32
	)

	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := runner.Run(ctx, agent.ID)
			if err == nil {
				successes.Add(1)
			} else if errors.Is(err, heartbeat.ErrAlreadyRunning) {
				conflicts.Add(1)
			} else {
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}
	wg.Wait()

	if successes.Load() != 1 {
		t.Errorf("successes = %d, want 1", successes.Load())
	}
	if conflicts.Load() != 1 {
		t.Errorf("conflicts = %d, want 1", conflicts.Load())
	}
}

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
