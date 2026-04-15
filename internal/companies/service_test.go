package companies_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ubunatic/paperclip-go/internal/companies"
	"github.com/ubunatic/paperclip-go/internal/testutil"
)

func TestCompanyCRUD(t *testing.T) {
	s := testutil.NewStore(t)
	svc := companies.New(s)
	ctx := context.Background()

	// Create
	c, err := svc.Create(ctx, "Acme Corp", "acme", "A test company")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if c.ID == "" {
		t.Fatal("expected non-empty ID")
	}
	if c.Name != "Acme Corp" {
		t.Errorf("Name = %q, want %q", c.Name, "Acme Corp")
	}
	if c.Shortname != "acme" {
		t.Errorf("Shortname = %q, want %q", c.Shortname, "acme")
	}
	if c.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}

	// Get by ID
	got, err := svc.Get(ctx, c.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != c.ID {
		t.Errorf("Get.ID = %q, want %q", got.ID, c.ID)
	}
	if got.Name != c.Name {
		t.Errorf("Get.Name = %q, want %q", got.Name, c.Name)
	}

	// Get not found
	_, err = svc.Get(ctx, "nonexistent-id")
	if !errors.Is(err, companies.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	// Create a second company
	_, err = svc.Create(ctx, "Beta Inc", "beta", "")
	if err != nil {
		t.Fatalf("Create Beta: %v", err)
	}

	// List
	list, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("List len = %d, want 2", len(list))
	}

	// Duplicate shortname should fail
	_, err = svc.Create(ctx, "Acme Dup", "acme", "")
	if err == nil {
		t.Fatal("expected error on duplicate shortname")
	}
}
