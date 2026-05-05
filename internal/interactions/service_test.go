package interactions_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ubunatic/paperclip-go/internal/activity"
	"github.com/ubunatic/paperclip-go/internal/agents"
	"github.com/ubunatic/paperclip-go/internal/companies"
	"github.com/ubunatic/paperclip-go/internal/domain"
	"github.com/ubunatic/paperclip-go/internal/interactions"
	"github.com/ubunatic/paperclip-go/internal/issues"
	"github.com/ubunatic/paperclip-go/internal/store"
	"github.com/ubunatic/paperclip-go/internal/testutil"
)

func setupTestData(t *testing.T, s *store.Store) (companyID, agentID, issueID string) {
	t.Helper()
	ctx := context.Background()

	// Create company
	companySvc := companies.New(s)
	company, err := companySvc.Create(ctx, "Test Corp", "test", "Test company")
	if err != nil {
		t.Fatalf("Create company: %v", err)
	}

	// Create agent
	agentSvc := agents.New(s, activity.New(s))
	agent, err := agentSvc.Create(ctx, company.ID, "alice", "Alice", "manager", nil, "stub")
	if err != nil {
		t.Fatalf("Create agent: %v", err)
	}

	// Create issue
	issueSvc := issues.New(s)
	issue, err := issueSvc.Create(ctx, company.ID, "Test Issue", "Test body", "default", "open", nil)
	if err != nil {
		t.Fatalf("Create issue: %v", err)
	}

	return company.ID, agent.ID, issue.ID
}

func TestInteractionCRUD(t *testing.T) {
	t.Run("create", func(t *testing.T) {
		s := testutil.NewStore(t)
		ctx := context.Background()
		companyID, agentID, issueID := setupTestData(t, s)

		svc := interactions.New(s)
		input := interactions.CreateInput{
			CompanyID:      companyID,
			IssueID:        issueID,
			AgentID:        &agentID,
			Kind:           "approval",
			IdempotencyKey: "key-001",
		}
		interaction, err := svc.Create(ctx, input)
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if interaction.ID == "" {
			t.Fatal("expected non-empty ID")
		}
		if interaction.Kind != "approval" {
			t.Errorf("Kind = %q, want %q", interaction.Kind, "approval")
		}
		if interaction.Status != domain.InteractionStatusPending {
			t.Errorf("Status = %q, want %q", interaction.Status, domain.InteractionStatusPending)
		}
		if interaction.IdempotencyKey != "key-001" {
			t.Errorf("IdempotencyKey = %q, want %q", interaction.IdempotencyKey, "key-001")
		}
	})

	t.Run("idempotency_dedup", func(t *testing.T) {
		s := testutil.NewStore(t)
		ctx := context.Background()
		companyID, agentID, issueID := setupTestData(t, s)

		svc := interactions.New(s)
		input := interactions.CreateInput{
			CompanyID:      companyID,
			IssueID:        issueID,
			AgentID:        &agentID,
			Kind:           "approval",
			IdempotencyKey: "key-001",
		}

		// Create first interaction
		interaction1, err := svc.Create(ctx, input)
		if err != nil {
			t.Fatalf("Create first: %v", err)
		}

		// Create with same issue_id and idempotency_key - should return existing
		interaction2, err := svc.Create(ctx, input)
		if err != nil {
			t.Fatalf("Create second (dedup): %v", err)
		}

		if interaction2.ID != interaction1.ID {
			t.Errorf("Dedup returned different ID: got %q, want %q", interaction2.ID, interaction1.ID)
		}
	})

	t.Run("get_by_id", func(t *testing.T) {
		s := testutil.NewStore(t)
		ctx := context.Background()
		companyID, agentID, issueID := setupTestData(t, s)

		svc := interactions.New(s)
		input := interactions.CreateInput{
			CompanyID:      companyID,
			IssueID:        issueID,
			AgentID:        &agentID,
			Kind:           "approval",
			IdempotencyKey: "key-001",
		}
		created, err := svc.Create(ctx, input)
		if err != nil {
			t.Fatalf("Create: %v", err)
		}

		got, err := svc.GetByID(ctx, created.ID)
		if err != nil {
			t.Fatalf("GetByID: %v", err)
		}
		if got.ID != created.ID {
			t.Errorf("GetByID.ID = %q, want %q", got.ID, created.ID)
		}
		if got.Kind != created.Kind {
			t.Errorf("GetByID.Kind = %q, want %q", got.Kind, created.Kind)
		}
	})

	t.Run("get_by_id_not_found", func(t *testing.T) {
		s := testutil.NewStore(t)
		ctx := context.Background()
		svc := interactions.New(s)

		_, err := svc.GetByID(ctx, "nonexistent-id")
		if !errors.Is(err, interactions.ErrNotFound) {
			t.Fatalf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("get_by_idempotency_key", func(t *testing.T) {
		s := testutil.NewStore(t)
		ctx := context.Background()
		companyID, agentID, issueID := setupTestData(t, s)

		svc := interactions.New(s)
		input := interactions.CreateInput{
			CompanyID:      companyID,
			IssueID:        issueID,
			AgentID:        &agentID,
			Kind:           "approval",
			IdempotencyKey: "key-001",
		}
		created, err := svc.Create(ctx, input)
		if err != nil {
			t.Fatalf("Create: %v", err)
		}

		got, err := svc.GetByIdempotencyKey(ctx, issueID, "key-001")
		if err != nil {
			t.Fatalf("GetByIdempotencyKey: %v", err)
		}
		if got.ID != created.ID {
			t.Errorf("GetByIdempotencyKey.ID = %q, want %q", got.ID, created.ID)
		}
	})

	t.Run("get_by_idempotency_key_not_found", func(t *testing.T) {
		s := testutil.NewStore(t)
		ctx := context.Background()
		_, _, issueID := setupTestData(t, s)

		svc := interactions.New(s)
		_, err := svc.GetByIdempotencyKey(ctx, issueID, "nonexistent-key")
		if !errors.Is(err, interactions.ErrNotFound) {
			t.Fatalf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("list_by_issue", func(t *testing.T) {
		s := testutil.NewStore(t)
		ctx := context.Background()
		companyID, agentID, issueID := setupTestData(t, s)

		svc := interactions.New(s)

		// Create multiple interactions
		for i := 1; i <= 3; i++ {
			input := interactions.CreateInput{
				CompanyID:      companyID,
				IssueID:        issueID,
				AgentID:        &agentID,
				Kind:           "approval",
				IdempotencyKey: "key-" + string(rune(i)),
			}
			_, err := svc.Create(ctx, input)
			if err != nil {
				t.Fatalf("Create %d: %v", i, err)
			}
		}

		items, err := svc.ListByIssue(ctx, issueID)
		if err != nil {
			t.Fatalf("ListByIssue: %v", err)
		}
		if len(items) != 3 {
			t.Errorf("ListByIssue len = %d, want 3", len(items))
		}
	})

	t.Run("list_by_issue_empty", func(t *testing.T) {
		s := testutil.NewStore(t)
		ctx := context.Background()
		_, _, issueID := setupTestData(t, s)

		svc := interactions.New(s)

		items, err := svc.ListByIssue(ctx, issueID)
		if err != nil {
			t.Fatalf("ListByIssue: %v", err)
		}
		if items == nil {
			t.Error("ListByIssue should return non-nil empty slice, got nil")
		}
		if len(items) != 0 {
			t.Errorf("ListByIssue len = %d, want 0", len(items))
		}
	})

	t.Run("resolve", func(t *testing.T) {
		s := testutil.NewStore(t)
		ctx := context.Background()
		companyID, agentID, issueID := setupTestData(t, s)

		svc := interactions.New(s)
		input := interactions.CreateInput{
			CompanyID:      companyID,
			IssueID:        issueID,
			AgentID:        &agentID,
			Kind:           "approval",
			IdempotencyKey: "key-001",
		}
		created, err := svc.Create(ctx, input)
		if err != nil {
			t.Fatalf("Create: %v", err)
		}

		result := "approved"
		resolved, err := svc.Resolve(ctx, created.ID, agentID, &result)
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if resolved.Status != domain.InteractionStatusResolved {
			t.Errorf("Status = %q, want %q", resolved.Status, domain.InteractionStatusResolved)
		}
		if resolved.ResolvedAt == nil {
			t.Error("ResolvedAt should not be nil")
		}
		if resolved.Result == nil || *resolved.Result != "approved" {
			t.Errorf("Result = %v, want %q", resolved.Result, "approved")
		}
		if resolved.ResolvedByAgentID == nil || *resolved.ResolvedByAgentID != agentID {
			t.Errorf("ResolvedByAgentID = %v, want %q", resolved.ResolvedByAgentID, agentID)
		}
	})

	t.Run("resolve_not_found", func(t *testing.T) {
		s := testutil.NewStore(t)
		ctx := context.Background()
		_, agentID, _ := setupTestData(t, s)

		svc := interactions.New(s)
		result := "approved"
		_, err := svc.Resolve(ctx, "nonexistent-id", agentID, &result)
		if !errors.Is(err, interactions.ErrNotFound) {
			t.Fatalf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("resolve_already_resolved", func(t *testing.T) {
		s := testutil.NewStore(t)
		ctx := context.Background()
		companyID, agentID, issueID := setupTestData(t, s)

		svc := interactions.New(s)
		input := interactions.CreateInput{
			CompanyID:      companyID,
			IssueID:        issueID,
			AgentID:        &agentID,
			Kind:           "approval",
			IdempotencyKey: "key-001",
		}
		created, err := svc.Create(ctx, input)
		if err != nil {
			t.Fatalf("Create: %v", err)
		}

		result := "approved"
		_, err = svc.Resolve(ctx, created.ID, agentID, &result)
		if err != nil {
			t.Fatalf("Resolve first: %v", err)
		}

		// Try to resolve again - should fail with ErrAlreadyResolved
		_, err = svc.Resolve(ctx, created.ID, agentID, &result)
		if !errors.Is(err, interactions.ErrAlreadyResolved) {
			t.Fatalf("expected ErrAlreadyResolved, got %v", err)
		}
	})

	t.Run("resolve_without_result", func(t *testing.T) {
		s := testutil.NewStore(t)
		ctx := context.Background()
		companyID, agentID, issueID := setupTestData(t, s)

		svc := interactions.New(s)
		input := interactions.CreateInput{
			CompanyID:      companyID,
			IssueID:        issueID,
			AgentID:        &agentID,
			Kind:           "approval",
			IdempotencyKey: "key-001",
		}
		created, err := svc.Create(ctx, input)
		if err != nil {
			t.Fatalf("Create: %v", err)
		}

		// Resolve without result (nil)
		resolved, err := svc.Resolve(ctx, created.ID, agentID, nil)
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if resolved.Status != domain.InteractionStatusResolved {
			t.Errorf("Status = %q, want %q", resolved.Status, domain.InteractionStatusResolved)
		}
		if resolved.Result != nil {
			t.Errorf("Result should be nil, got %v", resolved.Result)
		}
	})
}
