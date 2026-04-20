package comments_test

import (
	"context"
	"testing"

	"github.com/ubunatic/paperclip-go/internal/activity"
	"github.com/ubunatic/paperclip-go/internal/agents"
	"github.com/ubunatic/paperclip-go/internal/comments"
	"github.com/ubunatic/paperclip-go/internal/companies"
	"github.com/ubunatic/paperclip-go/internal/issues"
	"github.com/ubunatic/paperclip-go/internal/testutil"
)

func TestCreateComment(t *testing.T) {
	s := testutil.NewStore(t)
	ctx := context.Background()

	// Create a company and issue first
	companySvc := companies.New(s)
	company, err := companySvc.Create(ctx, "Test Corp", "test", "Test company")
	if err != nil {
		t.Fatalf("Create company: %v", err)
	}

	issueSvc := issues.New(s)
	issue, err := issueSvc.Create(ctx, company.ID, "Test Issue", "Test body", nil)
	if err != nil {
		t.Fatalf("Create issue: %v", err)
	}

	// Create a comment
	commentSvc := comments.New(s)
	comment, err := commentSvc.Create(ctx, issue.ID, nil, "system", "This is a system comment")
	if err != nil {
		t.Fatalf("Create comment: %v", err)
	}

	if comment.ID == "" {
		t.Fatal("expected non-empty ID")
	}
	if comment.IssueID != issue.ID {
		t.Errorf("IssueID = %q, want %q", comment.IssueID, issue.ID)
	}
	if comment.AuthorKind != "system" {
		t.Errorf("AuthorKind = %q, want %q", comment.AuthorKind, "system")
	}
	if comment.Body != "This is a system comment" {
		t.Errorf("Body = %q, want %q", comment.Body, "This is a system comment")
	}
	if comment.AuthorAgentID != nil {
		t.Errorf("AuthorAgentID = %v, want nil", comment.AuthorAgentID)
	}
	if comment.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
}

func TestCreateCommentWithAgent(t *testing.T) {
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
	issue, err := issueSvc.Create(ctx, company.ID, "Test Issue", "Test body", nil)
	if err != nil {
		t.Fatalf("Create issue: %v", err)
	}

	// Create a comment with an agent author
	commentSvc := comments.New(s)
	comment, err := commentSvc.Create(ctx, issue.ID, &agent.ID, "agent", "This is an agent comment")
	if err != nil {
		t.Fatalf("Create comment: %v", err)
	}

	if comment.AuthorAgentID == nil || *comment.AuthorAgentID != agent.ID {
		t.Errorf("AuthorAgentID = %v, want %q", comment.AuthorAgentID, agent.ID)
	}
	if comment.AuthorKind != "agent" {
		t.Errorf("AuthorKind = %q, want %q", comment.AuthorKind, "agent")
	}
}

func TestListByIssue(t *testing.T) {
	s := testutil.NewStore(t)
	ctx := context.Background()

	// Create a company and issue
	companySvc := companies.New(s)
	company, err := companySvc.Create(ctx, "Test Corp", "test", "Test company")
	if err != nil {
		t.Fatalf("Create company: %v", err)
	}

	issueSvc := issues.New(s)
	issue, err := issueSvc.Create(ctx, company.ID, "Test Issue", "Test body", nil)
	if err != nil {
		t.Fatalf("Create issue: %v", err)
	}

	// Create 3 comments
	commentSvc := comments.New(s)
	_, err = commentSvc.Create(ctx, issue.ID, nil, "system", "Comment 1")
	if err != nil {
		t.Fatalf("Create comment 1: %v", err)
	}
	_, err = commentSvc.Create(ctx, issue.ID, nil, "system", "Comment 2")
	if err != nil {
		t.Fatalf("Create comment 2: %v", err)
	}
	_, err = commentSvc.Create(ctx, issue.ID, nil, "system", "Comment 3")
	if err != nil {
		t.Fatalf("Create comment 3: %v", err)
	}

	// List comments for the issue
	comments_list, err := commentSvc.ListByIssue(ctx, issue.ID)
	if err != nil {
		t.Fatalf("ListByIssue: %v", err)
	}
	if len(comments_list) != 3 {
		t.Errorf("ListByIssue len = %d, want 3", len(comments_list))
	}

	// Verify all expected comments are present (without assuming strict order, since timestamps can be equal)
	expectedBodies := map[string]int{
		"Comment 1": 1,
		"Comment 2": 1,
		"Comment 3": 1,
	}
	for _, comment := range comments_list {
		expectedBodies[comment.Body]--
	}
	for body, remaining := range expectedBodies {
		if remaining != 0 {
			t.Errorf("ListByIssue missing or duplicated body %q (remaining=%d)", body, remaining)
		}
	}
}

func TestListByIssueEmpty(t *testing.T) {
	s := testutil.NewStore(t)
	ctx := context.Background()

	// Create a company and issue
	companySvc := companies.New(s)
	company, err := companySvc.Create(ctx, "Test Corp", "test", "Test company")
	if err != nil {
		t.Fatalf("Create company: %v", err)
	}

	issueSvc := issues.New(s)
	issue, err := issueSvc.Create(ctx, company.ID, "Test Issue", "Test body", nil)
	if err != nil {
		t.Fatalf("Create issue: %v", err)
	}

	// List comments for the issue (should be empty)
	commentSvc := comments.New(s)
	comments_list, err := commentSvc.ListByIssue(ctx, issue.ID)
	if err != nil {
		t.Fatalf("ListByIssue: %v", err)
	}
	if len(comments_list) != 0 {
		t.Errorf("ListByIssue len = %d, want 0", len(comments_list))
	}
}
