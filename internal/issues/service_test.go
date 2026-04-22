package issues_test

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

func TestCreateIssue(t *testing.T) {
	s := testutil.NewStore(t)
	ctx := context.Background()

	// Create a company first
	companySvc := companies.New(s)
	company, err := companySvc.Create(ctx, "Test Corp", "test", "Test company")
	if err != nil {
		t.Fatalf("Create company: %v", err)
	}

	// Create an issue
	svc := issues.New(s)
	issue, err := svc.Create(ctx, company.ID, "Test Issue", "This is a test issue", "", nil)
	if err != nil {
		t.Fatalf("Create issue: %v", err)
	}

	if issue.ID == "" {
		t.Fatal("expected non-empty ID")
	}
	if issue.Title != "Test Issue" {
		t.Errorf("Title = %q, want %q", issue.Title, "Test Issue")
	}
	if issue.Body != "This is a test issue" {
		t.Errorf("Body = %q, want %q", issue.Body, "This is a test issue")
	}
	if issue.Status != "open" {
		t.Errorf("Status = %q, want %q", issue.Status, "open")
	}
	if issue.AssigneeID != nil {
		t.Errorf("AssigneeID = %v, want nil", issue.AssigneeID)
	}
	if issue.CheckedOutBy != nil {
		t.Errorf("CheckedOutBy = %v, want nil", issue.CheckedOutBy)
	}
	if issue.CheckedOutAt != nil {
		t.Errorf("CheckedOutAt = %v, want nil", issue.CheckedOutAt)
	}
	if issue.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
}

func TestGetIssue(t *testing.T) {
	s := testutil.NewStore(t)
	ctx := context.Background()

	// Create a company and issue
	companySvc := companies.New(s)
	company, err := companySvc.Create(ctx, "Test Corp", "test", "Test company")
	if err != nil {
		t.Fatalf("Create company: %v", err)
	}

	svc := issues.New(s)
	issue, err := svc.Create(ctx, company.ID, "Test Issue", "Test body", "", nil)
	if err != nil {
		t.Fatalf("Create issue: %v", err)
	}

	// Get the issue
	got, err := svc.Get(ctx, issue.ID)
	if err != nil {
		t.Fatalf("Get issue: %v", err)
	}
	if got.ID != issue.ID {
		t.Errorf("Get.ID = %q, want %q", got.ID, issue.ID)
	}
	if got.Title != issue.Title {
		t.Errorf("Get.Title = %q, want %q", got.Title, issue.Title)
	}
}

func TestGetIssueNotFound(t *testing.T) {
	s := testutil.NewStore(t)
	svc := issues.New(s)
	ctx := context.Background()

	_, err := svc.Get(ctx, "nonexistent-id")
	if !errors.Is(err, issues.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestListByCompany(t *testing.T) {
	s := testutil.NewStore(t)
	ctx := context.Background()

	// Create a company
	companySvc := companies.New(s)
	company, err := companySvc.Create(ctx, "Test Corp", "test", "Test company")
	if err != nil {
		t.Fatalf("Create company: %v", err)
	}

	// Create 3 issues
	svc := issues.New(s)
	_, err = svc.Create(ctx, company.ID, "Issue 1", "Body 1", "", nil)
	if err != nil {
		t.Fatalf("Create issue 1: %v", err)
	}
	_, err = svc.Create(ctx, company.ID, "Issue 2", "Body 2", "", nil)
	if err != nil {
		t.Fatalf("Create issue 2: %v", err)
	}
	_, err = svc.Create(ctx, company.ID, "Issue 3", "Body 3", "", nil)
	if err != nil {
		t.Fatalf("Create issue 3: %v", err)
	}

	// List by company
	list, err := svc.ListByCompany(ctx, company.ID)
	if err != nil {
		t.Fatalf("ListByCompany: %v", err)
	}
	if len(list) != 3 {
		t.Errorf("ListByCompany len = %d, want 3", len(list))
	}
}

func TestCheckout(t *testing.T) {
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

	issueSvc := issues.New(s)
	issue, err := issueSvc.Create(ctx, company.ID, "Test Issue", "Body", "", nil)
	if err != nil {
		t.Fatalf("Create issue: %v", err)
	}

	// First checkout should succeed
	err = issueSvc.Checkout(ctx, issue.ID, agent.ID)
	if err != nil {
		t.Fatalf("First checkout: %v", err)
	}

	// Verify status changed to in_progress
	got, err := issueSvc.Get(ctx, issue.ID)
	if err != nil {
		t.Fatalf("Get issue after checkout: %v", err)
	}
	if got.Status != "in_progress" {
		t.Errorf("Status after checkout = %q, want %q", got.Status, "in_progress")
	}
	if got.CheckedOutBy == nil || *got.CheckedOutBy != agent.ID {
		t.Errorf("CheckedOutBy = %v, want %q", got.CheckedOutBy, agent.ID)
	}
	if got.CheckedOutAt == nil {
		t.Error("CheckedOutAt should not be nil after checkout")
	}

	// Second checkout by same agent should succeed (idempotent)
	err = issueSvc.Checkout(ctx, issue.ID, agent.ID)
	if err != nil {
		t.Fatalf("Second checkout (same agent): expected nil, got %v", err)
	}

	// Checkout by different agent should fail with ErrCheckoutConflict
	agent2, err := agentSvc.Create(ctx, company.ID, "agent2", "Agent 2", "engineer", nil, "stub")
	if err != nil {
		t.Fatalf("Create agent2: %v", err)
	}
	err = issueSvc.Checkout(ctx, issue.ID, agent2.ID)
	if !errors.Is(err, issues.ErrCheckoutConflict) {
		t.Fatalf("Checkout by different agent: expected ErrCheckoutConflict, got %v", err)
	}
}

func TestRelease(t *testing.T) {
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

	issueSvc := issues.New(s)
	issue, err := issueSvc.Create(ctx, company.ID, "Test Issue", "Body", "", nil)
	if err != nil {
		t.Fatalf("Create issue: %v", err)
	}

	// Checkout the issue
	err = issueSvc.Checkout(ctx, issue.ID, agent.ID)
	if err != nil {
		t.Fatalf("Checkout: %v", err)
	}

	// Release the issue
	err = issueSvc.Release(ctx, issue.ID, agent.ID)
	if err != nil {
		t.Fatalf("Release: %v", err)
	}

	// Verify fields were cleared
	got, err := issueSvc.Get(ctx, issue.ID)
	if err != nil {
		t.Fatalf("Get issue after release: %v", err)
	}
	if got.CheckedOutBy != nil {
		t.Errorf("CheckedOutBy after release = %v, want nil", got.CheckedOutBy)
	}
	if got.CheckedOutAt != nil {
		t.Errorf("CheckedOutAt after release = %v, want nil", got.CheckedOutAt)
	}

	// Second checkout should succeed now
	err = issueSvc.Checkout(ctx, issue.ID, agent.ID)
	if err != nil {
		t.Fatalf("Second checkout after release: %v", err)
	}
}

func TestUpdate(t *testing.T) {
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

	issueSvc := issues.New(s)
	issue, err := issueSvc.Create(ctx, company.ID, "Test Issue", "Body", "", nil)
	if err != nil {
		t.Fatalf("Create issue: %v", err)
	}

	// Update status
	updated, err := issueSvc.Update(ctx, issue.ID, "done", nil)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Status != "done" {
		t.Errorf("Status after update = %q, want %q", updated.Status, "done")
	}

	// Update assignee
	updated, err = issueSvc.Update(ctx, issue.ID, "", &agent.ID)
	if err != nil {
		t.Fatalf("Update assignee: %v", err)
	}
	if updated.AssigneeID == nil || *updated.AssigneeID != agent.ID {
		t.Errorf("AssigneeID after update = %v, want %q", updated.AssigneeID, agent.ID)
	}
}

func TestDeleteIssue(t *testing.T) {
	s := testutil.NewStore(t)
	ctx := context.Background()

	// Create a company and issue
	companySvc := companies.New(s)
	company, err := companySvc.Create(ctx, "Test Corp", "test", "Test company")
	if err != nil {
		t.Fatalf("Create company: %v", err)
	}

	issueSvc := issues.New(s)
	issue, err := issueSvc.Create(ctx, company.ID, "Test Issue", "Body", "", nil)
	if err != nil {
		t.Fatalf("Create issue: %v", err)
	}

	// Delete the issue should succeed
	err = issueSvc.Delete(ctx, issue.ID)
	if err != nil {
		t.Fatalf("Delete issue: %v", err)
	}

	// Get should return ErrNotFound
	_, err = issueSvc.Get(ctx, issue.ID)
	if !errors.Is(err, issues.ErrNotFound) {
		t.Fatalf("Get after delete: expected ErrNotFound, got %v", err)
	}
}

func TestDeleteIssueNotFound(t *testing.T) {
	s := testutil.NewStore(t)
	ctx := context.Background()

	issueSvc := issues.New(s)
	err := issueSvc.Delete(ctx, "nonexistent-id")
	if !errors.Is(err, issues.ErrNotFound) {
		t.Fatalf("Delete nonexistent: expected ErrNotFound, got %v", err)
	}
}

func TestDeleteIssueWithComments(t *testing.T) {
	s := testutil.NewStore(t)
	ctx := context.Background()

	// Create a company and issue
	companySvc := companies.New(s)
	company, err := companySvc.Create(ctx, "Test Corp", "test", "Test company")
	if err != nil {
		t.Fatalf("Create company: %v", err)
	}

	issueSvc := issues.New(s)
	issue, err := issueSvc.Create(ctx, company.ID, "Test Issue", "Body", "", nil)
	if err != nil {
		t.Fatalf("Create issue: %v", err)
	}

	// Create a comment for this issue by directly inserting into the database
	_, err = s.DB.ExecContext(ctx,
		`INSERT INTO comments(id, issue_id, author_agent_id, author_kind, body, created_at)
		 VALUES (?, ?, NULL, 'system', 'Test comment', ?)`,
		"comment-1", issue.ID, "2024-01-01T00:00:00Z",
	)
	if err != nil {
		t.Fatalf("Create comment: %v", err)
	}

	// Delete the issue should succeed (comments cascade delete)
	err = issueSvc.Delete(ctx, issue.ID)
	if err != nil {
		t.Fatalf("Delete issue with comments: %v", err)
	}

	// Verify issue is gone
	_, err = issueSvc.Get(ctx, issue.ID)
	if !errors.Is(err, issues.ErrNotFound) {
		t.Fatalf("Get after delete: expected ErrNotFound, got %v", err)
	}

	// Verify comment is also gone
	var commentCount int
	err = s.DB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM comments WHERE issue_id = ?`,
		issue.ID,
	).Scan(&commentCount)
	if err != nil {
		t.Fatalf("Querying comments: %v", err)
	}
	if commentCount != 0 {
		t.Errorf("Expected 0 comments after delete, got %d", commentCount)
	}
}

func TestDeleteIssueCheckedOut(t *testing.T) {
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

	issueSvc := issues.New(s)
	issue, err := issueSvc.Create(ctx, company.ID, "Test Issue", "Body", "", nil)
	if err != nil {
		t.Fatalf("Create issue: %v", err)
	}

	// Checkout the issue
	err = issueSvc.Checkout(ctx, issue.ID, agent.ID)
	if err != nil {
		t.Fatalf("Checkout: %v", err)
	}

	// Delete should fail with ErrCheckoutConflictDelete
	err = issueSvc.Delete(ctx, issue.ID)
	if !errors.Is(err, issues.ErrCheckoutConflictDelete) {
		t.Fatalf("Delete checked-out: expected ErrCheckoutConflictDelete, got %v", err)
	}

	// Verify issue still exists
	_, err = issueSvc.Get(ctx, issue.ID)
	if err != nil {
		t.Fatalf("Get after failed delete: %v", err)
	}
}

func TestUpdateInvalidStatus(t *testing.T) {
	s := testutil.NewStore(t)
	ctx := context.Background()

	// Create a company and issue
	companySvc := companies.New(s)
	company, err := companySvc.Create(ctx, "Test Corp", "test", "Test company")
	if err != nil {
		t.Fatalf("Create company: %v", err)
	}

	issueSvc := issues.New(s)
	issue, err := issueSvc.Create(ctx, company.ID, "Test Issue", "Body", "", nil)
	if err != nil {
		t.Fatalf("Create issue: %v", err)
	}

	// Try to update with invalid status
	_, err = issueSvc.Update(ctx, issue.ID, "invalid_status", nil)
	if !errors.Is(err, issues.ErrInvalidStatus) {
		t.Fatalf("Update with invalid status: expected ErrInvalidStatus, got %v", err)
	}
}

func TestCreateValidStatus(t *testing.T) {
	s := testutil.NewStore(t)
	ctx := context.Background()

	// Create a company
	companySvc := companies.New(s)
	company, err := companySvc.Create(ctx, "Test Corp", "test", "Test company")
	if err != nil {
		t.Fatalf("Create company: %v", err)
	}

	// Create an issue with explicit valid status "blocked"
	svc := issues.New(s)
	issue, err := svc.Create(ctx, company.ID, "Test Issue", "Body", "blocked", nil)
	if err != nil {
		t.Fatalf("Create issue: %v", err)
	}

	if issue.Status != "blocked" {
		t.Errorf("Status = %q, want %q", issue.Status, "blocked")
	}
}

func TestCreateInvalidStatus(t *testing.T) {
	s := testutil.NewStore(t)
	ctx := context.Background()

	// Create a company
	companySvc := companies.New(s)
	company, err := companySvc.Create(ctx, "Test Corp", "test", "Test company")
	if err != nil {
		t.Fatalf("Create company: %v", err)
	}

	// Try to create an issue with invalid status "bogus"
	svc := issues.New(s)
	_, err = svc.Create(ctx, company.ID, "Test Issue", "Body", "bogus", nil)
	if !errors.Is(err, issues.ErrInvalidStatus) {
		t.Fatalf("Create with invalid status: expected ErrInvalidStatus, got %v", err)
	}
}

