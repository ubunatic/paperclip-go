package secrets_test

import (
	"context"
	"testing"

	"github.com/ubunatic/paperclip-go/internal/companies"
	"github.com/ubunatic/paperclip-go/internal/secrets"
	"github.com/ubunatic/paperclip-go/internal/testutil"
)

func TestCreate(t *testing.T) {
	s := testutil.NewStore(t)
	svc := secrets.New(s)
	companySvc := companies.New(s)
	ctx := context.Background()

	// Create a company first
	company, err := companySvc.Create(ctx, "Test Corp", "test", "test company")
	if err != nil {
		t.Fatalf("creating company: %v", err)
	}

	// Create a secret
	secret, err := svc.Create(ctx, company.ID, "API_KEY", "secret123")
	if err != nil {
		t.Fatalf("creating secret: %v", err)
	}

	if secret.ID == "" {
		t.Errorf("secret.ID is empty")
	}
	if secret.CompanyID != company.ID {
		t.Errorf("secret.CompanyID = %q, want %q", secret.CompanyID, company.ID)
	}
	if secret.Name != "API_KEY" {
		t.Errorf("secret.Name = %q, want %q", secret.Name, "API_KEY")
	}
	if secret.Value != "secret123" {
		t.Errorf("secret.Value = %q, want %q", secret.Value, "secret123")
	}
	if secret.CreatedAt.IsZero() {
		t.Errorf("secret.CreatedAt is zero")
	}
	if secret.UpdatedAt.IsZero() {
		t.Errorf("secret.UpdatedAt is zero")
	}
}

func TestCreateDuplicate(t *testing.T) {
	s := testutil.NewStore(t)
	svc := secrets.New(s)
	companySvc := companies.New(s)
	ctx := context.Background()

	company, err := companySvc.Create(ctx, "Test Corp", "test", "test company")
	if err != nil {
		t.Fatalf("creating company: %v", err)
	}

	// Create first secret
	_, err = svc.Create(ctx, company.ID, "API_KEY", "secret123")
	if err != nil {
		t.Fatalf("creating first secret: %v", err)
	}

	// Try to create another with same name
	_, err = svc.Create(ctx, company.ID, "API_KEY", "different_value")
	if err != secrets.ErrDuplicate {
		t.Errorf("creating duplicate secret: got %v, want ErrDuplicate", err)
	}
}

func TestGetByID(t *testing.T) {
	s := testutil.NewStore(t)
	svc := secrets.New(s)
	companySvc := companies.New(s)
	ctx := context.Background()

	company, err := companySvc.Create(ctx, "Test Corp", "test", "test company")
	if err != nil {
		t.Fatalf("creating company: %v", err)
	}

	created, err := svc.Create(ctx, company.ID, "DB_URL", "postgres://...")
	if err != nil {
		t.Fatalf("creating secret: %v", err)
	}

	retrieved, err := svc.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("getting secret by ID: %v", err)
	}

	if retrieved.ID != created.ID {
		t.Errorf("retrieved.ID = %q, want %q", retrieved.ID, created.ID)
	}
	if retrieved.Value != "postgres://..." {
		t.Errorf("retrieved.Value = %q, want %q", retrieved.Value, "postgres://...")
	}
}

func TestGetByIDNotFound(t *testing.T) {
	s := testutil.NewStore(t)
	svc := secrets.New(s)
	ctx := context.Background()

	_, err := svc.GetByID(ctx, "nonexistent-id")
	if err != secrets.ErrNotFound {
		t.Errorf("getting nonexistent secret: got %v, want ErrNotFound", err)
	}
}

func TestListByCompany(t *testing.T) {
	s := testutil.NewStore(t)
	svc := secrets.New(s)
	companySvc := companies.New(s)
	ctx := context.Background()

	company, err := companySvc.Create(ctx, "Test Corp", "test", "test company")
	if err != nil {
		t.Fatalf("creating company: %v", err)
	}

	// Create 2 secrets
	_, err = svc.Create(ctx, company.ID, "SECRET1", "value1")
	if err != nil {
		t.Fatalf("creating secret 1: %v", err)
	}
	_, err = svc.Create(ctx, company.ID, "SECRET2", "value2")
	if err != nil {
		t.Fatalf("creating secret 2: %v", err)
	}

	// List should return 2 summaries
	summaries, err := svc.ListByCompany(ctx, company.ID)
	if err != nil {
		t.Fatalf("listing secrets: %v", err)
	}

	if len(summaries) != 2 {
		t.Errorf("list length = %d, want 2", len(summaries))
	}

	// Verify no value field in summary
	for _, summary := range summaries {
		if summary.ID == "" {
			t.Errorf("summary.ID is empty")
		}
		// SecretSummary doesn't have Value field
	}
}

func TestListByCompanyEmpty(t *testing.T) {
	s := testutil.NewStore(t)
	svc := secrets.New(s)
	companySvc := companies.New(s)
	ctx := context.Background()

	company, err := companySvc.Create(ctx, "Test Corp", "test", "test company")
	if err != nil {
		t.Fatalf("creating company: %v", err)
	}

	summaries, err := svc.ListByCompany(ctx, company.ID)
	if err != nil {
		t.Fatalf("listing secrets: %v", err)
	}

	if summaries == nil {
		t.Errorf("list is nil, want empty slice")
	}
	if len(summaries) != 0 {
		t.Errorf("list length = %d, want 0", len(summaries))
	}
}

func TestUpdateName(t *testing.T) {
	s := testutil.NewStore(t)
	svc := secrets.New(s)
	companySvc := companies.New(s)
	ctx := context.Background()

	company, err := companySvc.Create(ctx, "Test Corp", "test", "test company")
	if err != nil {
		t.Fatalf("creating company: %v", err)
	}

	created, err := svc.Create(ctx, company.ID, "OLD_NAME", "value123")
	if err != nil {
		t.Fatalf("creating secret: %v", err)
	}

	newName := "NEW_NAME"
	updated, err := svc.Update(ctx, created.ID, &newName, nil)
	if err != nil {
		t.Fatalf("updating secret name: %v", err)
	}

	if updated.Name != "NEW_NAME" {
		t.Errorf("updated.Name = %q, want %q", updated.Name, "NEW_NAME")
	}
	if updated.Value != "value123" {
		t.Errorf("updated.Value = %q, want %q", updated.Value, "value123")
	}
}

func TestUpdateValue(t *testing.T) {
	s := testutil.NewStore(t)
	svc := secrets.New(s)
	companySvc := companies.New(s)
	ctx := context.Background()

	company, err := companySvc.Create(ctx, "Test Corp", "test", "test company")
	if err != nil {
		t.Fatalf("creating company: %v", err)
	}

	created, err := svc.Create(ctx, company.ID, "API_KEY", "old_value")
	if err != nil {
		t.Fatalf("creating secret: %v", err)
	}

	newValue := "new_value"
	updated, err := svc.Update(ctx, created.ID, nil, &newValue)
	if err != nil {
		t.Fatalf("updating secret value: %v", err)
	}

	if updated.Name != "API_KEY" {
		t.Errorf("updated.Name = %q, want %q", updated.Name, "API_KEY")
	}
	if updated.Value != "new_value" {
		t.Errorf("updated.Value = %q, want %q", updated.Value, "new_value")
	}
}

func TestUpdateBoth(t *testing.T) {
	s := testutil.NewStore(t)
	svc := secrets.New(s)
	companySvc := companies.New(s)
	ctx := context.Background()

	company, err := companySvc.Create(ctx, "Test Corp", "test", "test company")
	if err != nil {
		t.Fatalf("creating company: %v", err)
	}

	created, err := svc.Create(ctx, company.ID, "OLD_NAME", "old_value")
	if err != nil {
		t.Fatalf("creating secret: %v", err)
	}

	newName := "NEW_NAME"
	newValue := "new_value"
	updated, err := svc.Update(ctx, created.ID, &newName, &newValue)
	if err != nil {
		t.Fatalf("updating secret: %v", err)
	}

	if updated.Name != "NEW_NAME" {
		t.Errorf("updated.Name = %q, want %q", updated.Name, "NEW_NAME")
	}
	if updated.Value != "new_value" {
		t.Errorf("updated.Value = %q, want %q", updated.Value, "new_value")
	}
}

func TestUpdateNotFound(t *testing.T) {
	s := testutil.NewStore(t)
	svc := secrets.New(s)
	ctx := context.Background()

	newName := "NEW_NAME"
	_, err := svc.Update(ctx, "nonexistent-id", &newName, nil)
	if err != secrets.ErrNotFound {
		t.Errorf("updating nonexistent secret: got %v, want ErrNotFound", err)
	}
}

func TestUpdateDuplicate(t *testing.T) {
	s := testutil.NewStore(t)
	svc := secrets.New(s)
	companySvc := companies.New(s)
	ctx := context.Background()

	company, err := companySvc.Create(ctx, "Test Corp", "test", "test company")
	if err != nil {
		t.Fatalf("creating company: %v", err)
	}

	// Create two secrets
	secret1, err := svc.Create(ctx, company.ID, "SECRET1", "value1")
	if err != nil {
		t.Fatalf("creating secret 1: %v", err)
	}
	_, err = svc.Create(ctx, company.ID, "SECRET2", "value2")
	if err != nil {
		t.Fatalf("creating secret 2: %v", err)
	}

	// Try to rename secret1 to SECRET2 (already exists)
	newName := "SECRET2"
	_, err = svc.Update(ctx, secret1.ID, &newName, nil)
	if err != secrets.ErrDuplicate {
		t.Errorf("updating to duplicate name: got %v, want ErrDuplicate", err)
	}
}

func TestDelete(t *testing.T) {
	s := testutil.NewStore(t)
	svc := secrets.New(s)
	companySvc := companies.New(s)
	ctx := context.Background()

	company, err := companySvc.Create(ctx, "Test Corp", "test", "test company")
	if err != nil {
		t.Fatalf("creating company: %v", err)
	}

	created, err := svc.Create(ctx, company.ID, "API_KEY", "secret123")
	if err != nil {
		t.Fatalf("creating secret: %v", err)
	}

	err = svc.Delete(ctx, created.ID)
	if err != nil {
		t.Fatalf("deleting secret: %v", err)
	}

	// Verify it's gone
	_, err = svc.GetByID(ctx, created.ID)
	if err != secrets.ErrNotFound {
		t.Errorf("after delete, GetByID: got %v, want ErrNotFound", err)
	}
}

func TestDeleteNotFound(t *testing.T) {
	s := testutil.NewStore(t)
	svc := secrets.New(s)
	ctx := context.Background()

	err := svc.Delete(ctx, "nonexistent-id")
	if err != secrets.ErrNotFound {
		t.Errorf("deleting nonexistent secret: got %v, want ErrNotFound", err)
	}
}

func TestUpdateNoFields(t *testing.T) {
	s := testutil.NewStore(t)
	svc := secrets.New(s)
	companySvc := companies.New(s)
	ctx := context.Background()

	company, err := companySvc.Create(ctx, "Test Corp", "test", "test company")
	if err != nil {
		t.Fatalf("creating company: %v", err)
	}

	created, err := svc.Create(ctx, company.ID, "API_KEY", "secret123")
	if err != nil {
		t.Fatalf("creating secret: %v", err)
	}

	// Try to update with no fields provided
	_, err = svc.Update(ctx, created.ID, nil, nil)
	if err == nil {
		t.Errorf("updating with no fields: expected error, got nil")
	}
}
