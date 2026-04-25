package activity_test

import (
	"context"
	"testing"

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
	a, err := log.Record(ctx, company.ID, "agent", "agent-123", "created", "company", company.ID, `{"name":"Test Corp"}`)
	if err != nil {
		t.Fatalf("Record activity: %v", err)
	}

	// Verify returned activity
	if a == nil {
		t.Fatalf("Record returned nil activity")
	}
	if a.ID == "" {
		t.Fatalf("Activity ID is empty")
	}
	if a.CompanyID != company.ID {
		t.Errorf("CompanyID = %s, want %s", a.CompanyID, company.ID)
	}
	if a.Action != "created" {
		t.Errorf("Action = %s, want created", a.Action)
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

	// Record multiple activities - ordering is now deterministic with ORDER BY id DESC as secondary sort
	log := activity.New(s)
	for i := 0; i < 5; i++ {
		_, err := log.Record(ctx, company.ID, "agent", "agent-123", "action", "entity", "entity-id", "{}")
		if err != nil {
			t.Fatalf("Record activity %d: %v", i, err)
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

	// Verify ordering is descending by id (most recent first)
	// Items are ordered by created_at DESC, id DESC, so check id ordering
	for i := 0; i < len(items)-1; i++ {
		if items[i].ID <= items[i+1].ID {
			t.Errorf("want DESC by id, got items[%d].ID=%s > items[%d].ID=%s", i, items[i].ID, i+1, items[i+1].ID)
		}
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

func TestListByEntity(t *testing.T) {
	s := testutil.NewStore(t)
	ctx := context.Background()

	// Create a company
	companySvc := companies.New(s)
	company, err := companySvc.Create(ctx, "Test Corp", "test", "")
	if err != nil {
		t.Fatalf("Create company: %v", err)
	}

	// Record activities for two different entities
	log := activity.New(s)
	const entity1Kind = "issue"
	const entity1ID = "issue-123"
	const entity2Kind = "issue"
	const entity2ID = "issue-456"

	// Record 3 activities for entity1
	for i := 0; i < 3; i++ {
		_, err := log.Record(ctx, company.ID, "agent", "agent-123", "created", entity1Kind, entity1ID, "{}")
		if err != nil {
			t.Fatalf("Record activity for entity1 %d: %v", i, err)
		}
	}

	// Record 2 activities for entity2
	for i := 0; i < 2; i++ {
		_, err := log.Record(ctx, company.ID, "agent", "agent-456", "updated", entity2Kind, entity2ID, "{}")
		if err != nil {
			t.Fatalf("Record activity for entity2 %d: %v", i, err)
		}
	}

	// List activities for entity1 (should be in chronological order, ascending)
	items1, err := log.ListByEntity(ctx, entity1Kind, entity1ID)
	if err != nil {
		t.Fatalf("ListByEntity for entity1: %v", err)
	}
	if len(items1) != 3 {
		t.Errorf("ListByEntity entity1 len = %d, want 3", len(items1))
	}

	// Verify chronological ordering (ascending by id since all have same created_at)
	// Items are ordered by created_at ASC, id ASC
	for i := 0; i < len(items1)-1; i++ {
		if items1[i].ID >= items1[i+1].ID {
			t.Errorf("want ASC by id, got items[%d].ID=%s >= items[%d].ID=%s", i, items1[i].ID, i+1, items1[i+1].ID)
		}
	}

	// List activities for entity2
	items2, err := log.ListByEntity(ctx, entity2Kind, entity2ID)
	if err != nil {
		t.Fatalf("ListByEntity for entity2: %v", err)
	}
	if len(items2) != 2 {
		t.Errorf("ListByEntity entity2 len = %d, want 2", len(items2))
	}

	// List activities for nonexistent entity
	items3, err := log.ListByEntity(ctx, "nonexistent", "nonexistent-id")
	if err != nil {
		t.Fatalf("ListByEntity for nonexistent: %v", err)
	}
	if len(items3) != 0 {
		t.Errorf("ListByEntity nonexistent len = %d, want 0", len(items3))
	}
}
