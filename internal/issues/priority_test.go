package issues_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ubunatic/paperclip-go/internal/companies"
	"github.com/ubunatic/paperclip-go/internal/issues"
	"github.com/ubunatic/paperclip-go/internal/testutil"
)

func TestIssuePriorityDefault(t *testing.T) {
	s := testutil.NewStore(t)
	ctx := context.Background()

	companySvc := companies.New(s)
	company, err := companySvc.Create(ctx, "Test Corp", "prio-test", "Priority test company")
	if err != nil {
		t.Fatalf("Create company: %v", err)
	}

	svc := issues.New(s)
	issue, err := svc.Create(ctx, company.ID, "Test Issue", "Body", "", "open", "", nil)
	if err != nil {
		t.Fatalf("Create issue: %v", err)
	}

	if issue.Priority != "medium" {
		t.Errorf("Priority = %q, want %q", issue.Priority, "medium")
	}

	fetched, err := svc.Get(ctx, issue.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if fetched.Priority != "medium" {
		t.Errorf("Fetched Priority = %q, want %q", fetched.Priority, "medium")
	}
}

func TestIssuePrioritySet(t *testing.T) {
	s := testutil.NewStore(t)
	ctx := context.Background()

	companySvc := companies.New(s)
	company, err := companySvc.Create(ctx, "Test Corp", "prio-set", "Priority set company")
	if err != nil {
		t.Fatalf("Create company: %v", err)
	}

	svc := issues.New(s)
	issue, err := svc.Create(ctx, company.ID, "High Prio Issue", "Body", "", "open", "high", nil)
	if err != nil {
		t.Fatalf("Create issue: %v", err)
	}

	if issue.Priority != "high" {
		t.Errorf("Priority = %q, want %q", issue.Priority, "high")
	}

	fetched, err := svc.Get(ctx, issue.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if fetched.Priority != "high" {
		t.Errorf("Fetched Priority = %q, want %q", fetched.Priority, "high")
	}
}

func TestIssueEstimateSet(t *testing.T) {
	s := testutil.NewStore(t)
	ctx := context.Background()

	companySvc := companies.New(s)
	company, err := companySvc.Create(ctx, "Test Corp", "est-set", "Estimate set company")
	if err != nil {
		t.Fatalf("Create company: %v", err)
	}

	svc := issues.New(s)
	issue, err := svc.Create(ctx, company.ID, "Estimated Issue", "Body", "", "open", "", nil)
	if err != nil {
		t.Fatalf("Create issue: %v", err)
	}

	estimate := 5
	updated, err := svc.Update(ctx, issue.ID, "", nil, nil, &estimate, nil, nil)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	if updated.Estimate == nil || *updated.Estimate != 5 {
		t.Errorf("Estimate = %v, want 5", updated.Estimate)
	}

	fetched, err := svc.Get(ctx, issue.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if fetched.Estimate == nil || *fetched.Estimate != 5 {
		t.Errorf("Fetched Estimate = %v, want 5", fetched.Estimate)
	}
}

func TestIssuePriorityInvalid(t *testing.T) {
	s := testutil.NewStore(t)
	ctx := context.Background()

	companySvc := companies.New(s)
	company, err := companySvc.Create(ctx, "Test Corp", "prio-invalid", "Invalid priority company")
	if err != nil {
		t.Fatalf("Create company: %v", err)
	}

	svc := issues.New(s)
	_, err = svc.Create(ctx, company.ID, "Bad Priority", "Body", "", "open", "nonsense", nil)
	if !errors.Is(err, issues.ErrInvalidPriority) {
		t.Errorf("expected ErrInvalidPriority for invalid priority, got %v", err)
	}
}

func TestIssuePriorityInvalidUpdate(t *testing.T) {
	s := testutil.NewStore(t)
	ctx := context.Background()

	companySvc := companies.New(s)
	company, err := companySvc.Create(ctx, "Test Corp", "prio-inv-upd", "Invalid priority update company")
	if err != nil {
		t.Fatalf("Create company: %v", err)
	}

	svc := issues.New(s)
	issue, err := svc.Create(ctx, company.ID, "Issue", "Body", "", "open", "", nil)
	if err != nil {
		t.Fatalf("Create issue: %v", err)
	}

	badPrio := "nonsense"
	_, err = svc.Update(ctx, issue.ID, "", nil, &badPrio, nil, nil, nil)
	if !errors.Is(err, issues.ErrInvalidPriority) {
		t.Errorf("expected ErrInvalidPriority on update, got %v", err)
	}
}

func TestIssueEstimateUpdate(t *testing.T) {
	s := testutil.NewStore(t)
	ctx := context.Background()

	companySvc := companies.New(s)
	company, err := companySvc.Create(ctx, "Test Corp", "est-update", "Estimate update company")
	if err != nil {
		t.Fatalf("Create company: %v", err)
	}

	svc := issues.New(s)
	issue, err := svc.Create(ctx, company.ID, "Update Issue", "Body", "", "open", "low", nil)
	if err != nil {
		t.Fatalf("Create issue: %v", err)
	}

	// Set estimate and priority independently
	estimate := 3
	prio := "urgent"
	updated, err := svc.Update(ctx, issue.ID, "", nil, &prio, &estimate, nil, nil)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	if updated.Priority != "urgent" {
		t.Errorf("Priority = %q, want %q", updated.Priority, "urgent")
	}
	if updated.Estimate == nil || *updated.Estimate != 3 {
		t.Errorf("Estimate = %v, want 3", updated.Estimate)
	}

	// Update only estimate, priority should remain
	estimate2 := 7
	updated2, err := svc.Update(ctx, issue.ID, "", nil, nil, &estimate2, nil, nil)
	if err != nil {
		t.Fatalf("Update2: %v", err)
	}

	if updated2.Priority != "urgent" {
		t.Errorf("Priority after estimate-only update = %q, want %q", updated2.Priority, "urgent")
	}
	if updated2.Estimate == nil || *updated2.Estimate != 7 {
		t.Errorf("Estimate after update = %v, want 7", updated2.Estimate)
	}
}
