package agents_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ubunatic/paperclip-go/internal/agents"
	"github.com/ubunatic/paperclip-go/internal/companies"
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
	svc := agents.New(s)
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

	svc := agents.New(s)
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
	svc := agents.New(s)
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
	svc := agents.New(s)
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
	svc := agents.New(s)
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
