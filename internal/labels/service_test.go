package labels

import (
	"context"
	"testing"

	"github.com/ubunatic/paperclip-go/internal/testutil"
)

func TestCreate(t *testing.T) {
	store := testutil.NewStore(t)
	svc := New(store)

	companyID := testutil.CreateTestCompany(t, store)

	// Create a label
	label, err := svc.Create(context.Background(), companyID, "bug", "#FF0000")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if label.ID == "" {
		t.Error("expected non-empty ID")
	}
	if label.Name != "bug" {
		t.Errorf("Name = %q, want %q", label.Name, "bug")
	}
	if label.Color != "#FF0000" {
		t.Errorf("Color = %q, want %q", label.Color, "#FF0000")
	}
	if label.CompanyID != companyID {
		t.Errorf("CompanyID = %q, want %q", label.CompanyID, companyID)
	}

	// Attempt duplicate creation → error
	_, err = svc.Create(context.Background(), companyID, "bug", "#0000FF")
	if err == nil {
		t.Error("expected duplicate error")
	}
}

func TestGet(t *testing.T) {
	store := testutil.NewStore(t)
	svc := New(store)

	companyID := testutil.CreateTestCompany(t, store)

	// Create a label
	created, _ := svc.Create(context.Background(), companyID, "feature", "#00FF00")

	// Get it back
	retrieved, err := svc.Get(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if retrieved.ID != created.ID {
		t.Errorf("ID = %q, want %q", retrieved.ID, created.ID)
	}
	if retrieved.Name != "feature" {
		t.Errorf("Name = %q, want %q", retrieved.Name, "feature")
	}

	// Get nonexistent → error
	_, err = svc.Get(context.Background(), "nonexistent-id")
	if err != ErrNotFound {
		t.Errorf("Get nonexistent: %v, want ErrNotFound", err)
	}
}

func TestListByCompany(t *testing.T) {
	store := testutil.NewStore(t)
	svc := New(store)

	companyID := testutil.CreateTestCompany(t, store)

	// Create multiple labels
	svc.Create(context.Background(), companyID, "bug", "#FF0000")
	svc.Create(context.Background(), companyID, "feature", "#00FF00")
	svc.Create(context.Background(), companyID, "urgent", "#0000FF")

	// List them
	labels, err := svc.ListByCompany(context.Background(), companyID)
	if err != nil {
		t.Fatalf("ListByCompany: %v", err)
	}

	if len(labels) != 3 {
		t.Errorf("len(labels) = %d, want 3", len(labels))
	}

	// Check ordering (should be alphabetical)
	if labels[0].Name != "bug" {
		t.Errorf("labels[0].Name = %q, want %q", labels[0].Name, "bug")
	}
	if labels[1].Name != "feature" {
		t.Errorf("labels[1].Name = %q, want %q", labels[1].Name, "feature")
	}
	if labels[2].Name != "urgent" {
		t.Errorf("labels[2].Name = %q, want %q", labels[2].Name, "urgent")
	}
}

func TestDelete(t *testing.T) {
	store := testutil.NewStore(t)
	svc := New(store)

	companyID := testutil.CreateTestCompany(t, store)

	// Create and delete a label
	created, _ := svc.Create(context.Background(), companyID, "temp", "#AAAAAA")
	err := svc.Delete(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Verify it's deleted
	_, err = svc.Get(context.Background(), created.ID)
	if err != ErrNotFound {
		t.Errorf("after delete, Get returned %v, want ErrNotFound", err)
	}

	// Delete nonexistent → error
	err = svc.Delete(context.Background(), "nonexistent-id")
	if err != ErrNotFound {
		t.Errorf("Delete nonexistent: %v, want ErrNotFound", err)
	}
}

func TestGetByNameAndCompany(t *testing.T) {
	store := testutil.NewStore(t)
	svc := New(store)

	companyID := testutil.CreateTestCompany(t, store)

	// Create a label
	created, _ := svc.Create(context.Background(), companyID, "docs", "#999999")

	// Get it by name
	retrieved, err := svc.GetByNameAndCompany(context.Background(), companyID, "docs")
	if err != nil {
		t.Fatalf("GetByNameAndCompany: %v", err)
	}

	if retrieved == nil {
		t.Error("expected label, got nil")
	}
	if retrieved.ID != created.ID {
		t.Errorf("ID = %q, want %q", retrieved.ID, created.ID)
	}

	// Get nonexistent → nil, no error
	retrieved, err = svc.GetByNameAndCompany(context.Background(), companyID, "nonexistent")
	if err != nil {
		t.Errorf("GetByNameAndCompany nonexistent: %v", err)
	}
	if retrieved != nil {
		t.Errorf("expected nil, got %v", retrieved)
	}
}

func TestLinkToIssue(t *testing.T) {
	store := testutil.NewStore(t)
	svc := New(store)

	companyID := testutil.CreateTestCompany(t, store)
	issueID := testutil.CreateTestIssue(t, store, companyID)

	// Create a label
	label, _ := svc.Create(context.Background(), companyID, "critical", "#FF0000")

	// Link it to the issue
	err := svc.LinkToIssue(context.Background(), issueID, label.ID)
	if err != nil {
		t.Fatalf("LinkToIssue: %v", err)
	}

	// Link again (idempotent)
	err = svc.LinkToIssue(context.Background(), issueID, label.ID)
	if err != nil {
		t.Fatalf("LinkToIssue (duplicate): %v", err)
	}

	// Link to nonexistent issue → error
	err = svc.LinkToIssue(context.Background(), "nonexistent-issue", label.ID)
	if err != ErrIssueNotFound {
		t.Errorf("LinkToIssue nonexistent issue: %v, want ErrIssueNotFound", err)
	}
}

func TestUnlinkFromIssue(t *testing.T) {
	store := testutil.NewStore(t)
	svc := New(store)

	companyID := testutil.CreateTestCompany(t, store)
	issueID := testutil.CreateTestIssue(t, store, companyID)

	// Create and link a label
	label, _ := svc.Create(context.Background(), companyID, "high-priority", "#FF7700")
	svc.LinkToIssue(context.Background(), issueID, label.ID)

	// Unlink it
	err := svc.UnlinkFromIssue(context.Background(), issueID, label.ID)
	if err != nil {
		t.Fatalf("UnlinkFromIssue: %v", err)
	}

	// Unlink again (idempotent)
	err = svc.UnlinkFromIssue(context.Background(), issueID, label.ID)
	if err != nil {
		t.Fatalf("UnlinkFromIssue (duplicate): %v", err)
	}
}

func TestGetLabelsForIssue(t *testing.T) {
	store := testutil.NewStore(t)
	svc := New(store)

	companyID := testutil.CreateTestCompany(t, store)
	issueID := testutil.CreateTestIssue(t, store, companyID)

	// Create and link multiple labels
	label1, _ := svc.Create(context.Background(), companyID, "backend", "#0000FF")
	label2, _ := svc.Create(context.Background(), companyID, "frontend", "#00FF00")
	label3, _ := svc.Create(context.Background(), companyID, "urgent", "#FF0000")

	svc.LinkToIssue(context.Background(), issueID, label1.ID)
	svc.LinkToIssue(context.Background(), issueID, label2.ID)
	svc.LinkToIssue(context.Background(), issueID, label3.ID)

	// Get labels for issue
	labels, err := svc.GetLabelsForIssue(context.Background(), issueID)
	if err != nil {
		t.Fatalf("GetLabelsForIssue: %v", err)
	}

	if len(labels) != 3 {
		t.Errorf("len(labels) = %d, want 3", len(labels))
	}

	// Check ordering (alphabetical)
	if labels[0].Name != "backend" {
		t.Errorf("labels[0].Name = %q, want %q", labels[0].Name, "backend")
	}
	if labels[1].Name != "frontend" {
		t.Errorf("labels[1].Name = %q, want %q", labels[1].Name, "frontend")
	}
	if labels[2].Name != "urgent" {
		t.Errorf("labels[2].Name = %q, want %q", labels[2].Name, "urgent")
	}

	// Get labels for nonexistent issue (should return empty list, not error)
	labels, err = svc.GetLabelsForIssue(context.Background(), "nonexistent-issue")
	if err != nil {
		t.Fatalf("GetLabelsForIssue nonexistent: %v", err)
	}
	if len(labels) != 0 {
		t.Errorf("len(labels) for nonexistent issue = %d, want 0", len(labels))
	}
}
