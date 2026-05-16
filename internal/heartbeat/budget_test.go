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

func intPtr(v int) *int { return &v }

func TestBudgetEnforcementBlocked(t *testing.T) {
	s := testutil.NewStore(t)
	ctx := context.Background()

	companySvc := companies.New(s)
	company, err := companySvc.Create(ctx, "Test Corp", "budget-test", "Budget test company")
	if err != nil {
		t.Fatalf("Create company: %v", err)
	}

	agentSvc := agents.New(s, activity.New(s))
	agent, err := agentSvc.Create(ctx, company.ID, "budgeted", "Budgeted Agent", "agent", nil, "mock")
	if err != nil {
		t.Fatalf("Create agent: %v", err)
	}

	// Set budget_limit=100 and budget_used=100 directly
	_, err = s.DB.ExecContext(ctx, `UPDATE agents SET budget_limit = 100, budget_used = 100 WHERE id = ?`, agent.ID)
	if err != nil {
		t.Fatalf("Set budget: %v", err)
	}

	mockAdapter := heartbeat.NewMockAdapter(func(a *domain.Agent, i *domain.Issue) (*domain.RunResult, error) {
		return &domain.RunResult{Status: "success", Summary: "ok"}, nil
	})
	registry := heartbeat.NewRegistry()
	registry.Register("mock", mockAdapter)

	issueSvc := issues.New(s)
	commentSvc := comments.New(s)
	actLog := activity.New(s)
	runner := heartbeat.New(s, agentSvc, issueSvc, commentSvc, actLog, registry, nil)

	_, err = runner.Run(ctx, agent.ID)
	if !errors.Is(err, heartbeat.ErrBudgetExceeded) {
		t.Errorf("expected ErrBudgetExceeded, got %v", err)
	}
}

func TestBudgetEnforcementAllowed(t *testing.T) {
	s := testutil.NewStore(t)
	ctx := context.Background()

	companySvc := companies.New(s)
	company, err := companySvc.Create(ctx, "Test Corp", "budget-allowed", "Budget test company")
	if err != nil {
		t.Fatalf("Create company: %v", err)
	}

	agentSvc := agents.New(s, activity.New(s))
	agent, err := agentSvc.Create(ctx, company.ID, "budgeted2", "Budgeted Agent 2", "agent", nil, "mock")
	if err != nil {
		t.Fatalf("Create agent: %v", err)
	}

	// Set budget_limit=100 and budget_used=50 (still within budget)
	_, err = s.DB.ExecContext(ctx, `UPDATE agents SET budget_limit = 100, budget_used = 50 WHERE id = ?`, agent.ID)
	if err != nil {
		t.Fatalf("Set budget: %v", err)
	}

	ran := false
	mockAdapter := heartbeat.NewMockAdapter(func(a *domain.Agent, i *domain.Issue) (*domain.RunResult, error) {
		ran = true
		return &domain.RunResult{Status: "success", Summary: "ok"}, nil
	})
	registry := heartbeat.NewRegistry()
	registry.Register("mock", mockAdapter)

	issueSvc := issues.New(s)
	commentSvc := comments.New(s)
	actLog := activity.New(s)
	runner := heartbeat.New(s, agentSvc, issueSvc, commentSvc, actLog, registry, nil)

	run, err := runner.Run(ctx, agent.ID)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !ran {
		t.Error("expected adapter to be called")
	}
	if run.Status != "success" {
		t.Errorf("Status = %q, want %q", run.Status, "success")
	}
}

func TestBudgetUsedIncrementedAfterRun(t *testing.T) {
	s := testutil.NewStore(t)
	ctx := context.Background()

	companySvc := companies.New(s)
	company, err := companySvc.Create(ctx, "Test Corp", "budget-inc", "Budget test company")
	if err != nil {
		t.Fatalf("Create company: %v", err)
	}

	agentSvc := agents.New(s, activity.New(s))
	agent, err := agentSvc.Create(ctx, company.ID, "costing", "Costing Agent", "agent", nil, "cost-mock")
	if err != nil {
		t.Fatalf("Create agent: %v", err)
	}

	mockAdapter := heartbeat.NewMockAdapter(func(a *domain.Agent, i *domain.Issue) (*domain.RunResult, error) {
		return &domain.RunResult{Status: "success", Summary: "done", Cost: 10}, nil
	})
	registry := heartbeat.NewRegistry()
	registry.Register("cost-mock", mockAdapter)

	issueSvc := issues.New(s)
	commentSvc := comments.New(s)
	actLog := activity.New(s)
	runner := heartbeat.New(s, agentSvc, issueSvc, commentSvc, actLog, registry, nil)

	_, err = runner.Run(ctx, agent.ID)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	updated, err := agentSvc.Get(ctx, agent.ID)
	if err != nil {
		t.Fatalf("Get agent: %v", err)
	}
	if updated.BudgetUsed != 10 {
		t.Errorf("BudgetUsed = %d, want 10", updated.BudgetUsed)
	}
}
