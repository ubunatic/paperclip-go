package settings_test

import (
	"context"
	"testing"

	"github.com/ubunatic/paperclip-go/internal/settings"
	"github.com/ubunatic/paperclip-go/internal/testutil"
)

func TestGetAllDefaults(t *testing.T) {
	s := testutil.NewStore(t)
	svc := settings.New(s)
	ctx := context.Background()

	// Seed defaults
	err := svc.SeedDefaults(ctx, map[string]string{
		"deployment_mode": "local_trusted",
		"allowed_origins": "localhost",
	})
	if err != nil {
		t.Fatalf("seeding defaults: %v", err)
	}

	// Get all
	all, err := svc.GetAll(ctx)
	if err != nil {
		t.Fatalf("getting all: %v", err)
	}

	// Verify defaults present
	if all["deployment_mode"] != "local_trusted" {
		t.Errorf("deployment_mode = %q, want 'local_trusted'", all["deployment_mode"])
	}
	if all["allowed_origins"] != "localhost" {
		t.Errorf("allowed_origins = %q, want 'localhost'", all["allowed_origins"])
	}
}

func TestPatch(t *testing.T) {
	s := testutil.NewStore(t)
	svc := settings.New(s)
	ctx := context.Background()

	// Seed defaults
	err := svc.SeedDefaults(ctx, map[string]string{
		"deployment_mode": "local_trusted",
		"allowed_origins": "localhost",
	})
	if err != nil {
		t.Fatalf("seeding defaults: %v", err)
	}

	// Patch one key
	updated, err := svc.Patch(ctx, map[string]string{
		"deployment_mode": "cloud",
	})
	if err != nil {
		t.Fatalf("patching: %v", err)
	}

	// Verify returned map
	if updated["deployment_mode"] != "cloud" {
		t.Errorf("returned deployment_mode = %q, want 'cloud'", updated["deployment_mode"])
	}
	if updated["allowed_origins"] != "localhost" {
		t.Errorf("returned allowed_origins = %q, want 'localhost'", updated["allowed_origins"])
	}
}

func TestGetAfterPatch(t *testing.T) {
	s := testutil.NewStore(t)
	svc := settings.New(s)
	ctx := context.Background()

	// Seed defaults
	err := svc.SeedDefaults(ctx, map[string]string{
		"deployment_mode": "local_trusted",
		"allowed_origins": "localhost",
	})
	if err != nil {
		t.Fatalf("seeding defaults: %v", err)
	}

	// Patch
	_, err = svc.Patch(ctx, map[string]string{
		"deployment_mode": "cloud",
	})
	if err != nil {
		t.Fatalf("patching: %v", err)
	}

	// GetAll again
	all, err := svc.GetAll(ctx)
	if err != nil {
		t.Fatalf("getting all after patch: %v", err)
	}

	// Verify persisted
	if all["deployment_mode"] != "cloud" {
		t.Errorf("persisted deployment_mode = %q, want 'cloud'", all["deployment_mode"])
	}
	if all["allowed_origins"] != "localhost" {
		t.Errorf("persisted allowed_origins = %q, want 'localhost'", all["allowed_origins"])
	}
}

func TestPatchEmptyBody(t *testing.T) {
	s := testutil.NewStore(t)
	svc := settings.New(s)
	ctx := context.Background()

	// Seed defaults
	err := svc.SeedDefaults(ctx, map[string]string{
		"deployment_mode": "local_trusted",
		"allowed_origins": "localhost",
	})
	if err != nil {
		t.Fatalf("seeding defaults: %v", err)
	}

	// Patch with empty map
	updated, err := svc.Patch(ctx, map[string]string{})
	if err != nil {
		t.Fatalf("patching empty: %v", err)
	}

	// Verify unchanged
	if updated["deployment_mode"] != "local_trusted" {
		t.Errorf("deployment_mode = %q, want 'local_trusted'", updated["deployment_mode"])
	}
	if updated["allowed_origins"] != "localhost" {
		t.Errorf("allowed_origins = %q, want 'localhost'", updated["allowed_origins"])
	}
}

func TestSeedDefaultsIdempotent(t *testing.T) {
	s := testutil.NewStore(t)
	svc := settings.New(s)
	ctx := context.Background()

	// Seed first time
	err := svc.SeedDefaults(ctx, map[string]string{
		"deployment_mode": "local_trusted",
		"allowed_origins": "localhost",
	})
	if err != nil {
		t.Fatalf("seeding first time: %v", err)
	}

	// Patch the setting
	_, err = svc.Patch(ctx, map[string]string{
		"deployment_mode": "cloud",
	})
	if err != nil {
		t.Fatalf("patching: %v", err)
	}

	// Seed again (should not overwrite)
	err = svc.SeedDefaults(ctx, map[string]string{
		"deployment_mode": "local_trusted",
		"allowed_origins": "localhost",
	})
	if err != nil {
		t.Fatalf("seeding second time: %v", err)
	}

	// Verify first value not overwritten
	all, err := svc.GetAll(ctx)
	if err != nil {
		t.Fatalf("getting all: %v", err)
	}

	if all["deployment_mode"] != "cloud" {
		t.Errorf("deployment_mode = %q, want 'cloud' (not overwritten)", all["deployment_mode"])
	}
	if all["allowed_origins"] != "localhost" {
		t.Errorf("allowed_origins = %q, want 'localhost'", all["allowed_origins"])
	}
}

func TestGetAllEmpty(t *testing.T) {
	s := testutil.NewStore(t)
	svc := settings.New(s)
	ctx := context.Background()

	// GetAll on fresh table
	all, err := svc.GetAll(ctx)
	if err != nil {
		t.Fatalf("getting all on fresh table: %v", err)
	}

	// Verify empty map (not nil)
	if all == nil {
		t.Errorf("returned nil, want empty map")
	}
	if len(all) != 0 {
		t.Errorf("returned map with %d entries, want 0", len(all))
	}
}
