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
	"github.com/ubunatic/paperclip-go/internal/domain"
	"github.com/ubunatic/paperclip-go/internal/heartbeat"
	"github.com/ubunatic/paperclip-go/internal/issues"
	"github.com/ubunatic/paperclip-go/internal/testutil"
)

// newErrorAdapter creates a MockAdapter that always returns the given error.
func newErrorAdapter(err error) *heartbeat.MockAdapter {
	return heartbeat.NewMockAdapter(func(agent *domain.Agent, issue *domain.Issue) (*domain.RunResult, error) {
		return nil, err
	})
}

func TestRunnerCreate(t *testing.T) {
	s := testutil.NewStore(t)
	ctx := context.Background()

	// Create a company and agent
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

	// Create heartbeat runner
	actLog := activity.New(s)
	commentSvc := comments.New(s)
	registry := heartbeat.NewDefaultRegistry()
	runner := heartbeat.New(s, agentSvc, nil, commentSvc, actLog, registry, nil)

	// Create a heartbeat run
	run, err := runner.Create(ctx, agent.ID, nil, "running")
	if err != nil {
		t.Fatalf("Create run: %v", err)
	}
	if run.ID == "" {
		t.Fatal("expected non-empty run ID")
	}
	if run.AgentID != agent.ID {
		t.Errorf("AgentID = %q, want %q", run.AgentID, agent.ID)
	}
	if run.Status != "running" {
		t.Errorf("Status = %q, want %q", run.Status, "running")
	}
	if run.IssueID != nil {
		t.Errorf("IssueID = %v, want nil", run.IssueID)
	}
}

func TestRunnerGetByID(t *testing.T) {
	s := testutil.NewStore(t)
	ctx := context.Background()

	// Create a company and agent
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

	// Create heartbeat runner
	actLog := activity.New(s)
	commentSvc := comments.New(s)
	registry := heartbeat.NewDefaultRegistry()
	runner := heartbeat.New(s, agentSvc, nil, commentSvc, actLog, registry, nil)

	// Create and fetch a heartbeat run
	run, err := runner.Create(ctx, agent.ID, nil, "running")
	if err != nil {
		t.Fatalf("Create run: %v", err)
	}

	got, err := runner.GetByID(ctx, run.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.ID != run.ID {
		t.Errorf("ID = %q, want %q", got.ID, run.ID)
	}
	if got.AgentID != agent.ID {
		t.Errorf("AgentID = %q, want %q", got.AgentID, agent.ID)
	}
}

func TestRunnerGetByIDNotFound(t *testing.T) {
	s := testutil.NewStore(t)
	ctx := context.Background()

	actLog := activity.New(s)
	commentSvc := comments.New(s)
	agentSvc := agents.New(s, activity.New(s))
	registry := heartbeat.NewDefaultRegistry()
	runner := heartbeat.New(s, agentSvc, nil, commentSvc, actLog, registry, nil)

	_, err := runner.GetByID(ctx, "nonexistent-id")
	if !errors.Is(err, heartbeat.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestRunnerUpdate(t *testing.T) {
	s := testutil.NewStore(t)
	ctx := context.Background()

	// Create a company and agent
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

	// Create heartbeat runner
	actLog := activity.New(s)
	commentSvc := comments.New(s)
	registry := heartbeat.NewDefaultRegistry()
	runner := heartbeat.New(s, agentSvc, nil, commentSvc, actLog, registry, nil)

	// Create and update a heartbeat run
	run, err := runner.Create(ctx, agent.ID, nil, "running")
	if err != nil {
		t.Fatalf("Create run: %v", err)
	}

	summary := "Test summary"
	updated, err := runner.Update(ctx, run.ID, "success", &summary, nil)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Status != "success" {
		t.Errorf("Status = %q, want %q", updated.Status, "success")
	}
	if updated.Summary == nil || *updated.Summary != summary {
		t.Errorf("Summary = %v, want %q", updated.Summary, summary)
	}
	if updated.FinishedAt == nil {
		t.Error("FinishedAt should not be nil after update")
	}
}

func TestRunnerListByAgent(t *testing.T) {
	s := testutil.NewStore(t)
	ctx := context.Background()

	// Create a company and agent
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

	// Create heartbeat runner
	actLog := activity.New(s)
	commentSvc := comments.New(s)
	registry := heartbeat.NewDefaultRegistry()
	runner := heartbeat.New(s, agentSvc, nil, commentSvc, actLog, registry, nil)

	// Create multiple heartbeat runs
	run1, err := runner.Create(ctx, agent.ID, nil, "running")
	if err != nil {
		t.Fatalf("Create run 1: %v", err)
	}

	run2, err := runner.Create(ctx, agent.ID, nil, "success")
	if err != nil {
		t.Fatalf("Create run 2: %v", err)
	}

	// List by agent
	runs, err := runner.ListByAgent(ctx, agent.ID)
	if err != nil {
		t.Fatalf("ListByAgent: %v", err)
	}
	if len(runs) != 2 {
		t.Errorf("ListByAgent length = %d, want 2", len(runs))
	}
	// Just verify both runs are returned (ordering may depend on timing)
	ids := make(map[string]bool)
	for _, run := range runs {
		ids[run.ID] = true
	}
	if !ids[run1.ID] || !ids[run2.ID] {
		t.Errorf("expected both run1 and run2 in results, got %v", ids)
	}
}

func TestRunnerRunSuccess(t *testing.T) {
	s := testutil.NewStore(t)
	ctx := context.Background()

	// Create a company and agent
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

	// Create heartbeat runner
	actLog := activity.New(s)
	commentSvc := comments.New(s)
	issueSvc := issues.New(s)
	registry := heartbeat.NewDefaultRegistry()
	runner := heartbeat.New(s, agentSvc, issueSvc, commentSvc, actLog, registry, nil)

	// Run a heartbeat
	run, err := runner.Run(ctx, agent.ID)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if run.Status != "success" {
		t.Errorf("Status = %q, want %q", run.Status, "success")
	}
	if run.Summary == nil || *run.Summary == "" {
		t.Error("Summary should not be empty after successful run")
	}
	if run.FinishedAt == nil {
		t.Error("FinishedAt should not be nil after completed run")
	}
}

func TestRunnerRunNotFound(t *testing.T) {
	s := testutil.NewStore(t)
	ctx := context.Background()

	// Create heartbeat runner with no agents
	actLog := activity.New(s)
	commentSvc := comments.New(s)
	agentSvc := agents.New(s, activity.New(s))
	issueSvc := issues.New(s)
	registry := heartbeat.NewDefaultRegistry()
	runner := heartbeat.New(s, agentSvc, issueSvc, commentSvc, actLog, registry, nil)

	// Run with non-existent agent
	_, err := runner.Run(ctx, "nonexistent-agent-id")
	if err == nil {
		t.Fatal("expected error for non-existent agent")
	}
	if !errors.Is(err, agents.ErrNotFound) {
		t.Errorf("expected agents.ErrNotFound, got %v", err)
	}
}

func TestRunnerRunWithIssue(t *testing.T) {
	s := testutil.NewStore(t)
	ctx := context.Background()

	// Create a company, agent, and issue
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

	issueSvc := issues.New(s)
	issue, err := issueSvc.Create(ctx, company.ID, "Test issue", "Issue body", "default", "open", "", nil)
	if err != nil {
		t.Fatalf("Create issue: %v", err)
	}

	// Create heartbeat runner
	actLog := activity.New(s)
	commentSvc := comments.New(s)
	registry := heartbeat.NewDefaultRegistry()
	runner := heartbeat.New(s, agentSvc, issueSvc, commentSvc, actLog, registry, nil)

	// Run a heartbeat
	run, err := runner.Run(ctx, agent.ID)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if run.Status != "success" {
		t.Errorf("Status = %q, want %q", run.Status, "success")
	}
	// The run should have selected the issue
	if run.IssueID == nil || *run.IssueID != issue.ID {
		t.Errorf("IssueID = %v, want %q", run.IssueID, issue.ID)
	}
}

func TestStubAdapterRun(t *testing.T) {
	adapter := &heartbeat.StubAdapter{}
	agent := &domain.Agent{ID: "test-agent"}

	result, err := adapter.Run(context.Background(), agent, nil)
	if err != nil {
		t.Fatalf("StubAdapter.Run: %v", err)
	}
	if result.Status != "success" {
		t.Errorf("Status = %q, want %q", result.Status, "success")
	}
	if result.Summary == "" {
		t.Error("Summary should not be empty")
	}
}

func TestRegistryGetAndRegister(t *testing.T) {
	registry := heartbeat.NewRegistry()
	adapter := &heartbeat.StubAdapter{}

	registry.Register("stub", adapter)
	got := registry.Get("stub")
	if got != adapter {
		t.Errorf("Get(stub) returned different adapter than registered")
	}

	notFound := registry.Get("nonexistent")
	if notFound != nil {
		t.Errorf("Get(nonexistent) should return nil, got %v", notFound)
	}
}

func TestDefaultRegistry(t *testing.T) {
	registry := heartbeat.NewDefaultRegistry()
	stub := registry.Get("stub")
	if stub == nil {
		t.Fatal("expected stub adapter in default registry")
	}
}

// TestRunnerRunAdapterError tests the error path when adapter returns an error
func TestRunnerRunAdapterError(t *testing.T) {
	s := testutil.NewStore(t)
	ctx := context.Background()

	// Create a company and agent
	companySvc := companies.New(s)
	company, err := companySvc.Create(ctx, "Test Corp", "test", "Test company")
	if err != nil {
		t.Fatalf("Create company: %v", err)
	}

	agentSvc := agents.New(s, activity.New(s))
	agent, err := agentSvc.Create(ctx, company.ID, "alice", "Alice", "agent", nil, "error-adapter")
	if err != nil {
		t.Fatalf("Create agent: %v", err)
	}

	// Create an error adapter that always fails
	errorAdapter := newErrorAdapter(errors.New("adapter error for testing"))

	// Create heartbeat runner with the error adapter
	actLog := activity.New(s)
	commentSvc := comments.New(s)
	issueSvc := issues.New(s)
	registry := heartbeat.NewRegistry()
	registry.Register("error-adapter", errorAdapter)
	runner := heartbeat.New(s, agentSvc, issueSvc, commentSvc, actLog, registry, nil)

	// Run a heartbeat, expecting an error
	run, err := runner.Run(ctx, agent.ID)
	if err == nil {
		t.Fatal("expected error from adapter, got nil")
	}
	if run != nil {
		t.Errorf("expected nil run when adapter returns error, got %v", run)
	}

	// Verify the heartbeat run was created and marked with error status
	runs, err := runner.ListByAgent(ctx, agent.ID)
	if err != nil {
		t.Fatalf("ListByAgent: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("expected 1 heartbeat run, got %d", len(runs))
	}

	run = runs[0]
	if run.Status != "error" {
		t.Errorf("Status = %q, want %q", run.Status, "error")
	}
	if run.FinishedAt == nil {
		t.Error("FinishedAt should not be nil after error")
	}
}

func TestRunnerCancel(t *testing.T) {
	s := testutil.NewStore(t)
	ctx := context.Background()

	// Create a company and agent
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

	// Create heartbeat runner
	actLog := activity.New(s)
	commentSvc := comments.New(s)
	registry := heartbeat.NewDefaultRegistry()
	runner := heartbeat.New(s, agentSvc, nil, commentSvc, actLog, registry, nil)

	// Create a heartbeat run with "running" status
	run, err := runner.Create(ctx, agent.ID, nil, "running")
	if err != nil {
		t.Fatalf("Create run: %v", err)
	}

	// Cancel the running run
	cancelled, err := runner.Cancel(ctx, run.ID)
	if err != nil {
		t.Fatalf("Cancel run: %v", err)
	}
	if cancelled.Status != "cancelled" {
		t.Errorf("Status = %q, want %q", cancelled.Status, "cancelled")
	}
	if cancelled.FinishedAt == nil {
		t.Error("FinishedAt should be set after cancel")
	}

	// Try to cancel an already-cancelled run (should fail with ErrTerminalStatus)
	_, err = runner.Cancel(ctx, run.ID)
	if !errors.Is(err, heartbeat.ErrTerminalStatus) {
		t.Errorf("Cancel already-cancelled run: expected ErrTerminalStatus, got %v", err)
	}

	// Try to cancel a non-existent run (should fail with ErrNotFound)
	_, err = runner.Cancel(ctx, "nonexistent")
	if !errors.Is(err, heartbeat.ErrNotFound) {
		t.Errorf("Cancel non-existent run: expected ErrNotFound, got %v", err)
	}
}

// TestRecoverStaleRuns verifies that RecoverStaleRuns resets runs older than Timeout
// to "error" status, and leaves younger runs untouched.
func TestRecoverStaleRuns(t *testing.T) {
	s := testutil.NewStore(t)
	ctx := context.Background()

	// Create company and agents
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
	agent2, err := agentSvc.Create(ctx, company.ID, "bob", "Bob", "agent", nil, "stub")
	if err != nil {
		t.Fatalf("Create agent2: %v", err)
	}

	actLog := activity.New(s)
	commentSvc := comments.New(s)
	registry := heartbeat.NewDefaultRegistry()
	runner := heartbeat.New(s, agentSvc, nil, commentSvc, actLog, registry, nil)
	// Use a short timeout so we can manufacture stale runs easily.
	runner.Timeout = 10 * time.Minute

	// Create a stale run: insert with started_at far in the past.
	staleStartedAt := time.Now().UTC().Add(-30 * time.Minute).Format(time.RFC3339)
	_, err = s.DB.ExecContext(ctx,
		`INSERT INTO heartbeat_runs(id, agent_id, status, started_at) VALUES (?, ?, 'running', ?)`,
		"stale-run-id", agent.ID, staleStartedAt,
	)
	if err != nil {
		t.Fatalf("insert stale run: %v", err)
	}

	// Create a fresh run: started_at just now (younger than Timeout).
	freshStartedAt := time.Now().UTC().Format(time.RFC3339)
	_, err = s.DB.ExecContext(ctx,
		`INSERT INTO heartbeat_runs(id, agent_id, status, started_at) VALUES (?, ?, 'running', ?)`,
		"fresh-run-id", agent2.ID, freshStartedAt,
	)
	if err != nil {
		t.Fatalf("insert fresh run: %v", err)
	}

	// Run the watchdog.
	runner.RecoverStaleRuns(ctx)

	// The stale run should be in "error" status.
	staleRun, err := runner.GetByID(ctx, "stale-run-id")
	if err != nil {
		t.Fatalf("GetByID stale: %v", err)
	}
	if staleRun.Status != "error" {
		t.Errorf("stale run status = %q, want %q", staleRun.Status, "error")
	}
	if staleRun.Error == nil || *staleRun.Error != "recovered: stale run on startup" {
		t.Errorf("stale run error = %v, want 'recovered: stale run on startup'", staleRun.Error)
	}

	// The fresh run should still be "running".
	freshRun, err := runner.GetByID(ctx, "fresh-run-id")
	if err != nil {
		t.Fatalf("GetByID fresh: %v", err)
	}
	if freshRun.Status != "running" {
		t.Errorf("fresh run status = %q, want %q", freshRun.Status, "running")
	}
}

// TestListOpenByPriority verifies that issues are returned in priority order
// (urgent > high > medium > low) and that archived issues are excluded.
func TestListOpenByPriority(t *testing.T) {
	s := testutil.NewStore(t)
	ctx := context.Background()

	companySvc := companies.New(s)
	company, err := companySvc.Create(ctx, "Test Corp", "test", "Test company")
	if err != nil {
		t.Fatalf("Create company: %v", err)
	}

	issueSvc := issues.New(s)

	// Create issues with explicit priorities.
	low, err := issueSvc.Create(ctx, company.ID, "Low issue", "", "default", "open", "low", nil)
	if err != nil {
		t.Fatalf("Create low issue: %v", err)
	}
	medium, err := issueSvc.Create(ctx, company.ID, "Medium issue", "", "default", "open", "medium", nil)
	if err != nil {
		t.Fatalf("Create medium issue: %v", err)
	}
	high, err := issueSvc.Create(ctx, company.ID, "High issue", "", "default", "open", "high", nil)
	if err != nil {
		t.Fatalf("Create high issue: %v", err)
	}
	urgent, err := issueSvc.Create(ctx, company.ID, "Urgent issue", "", "default", "open", "urgent", nil)
	if err != nil {
		t.Fatalf("Create urgent issue: %v", err)
	}
	// Archived issue with high priority — should not appear in results.
	archived, err := issueSvc.Create(ctx, company.ID, "Archived issue", "", "default", "open", "high", nil)
	if err != nil {
		t.Fatalf("Create archived issue: %v", err)
	}
	if err := issueSvc.Archive(ctx, archived.ID); err != nil {
		t.Fatalf("Archive issue: %v", err)
	}

	// List by priority.
	result, err := issueSvc.ListOpenByPriority(ctx, company.ID)
	if err != nil {
		t.Fatalf("ListOpenByPriority: %v", err)
	}

	// Should have 4 issues (not the archived one).
	if len(result) != 4 {
		t.Fatalf("expected 4 issues, got %d", len(result))
	}

	// Verify order: urgent, high, medium, low.
	wantOrder := []string{urgent.ID, high.ID, medium.ID, low.ID}
	for i, want := range wantOrder {
		if result[i].ID != want {
			t.Errorf("result[%d].ID = %q, want %q", i, result[i].ID, want)
		}
	}

	// Verify the archived issue is not present.
	for _, issue := range result {
		if issue.ID == archived.ID {
			t.Error("archived issue should not appear in ListOpenByPriority results")
		}
	}
}
