package heartbeat_test

import (
	"context"
	"testing"

	"github.com/ubunatic/paperclip-go/internal/activity"
	"github.com/ubunatic/paperclip-go/internal/agents"
	"github.com/ubunatic/paperclip-go/internal/approvals"
	"github.com/ubunatic/paperclip-go/internal/comments"
	"github.com/ubunatic/paperclip-go/internal/companies"
	"github.com/ubunatic/paperclip-go/internal/heartbeat"
	"github.com/ubunatic/paperclip-go/internal/issues"
	"github.com/ubunatic/paperclip-go/internal/testutil"
)

func TestApprovalGateSkipsIssue(t *testing.T) {
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

	issueSvc := issues.New(s)
	issue, err := issueSvc.Create(ctx, company.ID, "Test issue", "Issue body", "default", "open", "", nil)
	if err != nil {
		t.Fatalf("Create issue: %v", err)
	}

	approvalSvc := approvals.New(s)
	_, err = approvalSvc.Create(ctx, company.ID, agent.ID, issue.ID, "human_approval", nil)
	if err != nil {
		t.Fatalf("Create approval: %v", err)
	}

	actLog := activity.New(s)
	commentSvc := comments.New(s)
	registry := heartbeat.NewDefaultRegistry()
	runner := heartbeat.New(s, agentSvc, issueSvc, commentSvc, actLog, registry, approvalSvc)

	run, err := runner.Run(ctx, agent.ID)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if run.Status != "success" {
		t.Errorf("Status = %q, want %q", run.Status, "success")
	}
	if run.IssueID != nil {
		t.Errorf("IssueID = %v, want nil (issue should be skipped due to pending approval)", run.IssueID)
	}
}

func TestApprovalGateAllowsIssueWithoutApprovals(t *testing.T) {
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

	issueSvc := issues.New(s)
	issue, err := issueSvc.Create(ctx, company.ID, "Test issue", "Issue body", "default", "open", "", nil)
	if err != nil {
		t.Fatalf("Create issue: %v", err)
	}

	approvalSvc := approvals.New(s)
	actLog := activity.New(s)
	commentSvc := comments.New(s)
	registry := heartbeat.NewDefaultRegistry()
	runner := heartbeat.New(s, agentSvc, issueSvc, commentSvc, actLog, registry, approvalSvc)

	run, err := runner.Run(ctx, agent.ID)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if run.Status != "success" {
		t.Errorf("Status = %q, want %q", run.Status, "success")
	}
	if run.IssueID == nil || *run.IssueID != issue.ID {
		t.Errorf("IssueID = %v, want %q (issue should be selected when no pending approvals)", run.IssueID, issue.ID)
	}
}
