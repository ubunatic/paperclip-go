package heartbeat_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ubunatic/paperclip-go/internal/activity"
	"github.com/ubunatic/paperclip-go/internal/agents"
	"github.com/ubunatic/paperclip-go/internal/comments"
	"github.com/ubunatic/paperclip-go/internal/companies"
	"github.com/ubunatic/paperclip-go/internal/domain"
	"github.com/ubunatic/paperclip-go/internal/heartbeat"
	"github.com/ubunatic/paperclip-go/internal/issues"
	"github.com/ubunatic/paperclip-go/internal/testutil"
)

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
	runner := heartbeat.New(s, agentSvc, nil, commentSvc, actLog, registry)

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
	runner := heartbeat.New(s, agentSvc, nil, commentSvc, actLog, registry)

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
	runner := heartbeat.New(s, agentSvc, nil, commentSvc, actLog, registry)

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
	runner := heartbeat.New(s, agentSvc, nil, commentSvc, actLog, registry)

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
	runner := heartbeat.New(s, agentSvc, nil, commentSvc, actLog, registry)

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
	runner := heartbeat.New(s, agentSvc, issueSvc, commentSvc, actLog, registry)

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
	runner := heartbeat.New(s, agentSvc, issueSvc, commentSvc, actLog, registry)

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
	issue, err := issueSvc.Create(ctx, company.ID, "Test issue", "Issue body", "", nil)
	if err != nil {
		t.Fatalf("Create issue: %v", err)
	}

	// Create heartbeat runner
	actLog := activity.New(s)
	commentSvc := comments.New(s)
	registry := heartbeat.NewDefaultRegistry()
	runner := heartbeat.New(s, agentSvc, issueSvc, commentSvc, actLog, registry)

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
	errorAdapter := &ErrorAdapter{}

	// Create heartbeat runner with the error adapter
	actLog := activity.New(s)
	commentSvc := comments.New(s)
	issueSvc := issues.New(s)
	registry := heartbeat.NewRegistry()
	registry.Register("error-adapter", errorAdapter)
	runner := heartbeat.New(s, agentSvc, issueSvc, commentSvc, actLog, registry)

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

// ErrorAdapter is a test adapter that always returns an error
type ErrorAdapter struct{}

// Run implements the Adapter interface for ErrorAdapter, always returning an error
func (a *ErrorAdapter) Run(ctx context.Context, agent *domain.Agent, issue *domain.Issue) (*domain.RunResult, error) {
	return nil, errors.New("adapter error for testing")
}
