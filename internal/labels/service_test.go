package labels

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/ubunatic/paperclip-go/internal/store"
)

func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	dsn := filepath.Join(t.TempDir(), "test.db")
	s, err := store.Open(dsn)
	if err != nil {
		t.Fatalf("newTestStore: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestCreateLabel(t *testing.T) {
	s := newTestStore(t)
	svc := New(s)
	ctx := context.Background()

	// Create a company first
	_, err := s.DB.ExecContext(ctx,
		`INSERT INTO companies(id, name, shortname, description, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		"c1", "Test", "test", "desc", "2024-01-01T00:00:00Z", "2024-01-01T00:00:00Z",
	)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	label, err := svc.Create(ctx, "c1", "bug", "#ff0000")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if label.Name != "bug" || label.Color != "#ff0000" {
		t.Errorf("Create: got name=%q color=%q, want bug #ff0000", label.Name, label.Color)
	}
}

func TestCreateLabelDuplicate(t *testing.T) {
	s := newTestStore(t)
	svc := New(s)
	ctx := context.Background()

	// Create a company first
	_, err := s.DB.ExecContext(ctx,
		`INSERT INTO companies(id, name, shortname, description, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		"c1", "Test", "test", "desc", "2024-01-01T00:00:00Z", "2024-01-01T00:00:00Z",
	)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	_, err = svc.Create(ctx, "c1", "bug", "#ff0000")
	if err != nil {
		t.Fatalf("Create 1st: %v", err)
	}

	_, err = svc.Create(ctx, "c1", "bug", "#00ff00")
	if err != ErrDuplicate {
		t.Errorf("Create 2nd: got %v, want ErrDuplicate", err)
	}
}

func TestGetLabel(t *testing.T) {
	s := newTestStore(t)
	svc := New(s)
	ctx := context.Background()

	// Create a company first
	_, err := s.DB.ExecContext(ctx,
		`INSERT INTO companies(id, name, shortname, description, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		"c1", "Test", "test", "desc", "2024-01-01T00:00:00Z", "2024-01-01T00:00:00Z",
	)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	created, err := svc.Create(ctx, "c1", "bug", "#ff0000")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := svc.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != created.Name {
		t.Errorf("Get: got name=%q, want %q", got.Name, created.Name)
	}
}

func TestGetLabelNotFound(t *testing.T) {
	s := newTestStore(t)
	svc := New(s)
	ctx := context.Background()

	_, err := svc.Get(ctx, "nonexistent")
	if err != ErrNotFound {
		t.Errorf("Get: got %v, want ErrNotFound", err)
	}
}

func TestListByCompany(t *testing.T) {
	s := newTestStore(t)
	svc := New(s)
	ctx := context.Background()

	// Create a company first
	_, err := s.DB.ExecContext(ctx,
		`INSERT INTO companies(id, name, shortname, description, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		"c1", "Test", "test", "desc", "2024-01-01T00:00:00Z", "2024-01-01T00:00:00Z",
	)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	_, err = svc.Create(ctx, "c1", "bug", "#ff0000")
	if err != nil {
		t.Fatalf("Create 1: %v", err)
	}
	_, err = svc.Create(ctx, "c1", "feature", "#00ff00")
	if err != nil {
		t.Fatalf("Create 2: %v", err)
	}
	_, err = svc.Create(ctx, "c1", "docs", "#0000ff")
	if err != nil {
		t.Fatalf("Create 3: %v", err)
	}

	labels, err := svc.ListByCompany(ctx, "c1")
	if err != nil {
		t.Fatalf("ListByCompany: %v", err)
	}
	if len(labels) != 3 {
		t.Errorf("ListByCompany: got %d labels, want 3", len(labels))
	}
}

func TestDeleteLabel(t *testing.T) {
	s := newTestStore(t)
	svc := New(s)
	ctx := context.Background()

	// Create a company first
	_, err := s.DB.ExecContext(ctx,
		`INSERT INTO companies(id, name, shortname, description, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		"c1", "Test", "test", "desc", "2024-01-01T00:00:00Z", "2024-01-01T00:00:00Z",
	)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	created, err := svc.Create(ctx, "c1", "bug", "#ff0000")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	err = svc.Delete(ctx, created.ID)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err = svc.Get(ctx, created.ID)
	if err != ErrNotFound {
		t.Errorf("Get after delete: got %v, want ErrNotFound", err)
	}
}

func TestDeleteLabelNotFound(t *testing.T) {
	s := newTestStore(t)
	svc := New(s)
	ctx := context.Background()

	err := svc.Delete(ctx, "nonexistent")
	if err != ErrNotFound {
		t.Errorf("Delete: got %v, want ErrNotFound", err)
	}
}

func TestAddToIssue(t *testing.T) {
	s := newTestStore(t)
	svc := New(s)
	ctx := context.Background()

	// Create company
	_, err := s.DB.ExecContext(ctx,
		`INSERT INTO companies(id, name, shortname, description, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		"c1", "Test", "test", "desc", "2024-01-01T00:00:00Z", "2024-01-01T00:00:00Z",
	)
	if err != nil {
		t.Fatalf("setup company: %v", err)
	}

	// Create label
	label, err := svc.Create(ctx, "c1", "bug", "#ff0000")
	if err != nil {
		t.Fatalf("Create label: %v", err)
	}

	// Create issue
	_, err = s.DB.ExecContext(ctx,
		`INSERT INTO issues(id, company_id, title, body, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"i1", "c1", "Title", "Body", "open", "2024-01-01T00:00:00Z", "2024-01-01T00:00:00Z",
	)
	if err != nil {
		t.Fatalf("setup issue: %v", err)
	}

	err = svc.AddToIssue(ctx, "i1", label.ID, "c1")
	if err != nil {
		t.Fatalf("AddToIssue: %v", err)
	}

	labels, err := svc.ListForIssue(ctx, "i1")
	if err != nil {
		t.Fatalf("ListForIssue: %v", err)
	}
	if len(labels) != 1 {
		t.Errorf("ListForIssue: got %d labels, want 1", len(labels))
	}
}

func TestAddToIssueIdempotent(t *testing.T) {
	s := newTestStore(t)
	svc := New(s)
	ctx := context.Background()

	// Create company
	_, err := s.DB.ExecContext(ctx,
		`INSERT INTO companies(id, name, shortname, description, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		"c1", "Test", "test", "desc", "2024-01-01T00:00:00Z", "2024-01-01T00:00:00Z",
	)
	if err != nil {
		t.Fatalf("setup company: %v", err)
	}

	// Create label
	label, err := svc.Create(ctx, "c1", "bug", "#ff0000")
	if err != nil {
		t.Fatalf("Create label: %v", err)
	}

	// Create issue
	_, err = s.DB.ExecContext(ctx,
		`INSERT INTO issues(id, company_id, title, body, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"i1", "c1", "Title", "Body", "open", "2024-01-01T00:00:00Z", "2024-01-01T00:00:00Z",
	)
	if err != nil {
		t.Fatalf("setup issue: %v", err)
	}

	// Add twice - should be idempotent
	err = svc.AddToIssue(ctx, "i1", label.ID, "c1")
	if err != nil {
		t.Fatalf("AddToIssue 1: %v", err)
	}

	err = svc.AddToIssue(ctx, "i1", label.ID, "c1")
	if err != nil {
		t.Fatalf("AddToIssue 2: %v", err)
	}

	labels, err := svc.ListForIssue(ctx, "i1")
	if err != nil {
		t.Fatalf("ListForIssue: %v", err)
	}
	if len(labels) != 1 {
		t.Errorf("ListForIssue: got %d labels, want 1 (idempotent)", len(labels))
	}
}

func TestRemoveFromIssue(t *testing.T) {
	s := newTestStore(t)
	svc := New(s)
	ctx := context.Background()

	// Create company
	_, err := s.DB.ExecContext(ctx,
		`INSERT INTO companies(id, name, shortname, description, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		"c1", "Test", "test", "desc", "2024-01-01T00:00:00Z", "2024-01-01T00:00:00Z",
	)
	if err != nil {
		t.Fatalf("setup company: %v", err)
	}

	// Create label
	label, err := svc.Create(ctx, "c1", "bug", "#ff0000")
	if err != nil {
		t.Fatalf("Create label: %v", err)
	}

	// Create issue
	_, err = s.DB.ExecContext(ctx,
		`INSERT INTO issues(id, company_id, title, body, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"i1", "c1", "Title", "Body", "open", "2024-01-01T00:00:00Z", "2024-01-01T00:00:00Z",
	)
	if err != nil {
		t.Fatalf("setup issue: %v", err)
	}

	// Add label, then remove it
	err = svc.AddToIssue(ctx, "i1", label.ID, "c1")
	if err != nil {
		t.Fatalf("AddToIssue: %v", err)
	}

	err = svc.RemoveFromIssue(ctx, "i1", label.ID)
	if err != nil {
		t.Fatalf("RemoveFromIssue: %v", err)
	}

	labels, err := svc.ListForIssue(ctx, "i1")
	if err != nil {
		t.Fatalf("ListForIssue: %v", err)
	}
	if len(labels) != 0 {
		t.Errorf("ListForIssue: got %d labels, want 0", len(labels))
	}
}

func TestRemoveFromIssueNotFound(t *testing.T) {
	s := newTestStore(t)
	svc := New(s)
	ctx := context.Background()

	err := svc.RemoveFromIssue(ctx, "i1", "label1")
	if err != ErrNotFound {
		t.Errorf("RemoveFromIssue: got %v, want ErrNotFound", err)
	}
}

func TestDeleteLabelCascadesIssueLabels(t *testing.T) {
	s := newTestStore(t)
	svc := New(s)
	ctx := context.Background()

	// Create company
	_, err := s.DB.ExecContext(ctx,
		`INSERT INTO companies(id, name, shortname, description, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		"c1", "Test", "test", "desc", "2024-01-01T00:00:00Z", "2024-01-01T00:00:00Z",
	)
	if err != nil {
		t.Fatalf("setup company: %v", err)
	}

	// Create label
	label, err := svc.Create(ctx, "c1", "bug", "#ff0000")
	if err != nil {
		t.Fatalf("Create label: %v", err)
	}

	// Create issue
	_, err = s.DB.ExecContext(ctx,
		`INSERT INTO issues(id, company_id, title, body, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"i1", "c1", "Title", "Body", "open", "2024-01-01T00:00:00Z", "2024-01-01T00:00:00Z",
	)
	if err != nil {
		t.Fatalf("setup issue: %v", err)
	}

	// Add label to issue
	err = svc.AddToIssue(ctx, "i1", label.ID, "c1")
	if err != nil {
		t.Fatalf("AddToIssue: %v", err)
	}

	// Verify association exists
	labels, err := svc.ListForIssue(ctx, "i1")
	if err != nil {
		t.Fatalf("ListForIssue before delete: %v", err)
	}
	if len(labels) != 1 {
		t.Errorf("ListForIssue before delete: got %d, want 1", len(labels))
	}

	// Delete label
	err = svc.Delete(ctx, label.ID)
	if err != nil {
		t.Fatalf("Delete label: %v", err)
	}

	// Verify association is gone (cascade delete)
	labels, err = svc.ListForIssue(ctx, "i1")
	if err != nil {
		t.Fatalf("ListForIssue after delete: %v", err)
	}
	if len(labels) != 0 {
		t.Errorf("ListForIssue after delete: got %d, want 0 (should be cascaded)", len(labels))
	}
}
