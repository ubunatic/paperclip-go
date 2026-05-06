package workspaces_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ubunatic/paperclip-go/internal/ids"
	"github.com/ubunatic/paperclip-go/internal/store"
	"github.com/ubunatic/paperclip-go/internal/testutil"
	"github.com/ubunatic/paperclip-go/internal/workspaces"
)

func TestCreate(t *testing.T) {
	store := testutil.NewStore(t)
	svc := workspaces.New(store)
	ctx := context.Background()

	// Setup: create company and agent
	companyID, agentID := setupTestData(t, store)

	// Create workspace
	workspace, err := svc.Create(ctx, companyID, agentID, "/path/to/workspace", nil, "active")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if workspace == nil {
		t.Fatal("expected workspace, got nil")
	}
	if workspace.ID == "" {
		t.Error("expected ID to be set")
	}
	if workspace.CompanyID != companyID {
		t.Errorf("expected companyId %s, got %s", companyID, workspace.CompanyID)
	}
	if workspace.AgentID != agentID {
		t.Errorf("expected agentId %s, got %s", agentID, workspace.AgentID)
	}
	if workspace.Path != "/path/to/workspace" {
		t.Errorf("expected path /path/to/workspace, got %s", workspace.Path)
	}
	if workspace.Status != "active" {
		t.Errorf("expected status active, got %s", workspace.Status)
	}
	if workspace.IssueID != nil {
		t.Error("expected issueId to be nil")
	}
	if workspace.CreatedAt.IsZero() {
		t.Error("expected createdAt to be set")
	}
	if workspace.UpdatedAt.IsZero() {
		t.Error("expected updatedAt to be set")
	}
}

func TestCreateDuplicate(t *testing.T) {
	store := testutil.NewStore(t)
	svc := workspaces.New(store)
	ctx := context.Background()

	companyID, agentID := setupTestData(t, store)

	// Create first workspace
	_, err := svc.Create(ctx, companyID, agentID, "/path/to/workspace", nil, "active")
	if err != nil {
		t.Fatalf("Create first: %v", err)
	}

	// Create second workspace with same agent and path
	_, err = svc.Create(ctx, companyID, agentID, "/path/to/workspace", nil, "active")
	if !errors.Is(err, workspaces.ErrDuplicate) {
		t.Errorf("expected ErrDuplicate, got %v", err)
	}
}

func TestGetByID(t *testing.T) {
	store := testutil.NewStore(t)
	svc := workspaces.New(store)
	ctx := context.Background()

	companyID, agentID := setupTestData(t, store)

	// Create workspace
	created, err := svc.Create(ctx, companyID, agentID, "/path/to/workspace", nil, "active")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Get workspace
	retrieved, err := svc.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}

	if retrieved.ID != created.ID {
		t.Errorf("expected ID %s, got %s", created.ID, retrieved.ID)
	}
	if retrieved.Path != "/path/to/workspace" {
		t.Errorf("expected path /path/to/workspace, got %s", retrieved.Path)
	}
}

func TestGetByIDNotFound(t *testing.T) {
	store := testutil.NewStore(t)
	svc := workspaces.New(store)
	ctx := context.Background()

	_, err := svc.GetByID(ctx, "nonexistent-id")
	if !errors.Is(err, workspaces.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestListByCompany(t *testing.T) {
	store := testutil.NewStore(t)
	svc := workspaces.New(store)
	ctx := context.Background()

	companyID, agentID := setupTestData(t, store)

	// Create multiple workspaces
	w1, err := svc.Create(ctx, companyID, agentID, "/path/1", nil, "active")
	if err != nil {
		t.Fatalf("Create 1: %v", err)
	}

	w2, err := svc.Create(ctx, companyID, agentID, "/path/2", nil, "active")
	if err != nil {
		t.Fatalf("Create 2: %v", err)
	}

	// List workspaces
	list, err := svc.ListByCompany(ctx, companyID)
	if err != nil {
		t.Fatalf("ListByCompany: %v", err)
	}

	if len(list) != 2 {
		t.Errorf("expected 2 workspaces, got %d", len(list))
	}

	// Check both workspaces are present
	idSet := map[string]bool{
		list[0].ID: true,
		list[1].ID: true,
	}
	if !idSet[w1.ID] || !idSet[w2.ID] {
		t.Errorf("expected workspaces %s and %s, got %s and %s", w1.ID, w2.ID, list[0].ID, list[1].ID)
	}
}

func TestDelete(t *testing.T) {
	store := testutil.NewStore(t)
	svc := workspaces.New(store)
	ctx := context.Background()

	companyID, agentID := setupTestData(t, store)

	// Create workspace
	created, err := svc.Create(ctx, companyID, agentID, "/path/to/workspace", nil, "active")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Delete workspace
	err = svc.Delete(ctx, created.ID)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Verify it's gone
	_, err = svc.GetByID(ctx, created.ID)
	if !errors.Is(err, workspaces.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestDeleteNotFound(t *testing.T) {
	store := testutil.NewStore(t)
	svc := workspaces.New(store)
	ctx := context.Background()

	err := svc.Delete(ctx, "nonexistent-id")
	if !errors.Is(err, workspaces.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestCreateWithIssueID(t *testing.T) {
	store := testutil.NewStore(t)
	svc := workspaces.New(store)
	ctx := context.Background()

	companyID, agentID := setupTestData(t, store)

	// Create an issue first
	issueID := "issue-test-" + ids.NewUUID()
	_, err := store.DB.ExecContext(ctx,
		`INSERT INTO issues(id, company_id, title, body, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		issueID, companyID, "Test Issue", "Test body", "open", "2024-01-01T00:00:00Z", "2024-01-01T00:00:00Z",
	)
	if err != nil {
		t.Fatalf("create issue: %v", err)
	}

	// Create workspace with issueId
	workspace, err := svc.Create(ctx, companyID, agentID, "/path/to/workspace", &issueID, "active")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if workspace.IssueID == nil {
		t.Error("expected issueId to be set")
	}
	if *workspace.IssueID != issueID {
		t.Errorf("expected issueId %s, got %s", issueID, *workspace.IssueID)
	}
}

// setupTestData creates company and agent for testing.
func setupTestData(t *testing.T, s *store.Store) (companyID, agentID string) {
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

	return companyID, agentID
}
