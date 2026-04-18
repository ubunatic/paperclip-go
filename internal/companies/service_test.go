package companies_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ubunatic/paperclip-go/internal/agents"
	"github.com/ubunatic/paperclip-go/internal/companies"
	"github.com/ubunatic/paperclip-go/internal/issues"
	"github.com/ubunatic/paperclip-go/internal/testutil"
)

func TestCompanyCRUD(t *testing.T) {
	t.Run("create", func(t *testing.T) {
		s := testutil.NewStore(t)
		svc := companies.New(s)
		ctx := context.Background()

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
	})

	t.Run("get", func(t *testing.T) {
		s := testutil.NewStore(t)
		svc := companies.New(s)
		ctx := context.Background()

		c, err := svc.Create(ctx, "Acme Corp", "acme", "A test company")
		if err != nil {
			t.Fatalf("Create: %v", err)
		}

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
	})

	t.Run("get_not_found", func(t *testing.T) {
		s := testutil.NewStore(t)
		svc := companies.New(s)
		ctx := context.Background()

		_, err := svc.Get(ctx, "nonexistent-id")
		if !errors.Is(err, companies.ErrNotFound) {
			t.Fatalf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("list", func(t *testing.T) {
		s := testutil.NewStore(t)
		svc := companies.New(s)
		ctx := context.Background()

		_, err := svc.Create(ctx, "Acme Corp", "acme", "A test company")
		if err != nil {
			t.Fatalf("Create: %v", err)
		}

		_, err = svc.Create(ctx, "Beta Inc", "beta", "")
		if err != nil {
			t.Fatalf("Create Beta: %v", err)
		}

		list, err := svc.List(ctx)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(list) != 2 {
			t.Errorf("List len = %d, want 2", len(list))
		}
	})

	t.Run("list_empty", func(t *testing.T) {
		s := testutil.NewStore(t)
		svc := companies.New(s)
		ctx := context.Background()

		list, err := svc.List(ctx)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if list == nil {
			t.Error("List should return non-nil empty slice, got nil")
		}
		if len(list) != 0 {
			t.Errorf("List len = %d, want 0", len(list))
		}
	})

	t.Run("duplicate_shortname", func(t *testing.T) {
		s := testutil.NewStore(t)
		svc := companies.New(s)
		ctx := context.Background()

		_, err := svc.Create(ctx, "Acme Corp", "acme", "A test company")
		if err != nil {
			t.Fatalf("Create: %v", err)
		}

		_, err = svc.Create(ctx, "Acme Dup", "acme", "")
		if err == nil {
			t.Fatal("expected error on duplicate shortname")
		}
	})

	t.Run("delete", func(t *testing.T) {
		s := testutil.NewStore(t)
		svc := companies.New(s)
		ctx := context.Background()

		c, err := svc.Create(ctx, "Acme Corp", "acme", "A test company")
		if err != nil {
			t.Fatalf("Create: %v", err)
		}

		// Delete the company should succeed
		err = svc.Delete(ctx, c.ID)
		if err != nil {
			t.Fatalf("Delete: %v", err)
		}

		// Get should return ErrNotFound
		_, err = svc.Get(ctx, c.ID)
		if !errors.Is(err, companies.ErrNotFound) {
			t.Fatalf("Get after delete: expected ErrNotFound, got %v", err)
		}
	})

	t.Run("delete_not_found", func(t *testing.T) {
		s := testutil.NewStore(t)
		svc := companies.New(s)
		ctx := context.Background()

		err := svc.Delete(ctx, "nonexistent-id")
		if !errors.Is(err, companies.ErrNotFound) {
			t.Fatalf("Delete nonexistent: expected ErrNotFound, got %v", err)
		}
	})

	t.Run("delete_with_agents", func(t *testing.T) {
		s := testutil.NewStore(t)
		svc := companies.New(s)
		ctx := context.Background()

		c, err := svc.Create(ctx, "Acme Corp", "acme", "A test company")
		if err != nil {
			t.Fatalf("Create: %v", err)
		}

		// Create an agent in this company
		agentSvc := agents.New(s)
		_, err = agentSvc.Create(ctx, c.ID, "alice", "Alice", "manager", nil, "stub")
		if err != nil {
			t.Fatalf("Create agent: %v", err)
		}

		// Try to delete company - should fail with ErrHasDependents
		err = svc.Delete(ctx, c.ID)
		if !errors.Is(err, companies.ErrHasDependents) {
			t.Fatalf("Delete with agents: expected ErrHasDependents, got %v", err)
		}

		// Verify company still exists
		_, err = svc.Get(ctx, c.ID)
		if err != nil {
			t.Fatalf("Get after failed delete: %v", err)
		}
	})

	t.Run("delete_with_issues", func(t *testing.T) {
		s := testutil.NewStore(t)
		svc := companies.New(s)
		ctx := context.Background()

		c, err := svc.Create(ctx, "Acme Corp", "acme", "A test company")
		if err != nil {
			t.Fatalf("Create: %v", err)
		}

		// Create an issue in this company
		issueSvc := issues.New(s)
		_, err = issueSvc.Create(ctx, c.ID, "Test Issue", "Body", nil)
		if err != nil {
			t.Fatalf("Create issue: %v", err)
		}

		// Try to delete company - should fail with ErrHasDependents
		err = svc.Delete(ctx, c.ID)
		if !errors.Is(err, companies.ErrHasDependents) {
			t.Fatalf("Delete with issues: expected ErrHasDependents, got %v", err)
		}

		// Verify company still exists
		_, err = svc.Get(ctx, c.ID)
		if err != nil {
			t.Fatalf("Get after failed delete: %v", err)
		}
	})
}
