package companies_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ubunatic/paperclip-go/internal/activity"
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
		agentSvc := agents.New(s, activity.New(s))
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

	t.Run("delete_with_activity", func(t *testing.T) {
		s := testutil.NewStore(t)
		svc := companies.New(s)
		ctx := context.Background()

		c, err := svc.Create(ctx, "Acme Corp", "acme", "A test company")
		if err != nil {
			t.Fatalf("Create: %v", err)
		}

		// Create an activity log entry for this company
		_, err = s.DB.ExecContext(ctx,
			`INSERT INTO activity_log(id, company_id, actor_type, actor_id, action, entity_type, entity_id, meta_json, created_at)
			 VALUES (?, ?, 'system', 'system', 'created', 'company', ?, '{}', ?)`,
			"activity-1", c.ID, c.ID, "2024-01-01T00:00:00Z",
		)
		if err != nil {
			t.Fatalf("Create activity log: %v", err)
		}

		// Try to delete company - should fail with ErrHasDependents
		err = svc.Delete(ctx, c.ID)
		if !errors.Is(err, companies.ErrHasDependents) {
			t.Fatalf("Delete with activity: expected ErrHasDependents, got %v", err)
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
		_, err = issueSvc.Create(ctx, c.ID, "Test Issue", "Body", "default", "open", nil)
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

	t.Run("update_name", func(t *testing.T) {
		s := testutil.NewStore(t)
		svc := companies.New(s)
		ctx := context.Background()

		c, err := svc.Create(ctx, "Acme Corp", "acme", "Original description")
		if err != nil {
			t.Fatalf("Create: %v", err)
		}

		newName := "Acme Inc"
		updated, err := svc.Update(ctx, c.ID, &newName, nil)
		if err != nil {
			t.Fatalf("Update: %v", err)
		}
		if updated.Name != newName {
			t.Errorf("Updated.Name = %q, want %q", updated.Name, newName)
		}
		if updated.Description != "Original description" {
			t.Errorf("Description should be unchanged, got %q", updated.Description)
		}
	})

	t.Run("update_description", func(t *testing.T) {
		s := testutil.NewStore(t)
		svc := companies.New(s)
		ctx := context.Background()

		c, err := svc.Create(ctx, "Acme Corp", "acme", "Original description")
		if err != nil {
			t.Fatalf("Create: %v", err)
		}

		newDesc := "Updated description"
		updated, err := svc.Update(ctx, c.ID, nil, &newDesc)
		if err != nil {
			t.Fatalf("Update: %v", err)
		}
		if updated.Description != newDesc {
			t.Errorf("Updated.Description = %q, want %q", updated.Description, newDesc)
		}
		if updated.Name != "Acme Corp" {
			t.Errorf("Name should be unchanged, got %q", updated.Name)
		}
	})

	t.Run("update_both", func(t *testing.T) {
		s := testutil.NewStore(t)
		svc := companies.New(s)
		ctx := context.Background()

		c, err := svc.Create(ctx, "Acme Corp", "acme", "Original description")
		if err != nil {
			t.Fatalf("Create: %v", err)
		}

		newName := "Acme Inc"
		newDesc := "Updated description"
		updated, err := svc.Update(ctx, c.ID, &newName, &newDesc)
		if err != nil {
			t.Fatalf("Update: %v", err)
		}
		if updated.Name != newName {
			t.Errorf("Updated.Name = %q, want %q", updated.Name, newName)
		}
		if updated.Description != newDesc {
			t.Errorf("Updated.Description = %q, want %q", updated.Description, newDesc)
		}
	})

	t.Run("update_clear_description", func(t *testing.T) {
		s := testutil.NewStore(t)
		svc := companies.New(s)
		ctx := context.Background()

		c, err := svc.Create(ctx, "Acme Corp", "acme", "Original description")
		if err != nil {
			t.Fatalf("Create: %v", err)
		}

		emptyDesc := ""
		updated, err := svc.Update(ctx, c.ID, nil, &emptyDesc)
		if err != nil {
			t.Fatalf("Update: %v", err)
		}
		if updated.Description != "" {
			t.Errorf("Updated.Description should be empty, got %q", updated.Description)
		}
		if updated.Name != "Acme Corp" {
			t.Errorf("Name should be unchanged, got %q", updated.Name)
		}
	})

	t.Run("update_not_found", func(t *testing.T) {
		s := testutil.NewStore(t)
		svc := companies.New(s)
		ctx := context.Background()

		newName := "New Name"
		_, err := svc.Update(ctx, "nonexistent-id", &newName, nil)
		if !errors.Is(err, companies.ErrNotFound) {
			t.Fatalf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("update_no_fields", func(t *testing.T) {
		s := testutil.NewStore(t)
		svc := companies.New(s)
		ctx := context.Background()

		c, err := svc.Create(ctx, "Acme Corp", "acme", "Original description")
		if err != nil {
			t.Fatalf("Create: %v", err)
		}

		// Try to update with both fields nil - should return validation error
		_, err = svc.Update(ctx, c.ID, nil, nil)
		if err == nil {
			t.Fatal("expected error when updating with no fields provided")
		}
	})
}
