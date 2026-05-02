package approvals_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ubunatic/paperclip-go/internal/approvals"
	"github.com/ubunatic/paperclip-go/internal/domain"
	"github.com/ubunatic/paperclip-go/internal/ids"
	"github.com/ubunatic/paperclip-go/internal/store"
	"github.com/ubunatic/paperclip-go/internal/testutil"
)

func TestCreate(t *testing.T) {
	store := testutil.NewStore(t)
	svc := approvals.New(store)
	ctx := context.Background()

	// Setup: create company, agent, and issue
	companyID, agentID, issueID := setupTestData(t, store)

	// Create approval
	approval, err := svc.Create(ctx, companyID, agentID, issueID, "delete_file", nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if approval == nil {
		t.Fatal("expected approval, got nil")
	}
	if approval.ID == "" {
		t.Error("expected ID to be set")
	}
	if approval.Status != domain.ApprovalStatusPending {
		t.Errorf("expected status pending, got %v", approval.Status)
	}
	if approval.CompanyID != companyID {
		t.Errorf("expected companyId %s, got %s", companyID, approval.CompanyID)
	}
	if approval.AgentID != agentID {
		t.Errorf("expected agentId %s, got %s", agentID, approval.AgentID)
	}
	if approval.IssueID != issueID {
		t.Errorf("expected issueId %s, got %s", issueID, approval.IssueID)
	}
	if approval.Kind != "delete_file" {
		t.Errorf("expected kind delete_file, got %s", approval.Kind)
	}
	if approval.CreatedAt.IsZero() {
		t.Error("expected createdAt to be set")
	}
	if approval.ResolvedAt != nil {
		t.Error("expected resolvedAt to be nil")
	}
}

func TestCreateWithRequestBody(t *testing.T) {
	store := testutil.NewStore(t)
	svc := approvals.New(store)
	ctx := context.Background()

	companyID, agentID, issueID := setupTestData(t, store)

	requestBody := `{"file": "secrets.env"}`
	approval, err := svc.Create(ctx, companyID, agentID, issueID, "delete_file", &requestBody)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if approval.RequestBody == nil {
		t.Error("expected requestBody to be set")
	} else if *approval.RequestBody != requestBody {
		t.Errorf("expected requestBody %s, got %s", requestBody, *approval.RequestBody)
	}
}

func TestGetByID(t *testing.T) {
	store := testutil.NewStore(t)
	svc := approvals.New(store)
	ctx := context.Background()

	companyID, agentID, issueID := setupTestData(t, store)

	// Create approval
	created, err := svc.Create(ctx, companyID, agentID, issueID, "test_kind", nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Get approval
	retrieved, err := svc.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}

	if retrieved.ID != created.ID {
		t.Errorf("expected ID %s, got %s", created.ID, retrieved.ID)
	}
	if retrieved.Status != domain.ApprovalStatusPending {
		t.Errorf("expected status pending, got %v", retrieved.Status)
	}
}

func TestGetByIDNotFound(t *testing.T) {
	store := testutil.NewStore(t)
	svc := approvals.New(store)
	ctx := context.Background()

	_, err := svc.GetByID(ctx, "nonexistent-id")
	if !errors.Is(err, approvals.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestListByCompany(t *testing.T) {
	store := testutil.NewStore(t)
	svc := approvals.New(store)
	ctx := context.Background()

	companyID, agentID, issueID := setupTestData(t, store)

	// Create multiple approvals
	a1, err := svc.Create(ctx, companyID, agentID, issueID, "kind1", nil)
	if err != nil {
		t.Fatalf("Create 1: %v", err)
	}

	a2, err := svc.Create(ctx, companyID, agentID, issueID, "kind2", nil)
	if err != nil {
		t.Fatalf("Create 2: %v", err)
	}

	// List approvals
	list, err := svc.ListByCompany(ctx, companyID)
	if err != nil {
		t.Fatalf("ListByCompany: %v", err)
	}

	if len(list) != 2 {
		t.Errorf("expected 2 approvals, got %d", len(list))
	}

	// Check both approvals are present (order may vary for same-second creation)
	ids := map[string]bool{
		list[0].ID: true,
		list[1].ID: true,
	}
	if !ids[a1.ID] || !ids[a2.ID] {
		t.Errorf("expected approvals %s and %s, got %s and %s", a1.ID, a2.ID, list[0].ID, list[1].ID)
	}
}

func TestListByCompanyEmpty(t *testing.T) {
	store := testutil.NewStore(t)
	svc := approvals.New(store)
	ctx := context.Background()

	list, err := svc.ListByCompany(ctx, "nonexistent-company")
	if err != nil {
		t.Fatalf("ListByCompany: %v", err)
	}

	if len(list) != 0 {
		t.Errorf("expected empty list, got %d approvals", len(list))
	}
}

func TestApprove(t *testing.T) {
	store := testutil.NewStore(t)
	svc := approvals.New(store)
	ctx := context.Background()

	companyID, agentID, issueID := setupTestData(t, store)

	created, err := svc.Create(ctx, companyID, agentID, issueID, "test_kind", nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Approve
	approved, err := svc.Approve(ctx, created.ID)
	if err != nil {
		t.Fatalf("Approve: %v", err)
	}

	if approved.Status != domain.ApprovalStatusApproved {
		t.Errorf("expected status approved, got %v", approved.Status)
	}
	if approved.ResolvedAt == nil {
		t.Error("expected resolvedAt to be set")
	}
}

func TestReject(t *testing.T) {
	store := testutil.NewStore(t)
	svc := approvals.New(store)
	ctx := context.Background()

	companyID, agentID, issueID := setupTestData(t, store)

	created, err := svc.Create(ctx, companyID, agentID, issueID, "test_kind", nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Reject
	rejected, err := svc.Reject(ctx, created.ID)
	if err != nil {
		t.Fatalf("Reject: %v", err)
	}

	if rejected.Status != domain.ApprovalStatusRejected {
		t.Errorf("expected status rejected, got %v", rejected.Status)
	}
	if rejected.ResolvedAt == nil {
		t.Error("expected resolvedAt to be set")
	}
}

func TestDoubleApproveReturnsError(t *testing.T) {
	store := testutil.NewStore(t)
	svc := approvals.New(store)
	ctx := context.Background()

	companyID, agentID, issueID := setupTestData(t, store)

	created, err := svc.Create(ctx, companyID, agentID, issueID, "test_kind", nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// First approve
	_, err = svc.Approve(ctx, created.ID)
	if err != nil {
		t.Fatalf("Approve: %v", err)
	}

	// Second approve should fail
	_, err = svc.Approve(ctx, created.ID)
	if !errors.Is(err, approvals.ErrAlreadyResolved) {
		t.Errorf("expected ErrAlreadyResolved, got %v", err)
	}
}

func TestDoubleRejectReturnsError(t *testing.T) {
	store := testutil.NewStore(t)
	svc := approvals.New(store)
	ctx := context.Background()

	companyID, agentID, issueID := setupTestData(t, store)

	created, err := svc.Create(ctx, companyID, agentID, issueID, "test_kind", nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// First reject
	_, err = svc.Reject(ctx, created.ID)
	if err != nil {
		t.Fatalf("Reject: %v", err)
	}

	// Second reject should fail
	_, err = svc.Reject(ctx, created.ID)
	if !errors.Is(err, approvals.ErrAlreadyResolved) {
		t.Errorf("expected ErrAlreadyResolved, got %v", err)
	}
}

func TestApproveAfterRejectReturnsError(t *testing.T) {
	store := testutil.NewStore(t)
	svc := approvals.New(store)
	ctx := context.Background()

	companyID, agentID, issueID := setupTestData(t, store)

	created, err := svc.Create(ctx, companyID, agentID, issueID, "test_kind", nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Reject
	_, err = svc.Reject(ctx, created.ID)
	if err != nil {
		t.Fatalf("Reject: %v", err)
	}

	// Then try to approve
	_, err = svc.Approve(ctx, created.ID)
	if !errors.Is(err, approvals.ErrAlreadyResolved) {
		t.Errorf("expected ErrAlreadyResolved, got %v", err)
	}
}

func TestApproveNonexistentReturnsNotFound(t *testing.T) {
	store := testutil.NewStore(t)
	svc := approvals.New(store)
	ctx := context.Background()

	_, err := svc.Approve(ctx, "nonexistent-id")
	if !errors.Is(err, approvals.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestRejectNonexistentReturnsNotFound(t *testing.T) {
	store := testutil.NewStore(t)
	svc := approvals.New(store)
	ctx := context.Background()

	_, err := svc.Reject(ctx, "nonexistent-id")
	if !errors.Is(err, approvals.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// setupTestData creates company, agent, and issue for testing.
func setupTestData(t *testing.T, s *store.Store) (companyID, agentID, issueID string) {
	t.Helper()
	ctx := context.Background()

	// Create company
	companyID = "company-test-" + ids.NewUUID()
	_, err := s.DB.ExecContext(ctx,
		`INSERT INTO companies(id, name, shortname, description, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		companyID, "Test Company", "test", "Test", "2024-01-01T00:00:00Z", "2024-01-01T00:00:00Z",
	)
	if err != nil {
		t.Fatalf("create company: %v", err)
	}

	// Create agent
	agentID = "agent-test-" + ids.NewUUID()
	_, err = s.DB.ExecContext(ctx,
		`INSERT INTO agents(id, company_id, shortname, display_name, role, adapter, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		agentID, companyID, "test-agent", "Test Agent", "test", "stub", "2024-01-01T00:00:00Z", "2024-01-01T00:00:00Z",
	)
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}

	// Create issue
	issueID = "issue-test-" + ids.NewUUID()
	_, err = s.DB.ExecContext(ctx,
		`INSERT INTO issues(id, company_id, title, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		issueID, companyID, "Test Issue", "open", "2024-01-01T00:00:00Z", "2024-01-01T00:00:00Z",
	)
	if err != nil {
		t.Fatalf("create issue: %v", err)
	}

	return companyID, agentID, issueID
}
