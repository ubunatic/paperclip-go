package labels_test

import (
	"context"
	"testing"

	"github.com/ubunatic/paperclip-go/internal/companies"
	"github.com/ubunatic/paperclip-go/internal/labels"
	"github.com/ubunatic/paperclip-go/internal/testutil"
)

func TestCreateLabel(t *testing.T) {
	s := testutil.NewStore(t)
	companySvc := companies.New(s)
	labelSvc := labels.New(s)
	ctx := context.Background()

	// Create a company first
	company, err := companySvc.Create(ctx, "Test", "test", "desc")
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	label, err := labelSvc.Create(ctx, company.ID, "bug", "#ff0000")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if label.Name != "bug" || label.Color != "#ff0000" {
		t.Errorf("Create: got name=%q color=%q, want bug #ff0000", label.Name, label.Color)
	}
}

func TestCreateLabelDuplicate(t *testing.T) {
	s := testutil.NewStore(t)
	companySvc := companies.New(s)
	labelSvc := labels.New(s)
	ctx := context.Background()

	company, err := companySvc.Create(ctx, "Test", "test", "desc")
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	_, err = labelSvc.Create(ctx, company.ID, "bug", "#ff0000")
	if err != nil {
		t.Fatalf("Create 1st: %v", err)
	}

	_, err = labelSvc.Create(ctx, company.ID, "bug", "#00ff00")
	if err != labels.ErrDuplicate {
		t.Errorf("Create 2nd: got %v, want ErrDuplicate", err)
	}
}

func TestGetLabel(t *testing.T) {
	s := testutil.NewStore(t)
	companySvc := companies.New(s)
	labelSvc := labels.New(s)
	ctx := context.Background()

	company, err := companySvc.Create(ctx, "Test", "test", "desc")
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	created, err := labelSvc.Create(ctx, company.ID, "bug", "#ff0000")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := labelSvc.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != created.Name {
		t.Errorf("Get: got name=%q, want %q", got.Name, created.Name)
	}
}

func TestGetLabelNotFound(t *testing.T) {
	s := testutil.NewStore(t)
	svc := labels.New(s)
	ctx := context.Background()

	_, err := svc.Get(ctx, "nonexistent")
	if err != labels.ErrNotFound {
		t.Errorf("Get: got %v, want ErrNotFound", err)
	}
}

func TestListByCompany(t *testing.T) {
	s := testutil.NewStore(t)
	companySvc := companies.New(s)
	labelSvc := labels.New(s)
	ctx := context.Background()

	company, err := companySvc.Create(ctx, "Test", "test", "desc")
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	_, err = labelSvc.Create(ctx, company.ID, "bug", "#ff0000")
	if err != nil {
		t.Fatalf("Create 1: %v", err)
	}
	_, err = labelSvc.Create(ctx, company.ID, "feature", "#00ff00")
	if err != nil {
		t.Fatalf("Create 2: %v", err)
	}
	_, err = labelSvc.Create(ctx, company.ID, "docs", "#0000ff")
	if err != nil {
		t.Fatalf("Create 3: %v", err)
	}

	labels, err := labelSvc.ListByCompany(ctx, company.ID)
	if err != nil {
		t.Fatalf("ListByCompany: %v", err)
	}
	if len(labels) != 3 {
		t.Errorf("ListByCompany: got %d labels, want 3", len(labels))
	}
}

func TestDeleteLabel(t *testing.T) {
	s := testutil.NewStore(t)
	svc := labels.New(s)
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
	if err != labels.ErrNotFound {
		t.Errorf("Get after delete: got %v, want ErrNotFound", err)
	}
}

func TestDeleteLabelNotFound(t *testing.T) {
	s := testutil.NewStore(t)
	svc := labels.New(s)
	ctx := context.Background()

	err := svc.Delete(ctx, "nonexistent")
	if err != labels.ErrNotFound {
		t.Errorf("Delete: got %v, want ErrNotFound", err)
	}
}

func TestLinkToIssue(t *testing.T) {
	s := testutil.NewStore(t)
	svc := labels.New(s)
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

	err = svc.LinkToIssue(ctx, "i1", label.ID)
	if err != nil {
		t.Fatalf("LinkToIssue: %v", err)
	}

	labels, err := svc.GetLabelsForIssue(ctx, "i1")
	if err != nil {
		t.Fatalf("GetLabelsForIssue: %v", err)
	}
	if len(labels) != 1 {
		t.Errorf("GetLabelsForIssue: got %d labels, want 1", len(labels))
	}
}

func TestLinkToIssueIdempotent(t *testing.T) {
	s := testutil.NewStore(t)
	svc := labels.New(s)
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
	err = svc.LinkToIssue(ctx, "i1", label.ID)
	if err != nil {
		t.Fatalf("LinkToIssue 1: %v", err)
	}

	err = svc.LinkToIssue(ctx, "i1", label.ID)
	if err != nil {
		t.Fatalf("LinkToIssue 2: %v", err)
	}

	labels, err := svc.GetLabelsForIssue(ctx, "i1")
	if err != nil {
		t.Fatalf("GetLabelsForIssue: %v", err)
	}
	if len(labels) != 1 {
		t.Errorf("GetLabelsForIssue: got %d labels, want 1 (idempotent)", len(labels))
	}
}

func TestLinkToIssueIssueNotFound(t *testing.T) {
	s := testutil.NewStore(t)
	svc := labels.New(s)
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

	// Try to link to non-existent issue
	err = svc.LinkToIssue(ctx, "nonexistent-issue", label.ID)
	if err != labels.ErrIssueNotFound {
		t.Errorf("LinkToIssue: got %v, want ErrIssueNotFound", err)
	}
}

func TestLinkToIssueLabelNotFound(t *testing.T) {
	s := testutil.NewStore(t)
	svc := labels.New(s)
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

	// Create issue
	_, err = s.DB.ExecContext(ctx,
		`INSERT INTO issues(id, company_id, title, body, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"i1", "c1", "Title", "Body", "open", "2024-01-01T00:00:00Z", "2024-01-01T00:00:00Z",
	)
	if err != nil {
		t.Fatalf("setup issue: %v", err)
	}

	// Try to link to non-existent label
	err = svc.LinkToIssue(ctx, "i1", "nonexistent-label")
	if err != labels.ErrNotFound {
		t.Errorf("LinkToIssue: got %v, want ErrNotFound", err)
	}
}

func TestLinkToIssueCompanyMismatch(t *testing.T) {
	s := testutil.NewStore(t)
	svc := labels.New(s)
	ctx := context.Background()

	// Create two companies
	_, err := s.DB.ExecContext(ctx,
		`INSERT INTO companies(id, name, shortname, description, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		"c1", "Company1", "c1", "desc", "2024-01-01T00:00:00Z", "2024-01-01T00:00:00Z",
	)
	if err != nil {
		t.Fatalf("setup company c1: %v", err)
	}

	_, err = s.DB.ExecContext(ctx,
		`INSERT INTO companies(id, name, shortname, description, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		"c2", "Company2", "c2", "desc", "2024-01-01T00:00:00Z", "2024-01-01T00:00:00Z",
	)
	if err != nil {
		t.Fatalf("setup company c2: %v", err)
	}

	// Create label in c1
	label, err := svc.Create(ctx, "c1", "bug", "#ff0000")
	if err != nil {
		t.Fatalf("Create label: %v", err)
	}

	// Create issue in c2
	_, err = s.DB.ExecContext(ctx,
		`INSERT INTO issues(id, company_id, title, body, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"i1", "c2", "Title", "Body", "open", "2024-01-01T00:00:00Z", "2024-01-01T00:00:00Z",
	)
	if err != nil {
		t.Fatalf("setup issue: %v", err)
	}

	// Try to link label from c1 to issue in c2
	err = svc.LinkToIssue(ctx, "i1", label.ID)
	if err != labels.ErrCompanyMismatch {
		t.Errorf("LinkToIssue: got %v, want ErrCompanyMismatch", err)
	}
}

func TestUnlinkFromIssue(t *testing.T) {
	s := testutil.NewStore(t)
	svc := labels.New(s)
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
	err = svc.LinkToIssue(ctx, "i1", label.ID)
	if err != nil {
		t.Fatalf("LinkToIssue: %v", err)
	}

	err = svc.UnlinkFromIssue(ctx, "i1", label.ID)
	if err != nil {
		t.Fatalf("UnlinkFromIssue: %v", err)
	}

	labels, err := svc.GetLabelsForIssue(ctx, "i1")
	if err != nil {
		t.Fatalf("GetLabelsForIssue: %v", err)
	}
	if len(labels) != 0 {
		t.Errorf("GetLabelsForIssue: got %d labels, want 0", len(labels))
	}
}

func TestUnlinkFromIssueNotFound(t *testing.T) {
	s := testutil.NewStore(t)
	svc := labels.New(s)
	ctx := context.Background()

	err := svc.UnlinkFromIssue(ctx, "i1", "label1")
	if err != labels.ErrAssociationNotFound {
		t.Errorf("UnlinkFromIssue: got %v, want ErrAssociationNotFound", err)
	}
}

func TestDeleteLabelCascadesIssueLabels(t *testing.T) {
	s := testutil.NewStore(t)
	svc := labels.New(s)
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
	err = svc.LinkToIssue(ctx, "i1", label.ID)
	if err != nil {
		t.Fatalf("LinkToIssue: %v", err)
	}

	// Verify association exists
	labels, err := svc.GetLabelsForIssue(ctx, "i1")
	if err != nil {
		t.Fatalf("GetLabelsForIssue before delete: %v", err)
	}
	if len(labels) != 1 {
		t.Errorf("GetLabelsForIssue before delete: got %d, want 1", len(labels))
	}

	// Delete label
	err = svc.Delete(ctx, label.ID)
	if err != nil {
		t.Fatalf("Delete label: %v", err)
	}

	// Verify association is gone (cascade delete)
	labels, err = svc.GetLabelsForIssue(ctx, "i1")
	if err != nil {
		t.Fatalf("GetLabelsForIssue after delete: %v", err)
	}
	if len(labels) != 0 {
		t.Errorf("GetLabelsForIssue after delete: got %d, want 0 (should be cascaded)", len(labels))
	}
}
