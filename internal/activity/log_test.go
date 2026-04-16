package activity_test

import (
	"context"
	"testing"
	"time"

	"github.com/ubunatic/paperclip-go/internal/activity"
	"github.com/ubunatic/paperclip-go/internal/companies"
	"github.com/ubunatic/paperclip-go/internal/testutil"
)

func TestRecordActivity(t *testing.T) {
	s := testutil.NewStore(t)
	ctx := context.Background()

	// Create a company first
	companySvc := companies.New(s)
	company, err := companySvc.Create(ctx, "Test Corp", "test", "")
	if err != nil {
		t.Fatalf("Create company: %v", err)
	}

	// Record an activity
	log := activity.New(s)
	err = log.Record(ctx, company.ID, "agent", "agent-123", "created", "company", company.ID, `{"name":"Test Corp"}`)
	if err != nil {
		t.Fatalf("Record activity: %v", err)
	}
}

func TestListActivities(t *testing.T) {
	s := testutil.NewStore(t)
	ctx := context.Background()

	// Create a company
	companySvc := companies.New(s)
	company, err := companySvc.Create(ctx, "Test Corp", "test", "")
	if err != nil {
		t.Fatalf("Create company: %v", err)
	}

	// Record multiple activities with 1 second delay between each to ensure distinct timestamps
	log := activity.New(s)
	for i := 0; i < 5; i++ {
		err := log.Record(ctx, company.ID, "agent", "agent-123", "action", "entity", "entity-id", "{}")
		if err != nil {
			t.Fatalf("Record activity %d: %v", i, err)
		}
		if i < 4 {
			time.Sleep(1 * time.Second)
		}
	}

	// List with limit 3
	items, err := log.List(ctx, company.ID, 3)
	if err != nil {
		t.Fatalf("List activities: %v", err)
	}
	if len(items) != 3 {
		t.Errorf("List len = %d, want 3", len(items))
	}

	// Verify ordering is descending by created_at (most recent first)
	if items[0].CreatedAt.Before(items[1].CreatedAt) {
		t.Error("expected items to be ordered descending by created_at")
	}

	// List all (high limit)
	items2, err := log.List(ctx, company.ID, 100)
	if err != nil {
		t.Fatalf("List all activities: %v", err)
	}
	if len(items2) != 5 {
		t.Errorf("List all len = %d, want 5", len(items2))
	}
}
