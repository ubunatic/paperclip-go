package agents_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ubunatic/paperclip-go/internal/activity"
	"github.com/ubunatic/paperclip-go/internal/agents"
	"github.com/ubunatic/paperclip-go/internal/companies"
	"github.com/ubunatic/paperclip-go/internal/issues"
	"github.com/ubunatic/paperclip-go/internal/testutil"
)

func TestAgentCreate(t *testing.T) {
	s := testutil.NewStore(t)
	ctx := context.Background()

	// Create a company first
	companySvc := companies.New(s)
	company, err := companySvc.Create(ctx, "Test Corp", "test", "Test company")
	if err != nil {
		t.Fatalf("Create company: %v", err)
	}

	// Create an agent
	svc := agents.New(s, activity.New(s))
	agent, err := svc.Create(ctx, company.ID, "alice", "Alice", "manager", nil, "stub")
	if err != nil {
		t.Fatalf("Create agent: %v", err)
	}
	if agent.ID == "" {
		t.Fatal("expected non-empty ID")
	}
	if agent.Shortname != "alice" {
		t.Errorf("Shortname = %q, want %q", agent.Shortname, "alice")
	}
	if agent.DisplayName != "Alice" {
		t.Errorf("DisplayName = %q, want %q", agent.DisplayName, "Alice")
	}
	if agent.Role != "manager" {
		t.Errorf("Role = %q, want %q", agent.Role, "manager")
	}
	if agent.ReportsTo != nil {
		t.Errorf("ReportsTo = %v, want nil", agent.ReportsTo)
	}
	if agent.Adapter != "stub" {
		t.Errorf("Adapter = %q, want %q", agent.Adapter, "stub")
	}
	if agent.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
	if agent.RuntimeState != "idle" {
		t.Errorf("RuntimeState = %q, want %q", agent.RuntimeState, "idle")
	}
}

func TestAgentGet(t *testing.T) {
	s := testutil.NewStore(t)
	ctx := context.Background()

	// Create a company and agent
	companySvc := companies.New(s)
	company, err := companySvc.Create(ctx, "Test Corp", "test", "Test company")
	if err != nil {
		t.Fatalf("Create company: %v", err)
	}

	svc := agents.New(s, activity.New(s))
	agent, err := svc.Create(ctx, company.ID, "alice", "Alice", "manager", nil, "stub")
	if err != nil {
		t.Fatalf("Create agent: %v", err)
	}

	// Get the agent
	got, err := svc.Get(ctx, agent.ID)
	if err != nil {
		t.Fatalf("Get agent: %v", err)
	}
	if got.ID != agent.ID {
		t.Errorf("Get.ID = %q, want %q", got.ID, agent.ID)
	}
	if got.Shortname != agent.Shortname {
		t.Errorf("Get.Shortname = %q, want %q", got.Shortname, agent.Shortname)
	}
}

func TestAgentGetNotFound(t *testing.T) {
	s := testutil.NewStore(t)
	svc := agents.New(s, activity.New(s))
	ctx := context.Background()

	_, err := svc.Get(ctx, "nonexistent-id")
	if !errors.Is(err, agents.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestAgentListByCompany(t *testing.T) {
	s := testutil.NewStore(t)
	ctx := context.Background()

	// Create two companies
	companySvc := companies.New(s)
	company1, err := companySvc.Create(ctx, "Corp A", "corpa", "")
	if err != nil {
		t.Fatalf("Create company 1: %v", err)
	}
	company2, err := companySvc.Create(ctx, "Corp B", "corpb", "")
	if err != nil {
		t.Fatalf("Create company 2: %v", err)
	}

	// Create agents for both companies
	svc := agents.New(s, activity.New(s))
	_, err = svc.Create(ctx, company1.ID, "alice", "Alice", "", nil, "stub")
	if err != nil {
		t.Fatalf("Create agent 1: %v", err)
	}
	_, err = svc.Create(ctx, company1.ID, "bob", "Bob", "", nil, "stub")
	if err != nil {
		t.Fatalf("Create agent 2: %v", err)
	}
	_, err = svc.Create(ctx, company2.ID, "charlie", "Charlie", "", nil, "stub")
	if err != nil {
		t.Fatalf("Create agent 3: %v", err)
	}

	// List by company1
	list, err := svc.ListByCompany(ctx, company1.ID)
	if err != nil {
		t.Fatalf("ListByCompany: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("ListByCompany len = %d, want 2", len(list))
	}

	// List by company2
	list2, err := svc.ListByCompany(ctx, company2.ID)
	if err != nil {
		t.Fatalf("ListByCompany 2: %v", err)
	}
	if len(list2) != 1 {
		t.Errorf("ListByCompany 2 len = %d, want 1", len(list2))
	}
}

func TestAgentUniqueConstraint(t *testing.T) {
	s := testutil.NewStore(t)
	ctx := context.Background()

	// Create a company
	companySvc := companies.New(s)
	company, err := companySvc.Create(ctx, "Test Corp", "test", "")
	if err != nil {
		t.Fatalf("Create company: %v", err)
	}

	// Create first agent
	svc := agents.New(s, activity.New(s))
	_, err = svc.Create(ctx, company.ID, "alice", "Alice", "", nil, "stub")
	if err != nil {
		t.Fatalf("Create agent 1: %v", err)
	}

	// Try to create duplicate (same company, same shortname)
	_, err = svc.Create(ctx, company.ID, "alice", "Different Name", "", nil, "stub")
	if err == nil {
		t.Fatal("expected error on duplicate shortname within company")
	}
}

func TestDeleteAgent(t *testing.T) {
	s := testutil.NewStore(t)
	ctx := context.Background()

	// Create a company and agent
	companySvc := companies.New(s)
	company, err := companySvc.Create(ctx, "Test Corp", "test", "Test company")
	if err != nil {
		t.Fatalf("Create company: %v", err)
	}

	svc := agents.New(s, activity.New(s))
	agent, err := svc.Create(ctx, company.ID, "alice", "Alice", "manager", nil, "stub")
	if err != nil {
		t.Fatalf("Create agent: %v", err)
	}

	// Delete the agent should succeed
	err = svc.Delete(ctx, agent.ID)
	if err != nil {
		t.Fatalf("Delete agent: %v", err)
	}

	// Get should return ErrNotFound
	_, err = svc.Get(ctx, agent.ID)
	if !errors.Is(err, agents.ErrNotFound) {
		t.Fatalf("Get after delete: expected ErrNotFound, got %v", err)
	}
}

func TestDeleteAgentNotFound(t *testing.T) {
	s := testutil.NewStore(t)
	ctx := context.Background()

	svc := agents.New(s, activity.New(s))
	err := svc.Delete(ctx, "nonexistent-id")
	if !errors.Is(err, agents.ErrNotFound) {
		t.Fatalf("Delete nonexistent: expected ErrNotFound, got %v", err)
	}
}

func TestDeleteAgentNoCheckouts(t *testing.T) {
	s := testutil.NewStore(t)
	ctx := context.Background()

	// Create a company and agents
	companySvc := companies.New(s)
	company, err := companySvc.Create(ctx, "Test Corp", "test", "Test company")
	if err != nil {
		t.Fatalf("Create company: %v", err)
	}

	agentSvc := agents.New(s, activity.New(s))
	agent, err := agentSvc.Create(ctx, company.ID, "alice", "Alice", "engineer", nil, "stub")
	if err != nil {
		t.Fatalf("Create agent: %v", err)
	}

	// Delete agent with no checkouts should succeed
	err = agentSvc.Delete(ctx, agent.ID)
	if err != nil {
		t.Fatalf("Delete agent with no checkouts: %v", err)
	}
}

func TestDeleteAgentWithHeartbeatRuns(t *testing.T) {
	s := testutil.NewStore(t)
	ctx := context.Background()

	// Create a company, agent, issue, and heartbeat run
	companySvc := companies.New(s)
	company, err := companySvc.Create(ctx, "Test Corp", "test", "Test company")
	if err != nil {
		t.Fatalf("Create company: %v", err)
	}

	agentSvc := agents.New(s, activity.New(s))
	agent, err := agentSvc.Create(ctx, company.ID, "alice", "Alice", "engineer", nil, "stub")
	if err != nil {
		t.Fatalf("Create agent: %v", err)
	}

	// Create an issue
	issueSvc := issues.New(s)
	issue, err := issueSvc.Create(ctx, company.ID, "Test Issue", "Body", nil)
	if err != nil {
		t.Fatalf("Create issue: %v", err)
	}

	// Create a heartbeat run for this agent
	_, err = s.DB.ExecContext(ctx,
		`INSERT INTO heartbeat_runs(id, agent_id, issue_id, status, started_at)
		 VALUES (?, ?, ?, 'running', ?)`,
		"heartbeat-1", agent.ID, issue.ID, "2024-01-01T00:00:00Z",
	)
	if err != nil {
		t.Fatalf("Create heartbeat run: %v", err)
	}

	// Try to delete agent - should fail with ErrHasActiveCheckout
	err = agentSvc.Delete(ctx, agent.ID)
	if !errors.Is(err, agents.ErrHasActiveCheckout) {
		t.Fatalf("Delete with heartbeat runs: expected ErrHasActiveCheckout, got %v", err)
	}

	// Verify agent still exists
	_, err = agentSvc.Get(ctx, agent.ID)
	if err != nil {
		t.Fatalf("Get after failed delete: %v", err)
	}
}

func TestDeleteAgentHasActiveCheckout(t *testing.T) {
	s := testutil.NewStore(t)
	ctx := context.Background()

	// Create a company, agent, and issue
	companySvc := companies.New(s)
	company, err := companySvc.Create(ctx, "Test Corp", "test", "Test company")
	if err != nil {
		t.Fatalf("Create company: %v", err)
	}

	agentSvc := agents.New(s, activity.New(s))
	agent, err := agentSvc.Create(ctx, company.ID, "alice", "Alice", "engineer", nil, "stub")
	if err != nil {
		t.Fatalf("Create agent: %v", err)
	}

	// Create an issue assigned to the agent
	issueSvc := issues.New(s)
	agentIDPtr := &agent.ID
	issue, err := issueSvc.Create(ctx, company.ID, "Test Issue", "Body", agentIDPtr)
	if err != nil {
		t.Fatalf("Create issue: %v", err)
	}

	// Checkout the issue (sets in_progress status and checked_out_by)
	err = issueSvc.Checkout(ctx, issue.ID, agent.ID)
	if err != nil {
		t.Fatalf("Checkout: %v", err)
	}

	// Try to delete agent - should fail with ErrHasActiveCheckout
	err = agentSvc.Delete(ctx, agent.ID)
	if !errors.Is(err, agents.ErrHasActiveCheckout) {
		t.Fatalf("Delete with active checkout: expected ErrHasActiveCheckout, got %v", err)
	}

	// Verify agent still exists
	_, err = agentSvc.Get(ctx, agent.ID)
	if err != nil {
		t.Fatalf("Get after failed delete: %v", err)
	}
}

func TestPauseAgent(t *testing.T) {
	s := testutil.NewStore(t)
	ctx := context.Background()

	// Create a company and agent
	companySvc := companies.New(s)
	company, err := companySvc.Create(ctx, "Test Corp", "test", "Test company")
	if err != nil {
		t.Fatalf("Create company: %v", err)
	}

	svc := agents.New(s, activity.New(s))
	agent, err := svc.Create(ctx, company.ID, "alice", "Alice", "manager", nil, "stub")
	if err != nil {
		t.Fatalf("Create agent: %v", err)
	}

	// Pause from idle state
	paused, err := svc.Pause(ctx, agent.ID)
	if err != nil {
		t.Fatalf("Pause: %v", err)
	}
	if paused.RuntimeState != "paused" {
		t.Errorf("RuntimeState = %q, want %q", paused.RuntimeState, "paused")
	}

	// Verify persistence
	fetched, err := svc.Get(ctx, agent.ID)
	if err != nil {
		t.Fatalf("Get after pause: %v", err)
	}
	if fetched.RuntimeState != "paused" {
		t.Errorf("Fetched RuntimeState = %q, want %q", fetched.RuntimeState, "paused")
	}
}

func TestPauseFromRunning(t *testing.T) {
	s := testutil.NewStore(t)
	ctx := context.Background()

	// Create a company and agent
	companySvc := companies.New(s)
	company, err := companySvc.Create(ctx, "Test Corp", "test", "Test company")
	if err != nil {
		t.Fatalf("Create company: %v", err)
	}

	svc := agents.New(s, activity.New(s))
	agent, err := svc.Create(ctx, company.ID, "alice", "Alice", "manager", nil, "stub")
	if err != nil {
		t.Fatalf("Create agent: %v", err)
	}

	// Pause to set state to paused first
	_, err = svc.Pause(ctx, agent.ID)
	if err != nil {
		t.Fatalf("Pause: %v", err)
	}

	// Resume to set state to running
	_, err = svc.Resume(ctx, agent.ID)
	if err != nil {
		t.Fatalf("Resume: %v", err)
	}

	// Pause again from running state
	paused, err := svc.Pause(ctx, agent.ID)
	if err != nil {
		t.Fatalf("Pause from running: %v", err)
	}
	if paused.RuntimeState != "paused" {
		t.Errorf("RuntimeState = %q, want %q", paused.RuntimeState, "paused")
	}

	// Verify persistence
	fetched, err := svc.Get(ctx, agent.ID)
	if err != nil {
		t.Fatalf("Get after pause: %v", err)
	}
	if fetched.RuntimeState != "paused" {
		t.Errorf("Fetched RuntimeState = %q, want %q", fetched.RuntimeState, "paused")
	}
}

func TestResumeAgent(t *testing.T) {
	s := testutil.NewStore(t)
	ctx := context.Background()

	// Create a company and agent
	companySvc := companies.New(s)
	company, err := companySvc.Create(ctx, "Test Corp", "test", "Test company")
	if err != nil {
		t.Fatalf("Create company: %v", err)
	}

	svc := agents.New(s, activity.New(s))
	agent, err := svc.Create(ctx, company.ID, "alice", "Alice", "manager", nil, "stub")
	if err != nil {
		t.Fatalf("Create agent: %v", err)
	}

	// Pause first
	_, err = svc.Pause(ctx, agent.ID)
	if err != nil {
		t.Fatalf("Pause: %v", err)
	}

	// Resume from paused state
	resumed, err := svc.Resume(ctx, agent.ID)
	if err != nil {
		t.Fatalf("Resume: %v", err)
	}
	if resumed.RuntimeState != "running" {
		t.Errorf("RuntimeState = %q, want %q", resumed.RuntimeState, "running")
	}

	// Verify persistence
	fetched, err := svc.Get(ctx, agent.ID)
	if err != nil {
		t.Fatalf("Get after resume: %v", err)
	}
	if fetched.RuntimeState != "running" {
		t.Errorf("Fetched RuntimeState = %q, want %q", fetched.RuntimeState, "running")
	}
}

func TestTerminateAgent(t *testing.T) {
	s := testutil.NewStore(t)
	ctx := context.Background()

	// Create a company and agent
	companySvc := companies.New(s)
	company, err := companySvc.Create(ctx, "Test Corp", "test", "Test company")
	if err != nil {
		t.Fatalf("Create company: %v", err)
	}

	svc := agents.New(s, activity.New(s))
	agent, err := svc.Create(ctx, company.ID, "alice", "Alice", "manager", nil, "stub")
	if err != nil {
		t.Fatalf("Create agent: %v", err)
	}

	// Terminate from idle state
	terminated, err := svc.Terminate(ctx, agent.ID)
	if err != nil {
		t.Fatalf("Terminate: %v", err)
	}
	if terminated.RuntimeState != "terminated" {
		t.Errorf("RuntimeState = %q, want %q", terminated.RuntimeState, "terminated")
	}

	// Verify persistence
	fetched, err := svc.Get(ctx, agent.ID)
	if err != nil {
		t.Fatalf("Get after terminate: %v", err)
	}
	if fetched.RuntimeState != "terminated" {
		t.Errorf("Fetched RuntimeState = %q, want %q", fetched.RuntimeState, "terminated")
	}
}

func TestInvalidStateTransitions(t *testing.T) {
	s := testutil.NewStore(t)
	ctx := context.Background()

	// Create a company and agent
	companySvc := companies.New(s)
	company, err := companySvc.Create(ctx, "Test Corp", "test", "Test company")
	if err != nil {
		t.Fatalf("Create company: %v", err)
	}

	svc := agents.New(s, activity.New(s))
	agent, err := svc.Create(ctx, company.ID, "alice", "Alice", "manager", nil, "stub")
	if err != nil {
		t.Fatalf("Create agent: %v", err)
	}

	// Pause first
	_, err = svc.Pause(ctx, agent.ID)
	if err != nil {
		t.Fatalf("Pause: %v", err)
	}

	// Try to pause again (already paused)
	_, err = svc.Pause(ctx, agent.ID)
	if !errors.Is(err, agents.ErrInvalidTransition) {
		t.Errorf("Pause paused: expected ErrInvalidTransition, got %v", err)
	}

	// Resume back to running for terminate test
	_, err = svc.Resume(ctx, agent.ID)
	if err != nil {
		t.Fatalf("Resume: %v", err)
	}

	// Try to resume again (already running)
	_, err = svc.Resume(ctx, agent.ID)
	if !errors.Is(err, agents.ErrInvalidTransition) {
		t.Errorf("Resume running: expected ErrInvalidTransition, got %v", err)
	}

	// Terminate
	_, err = svc.Terminate(ctx, agent.ID)
	if err != nil {
		t.Fatalf("Terminate: %v", err)
	}

	// Try to terminate again (already terminated)
	_, err = svc.Terminate(ctx, agent.ID)
	if !errors.Is(err, agents.ErrInvalidTransition) {
		t.Errorf("Terminate terminated: expected ErrInvalidTransition, got %v", err)
	}

	// Try to pause terminated agent
	_, err = svc.Pause(ctx, agent.ID)
	if !errors.Is(err, agents.ErrInvalidTransition) {
		t.Errorf("Pause terminated: expected ErrInvalidTransition, got %v", err)
	}
}

func TestResumeOnIdleState(t *testing.T) {
	s := testutil.NewStore(t)
	ctx := context.Background()

	// Create a company and agent (initially in idle state)
	companySvc := companies.New(s)
	company, err := companySvc.Create(ctx, "Test Corp", "test", "Test company")
	if err != nil {
		t.Fatalf("Create company: %v", err)
	}

	svc := agents.New(s, activity.New(s))
	agent, err := svc.Create(ctx, company.ID, "alice", "Alice", "manager", nil, "stub")
	if err != nil {
		t.Fatalf("Create agent: %v", err)
	}

	// Try to resume on idle state (should fail)
	_, err = svc.Resume(ctx, agent.ID)
	if !errors.Is(err, agents.ErrInvalidTransition) {
		t.Errorf("Resume on idle: expected ErrInvalidTransition, got %v", err)
	}
}

func TestActivityLoggingOnTransition(t *testing.T) {
	s := testutil.NewStore(t)
	ctx := context.Background()
	log := activity.New(s)

	// Create company
	companySvc := companies.New(s)
	company, err := companySvc.Create(ctx, "Test Corp", "test", "Test company")
	if err != nil {
		t.Fatalf("Create company: %v", err)
	}

	// Create agent
	svc := agents.New(s, log)
	agent, err := svc.Create(ctx, company.ID, "alice", "Alice", "manager", nil, "stub")
	if err != nil {
		t.Fatalf("Create agent: %v", err)
	}

	// Call Pause (should log the transition)
	_, err = svc.Pause(ctx, agent.ID)
	if err != nil {
		t.Fatalf("Pause: %v", err)
	}

	// Query activity_log and assert entry exists with action="pause", entity_kind="agent"
	var count int
	err = s.DB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM activity_log WHERE entity_id = ? AND entity_kind = 'agent' AND action = 'pause'`,
		agent.ID,
	).Scan(&count)
	if err != nil {
		t.Fatalf("Query activity_log: %v", err)
	}
	if count == 0 {
		t.Error("expected activity log entry for pause action, found none")
	}

	// Call Resume and verify log entry
	_, err = svc.Resume(ctx, agent.ID)
	if err != nil {
		t.Fatalf("Resume: %v", err)
	}

	err = s.DB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM activity_log WHERE entity_id = ? AND entity_kind = 'agent' AND action = 'resume'`,
		agent.ID,
	).Scan(&count)
	if err != nil {
		t.Fatalf("Query activity_log for resume: %v", err)
	}
	if count == 0 {
		t.Error("expected activity log entry for resume action, found none")
	}
}

func TestAgentConfigurationNotAltered(t *testing.T) {
	s := testutil.NewStore(t)
	ctx := context.Background()

	// Create a company
	companySvc := companies.New(s)
	company, err := companySvc.Create(ctx, "Test Corp", "test", "Test company")
	if err != nil {
		t.Fatalf("Create company: %v", err)
	}

	// Create an agent
	svc := agents.New(s, activity.New(s))
	agent, err := svc.Create(ctx, company.ID, "alice", "Alice", "manager", nil, "stub")
	if err != nil {
		t.Fatalf("Create agent: %v", err)
	}

	// Update with initial configuration
	config := map[string]any{"key1": "value1"}
	updated, err := svc.Update(ctx, agent.ID, nil, nil, nil, config)
	if err != nil {
		t.Fatalf("Update with config: %v", err)
	}
	if updated.Configuration == nil || updated.Configuration["key1"] != "value1" {
		t.Errorf("Configuration not set correctly: %v", updated.Configuration)
	}

	// Update only displayName, verify configuration unchanged
	newDisplay := "Alice Updated"
	updated2, err := svc.Update(ctx, agent.ID, &newDisplay, nil, nil, nil)
	if err != nil {
		t.Fatalf("Update displayName: %v", err)
	}
	if updated2.DisplayName != "Alice Updated" {
		t.Errorf("DisplayName = %q, want %q", updated2.DisplayName, "Alice Updated")
	}
	if updated2.Configuration == nil || updated2.Configuration["key1"] != "value1" {
		t.Errorf("Configuration was altered: %v", updated2.Configuration)
	}
}

func TestAgentConfigurationUpdateNotFound(t *testing.T) {
	s := testutil.NewStore(t)
	svc := agents.New(s, activity.New(s))
	ctx := context.Background()

	// Try to update configuration for non-existent agent
	config := map[string]any{"key1": "value1"}
	_, err := svc.Update(ctx, "nonexistent-id", nil, nil, nil, config)
	if !errors.Is(err, agents.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
