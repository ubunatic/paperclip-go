package routines_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ubunatic/paperclip-go/internal/ids"
	"github.com/ubunatic/paperclip-go/internal/routines"
	"github.com/ubunatic/paperclip-go/internal/store"
	"github.com/ubunatic/paperclip-go/internal/testutil"
)

func setupTestData(t *testing.T, s *store.Store) (companyID, agentID string) {
	t.Helper()
	ctx := context.Background()

	// Create company
	companyID = "company-test-" + ids.NewUUID()
	_, err := s.DB.ExecContext(ctx,
		`INSERT INTO companies(id, name, shortname, description, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		companyID, "Test Company", "test", "Test", "2024-01-01T00:00:00Z", "2024-01-01T00:00:00Z",
	)
	if err != nil {
		t.Fatalf("create company: %v", err)
	}

	// Create agent
	agentID = "agent-test-" + ids.NewUUID()
	_, err = s.DB.ExecContext(ctx,
		`INSERT INTO agents(id, company_id, shortname, display_name, role, adapter, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		agentID, companyID, "test-agent", "Test Agent", "test", "stub", "2024-01-01T00:00:00Z", "2024-01-01T00:00:00Z",
	)
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}

	return companyID, agentID
}

func TestCreate(t *testing.T) {
	s := testutil.NewStore(t)
	svc := routines.New(s)
	ctx := context.Background()

	companyID, agentID := setupTestData(t, s)

	// Create routine
	routine, err := svc.Create(ctx, companyID, agentID, "Morning Sync", "0 9 * * *")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if routine == nil {
		t.Fatal("expected routine, got nil")
	}
	if routine.ID == "" {
		t.Error("expected ID to be set")
	}
	if routine.CompanyID != companyID {
		t.Errorf("expected companyId %s, got %s", companyID, routine.CompanyID)
	}
	if routine.AgentID != agentID {
		t.Errorf("expected agentId %s, got %s", agentID, routine.AgentID)
	}
	if routine.Name != "Morning Sync" {
		t.Errorf("expected name Morning Sync, got %s", routine.Name)
	}
	if routine.CronExpr != "0 9 * * *" {
		t.Errorf("expected cronExpr 0 9 * * *, got %s", routine.CronExpr)
	}
	if !routine.Enabled {
		t.Error("expected enabled to be true")
	}
	if routine.LastRunAt != nil {
		t.Error("expected lastRunAt to be nil")
	}
	if routine.CreatedAt.IsZero() {
		t.Error("expected createdAt to be set")
	}
	if routine.UpdatedAt.IsZero() {
		t.Error("expected updatedAt to be set")
	}
}

func TestCreateInvalidCron(t *testing.T) {
	s := testutil.NewStore(t)
	svc := routines.New(s)
	ctx := context.Background()

	companyID, agentID := setupTestData(t, s)

	// Try to create routine with invalid cron
	_, err := svc.Create(ctx, companyID, agentID, "Bad Cron", "invalid cron")
	if !errors.Is(err, routines.ErrInvalidCron) {
		t.Errorf("expected ErrInvalidCron, got %v", err)
	}
}

func TestCreateDuplicateName(t *testing.T) {
	s := testutil.NewStore(t)
	svc := routines.New(s)
	ctx := context.Background()

	companyID, agentID := setupTestData(t, s)

	// Create first routine
	_, err := svc.Create(ctx, companyID, agentID, "Daily Task", "0 9 * * *")
	if err != nil {
		t.Fatalf("Create 1: %v", err)
	}

	// Try to create second routine with same name
	_, err = svc.Create(ctx, companyID, agentID, "Daily Task", "0 10 * * *")
	if !errors.Is(err, routines.ErrNameConflict) {
		t.Errorf("expected ErrNameConflict, got %v", err)
	}
}

func TestGetByID(t *testing.T) {
	s := testutil.NewStore(t)
	svc := routines.New(s)
	ctx := context.Background()

	companyID, agentID := setupTestData(t, s)

	// Create routine
	created, err := svc.Create(ctx, companyID, agentID, "Test Routine", "0 9 * * *")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Get routine
	retrieved, err := svc.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}

	if retrieved.ID != created.ID {
		t.Errorf("expected ID %s, got %s", created.ID, retrieved.ID)
	}
	if retrieved.Name != "Test Routine" {
		t.Errorf("expected name Test Routine, got %s", retrieved.Name)
	}
}

func TestGetByIDNotFound(t *testing.T) {
	s := testutil.NewStore(t)
	svc := routines.New(s)
	ctx := context.Background()

	_, err := svc.GetByID(ctx, "nonexistent-id")
	if !errors.Is(err, routines.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestListByCompany(t *testing.T) {
	s := testutil.NewStore(t)
	svc := routines.New(s)
	ctx := context.Background()

	companyID, agentID := setupTestData(t, s)

	// Create two routines
	r1, err := svc.Create(ctx, companyID, agentID, "Task 1", "0 9 * * *")
	if err != nil {
		t.Fatalf("Create 1: %v", err)
	}

	r2, err := svc.Create(ctx, companyID, agentID, "Task 2", "0 10 * * *")
	if err != nil {
		t.Fatalf("Create 2: %v", err)
	}

	// List routines
	list, err := svc.ListByCompany(ctx, companyID)
	if err != nil {
		t.Fatalf("ListByCompany: %v", err)
	}

	if len(list) != 2 {
		t.Errorf("expected 2 routines, got %d", len(list))
	}

	// Check both routines are present
	ids := map[string]bool{
		list[0].ID: true,
		list[1].ID: true,
	}
	if !ids[r1.ID] || !ids[r2.ID] {
		t.Errorf("expected routines %s and %s, got %s and %s", r1.ID, r2.ID, list[0].ID, list[1].ID)
	}
}

func TestListByCompanyEmpty(t *testing.T) {
	s := testutil.NewStore(t)
	svc := routines.New(s)
	ctx := context.Background()

	list, err := svc.ListByCompany(ctx, "nonexistent-company")
	if err != nil {
		t.Fatalf("ListByCompany: %v", err)
	}

	if len(list) != 0 {
		t.Errorf("expected empty list, got %d routines", len(list))
	}
}

func TestUpdate(t *testing.T) {
	s := testutil.NewStore(t)
	svc := routines.New(s)
	ctx := context.Background()

	companyID, agentID := setupTestData(t, s)

	// Create routine
	created, err := svc.Create(ctx, companyID, agentID, "Original Name", "0 9 * * *")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Update name and cron
	newName := "Updated Name"
	newCron := "0 10 * * *"
	newEnabled := false

	updated, err := svc.Update(ctx, created.ID, routines.UpdateInput{
		Name:     &newName,
		CronExpr: &newCron,
		Enabled:  &newEnabled,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	if updated.Name != newName {
		t.Errorf("expected name %s, got %s", newName, updated.Name)
	}
	if updated.CronExpr != newCron {
		t.Errorf("expected cronExpr %s, got %s", newCron, updated.CronExpr)
	}
	if updated.Enabled != newEnabled {
		t.Errorf("expected enabled %v, got %v", newEnabled, updated.Enabled)
	}
}

func TestUpdatePartial(t *testing.T) {
	s := testutil.NewStore(t)
	svc := routines.New(s)
	ctx := context.Background()

	companyID, agentID := setupTestData(t, s)

	// Create routine
	created, err := svc.Create(ctx, companyID, agentID, "Original Name", "0 9 * * *")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Update only name
	newName := "New Name"
	updated, err := svc.Update(ctx, created.ID, routines.UpdateInput{
		Name: &newName,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	if updated.Name != newName {
		t.Errorf("expected name %s, got %s", newName, updated.Name)
	}
	if updated.CronExpr != "0 9 * * *" {
		t.Errorf("expected cronExpr unchanged, got %s", updated.CronExpr)
	}
	if !updated.Enabled {
		t.Errorf("expected enabled unchanged, got %v", updated.Enabled)
	}
}

func TestUpdateInvalidCron(t *testing.T) {
	s := testutil.NewStore(t)
	svc := routines.New(s)
	ctx := context.Background()

	companyID, agentID := setupTestData(t, s)

	// Create routine
	created, err := svc.Create(ctx, companyID, agentID, "Test", "0 9 * * *")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Try to update with invalid cron
	badCron := "invalid"
	_, err = svc.Update(ctx, created.ID, routines.UpdateInput{
		CronExpr: &badCron,
	})
	if !errors.Is(err, routines.ErrInvalidCron) {
		t.Errorf("expected ErrInvalidCron, got %v", err)
	}
}

func TestDelete(t *testing.T) {
	s := testutil.NewStore(t)
	svc := routines.New(s)
	ctx := context.Background()

	companyID, agentID := setupTestData(t, s)

	// Create routine
	created, err := svc.Create(ctx, companyID, agentID, "To Delete", "0 9 * * *")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Delete routine
	err = svc.Delete(ctx, created.ID)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Verify it's gone
	_, err = svc.GetByID(ctx, created.ID)
	if !errors.Is(err, routines.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestDeleteNotFound(t *testing.T) {
	s := testutil.NewStore(t)
	svc := routines.New(s)
	ctx := context.Background()

	err := svc.Delete(ctx, "nonexistent-id")
	if !errors.Is(err, routines.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestTrigger(t *testing.T) {
	s := testutil.NewStore(t)
	mockTime := time.Date(2024, 5, 4, 12, 30, 0, 0, time.UTC)
	svc := routines.NewWithClock(s, func() time.Time { return mockTime })
	ctx := context.Background()

	companyID, agentID := setupTestData(t, s)

	// Create routine
	created, err := svc.Create(ctx, companyID, agentID, "To Trigger", "0 9 * * *")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Initially, last_run_at should be nil
	if created.LastRunAt != nil {
		t.Error("expected lastRunAt to be nil initially")
	}

	// Trigger routine
	triggered, err := svc.Trigger(ctx, created.ID)
	if err != nil {
		t.Fatalf("Trigger: %v", err)
	}

	if triggered.LastRunAt == nil {
		t.Error("expected lastRunAt to be set after trigger")
	} else if !triggered.LastRunAt.Equal(mockTime) {
		t.Errorf("expected lastRunAt to be %v, got %v", mockTime, *triggered.LastRunAt)
	}
}

func TestDueRoutines_MockClock(t *testing.T) {
	s := testutil.NewStore(t)
	ctx := context.Background()

	companyID, agentID := setupTestData(t, s)

	// Test clock at 08:59
	time0859 := time.Date(2024, 5, 4, 8, 59, 0, 0, time.UTC)
	svc0859 := routines.NewWithClock(s, func() time.Time { return time0859 })

	// Create routine that runs at 09:00
	routine, err := svc0859.Create(ctx, companyID, agentID, "Morning Task", "0 9 * * *")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// At 08:59, routine should not be due
	due, err := svc0859.DueRoutines(ctx, time0859)
	if err != nil {
		t.Fatalf("DueRoutines: %v", err)
	}
	if len(due) != 0 {
		t.Errorf("expected 0 due routines at 08:59, got %d", len(due))
	}

	// Now test at 09:00
	time0900 := time.Date(2024, 5, 4, 9, 0, 0, 0, time.UTC)
	svc0900 := routines.NewWithClock(s, func() time.Time { return time0900 })

	// Should be due now
	due, err = svc0900.DueRoutines(ctx, time0900)
	if err != nil {
		t.Fatalf("DueRoutines: %v", err)
	}
	if len(due) != 1 {
		t.Errorf("expected 1 due routine at 09:00, got %d", len(due))
	}
	if due[0].ID != routine.ID {
		t.Errorf("expected routine %s, got %s", routine.ID, due[0].ID)
	}

	// At 09:01, with last_run_at still null, should not be due
	time0901 := time.Date(2024, 5, 4, 9, 1, 0, 0, time.UTC)
	svc0901 := routines.NewWithClock(s, func() time.Time { return time0901 })

	due, err = svc0901.DueRoutines(ctx, time0901)
	if err != nil {
		t.Fatalf("DueRoutines: %v", err)
	}
	if len(due) != 0 {
		t.Errorf("expected 0 due routines at 09:01, got %d", len(due))
	}
}

func TestMarkDispatched_Idempotent(t *testing.T) {
	s := testutil.NewStore(t)
	svc := routines.New(s)
	ctx := context.Background()

	companyID, agentID := setupTestData(t, s)

	// Create routine
	routine, err := svc.Create(ctx, companyID, agentID, "To Dispatch", "0 9 * * *")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// First dispatch should succeed
	now := time.Date(2024, 5, 4, 9, 0, 0, 0, time.UTC)
	marked, err := svc.MarkDispatched(ctx, routine.ID, "fingerprint1", now)
	if err != nil {
		t.Fatalf("MarkDispatched (1st): %v", err)
	}
	if !marked {
		t.Error("expected first dispatch to succeed")
	}

	// Verify dispatch_fingerprint was set
	retrieved, err := svc.GetByID(ctx, routine.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if retrieved.DispatchFingerprint == nil || *retrieved.DispatchFingerprint != "fingerprint1" {
		t.Errorf("expected dispatch_fingerprint to be set to fingerprint1, got %v", retrieved.DispatchFingerprint)
	}

	// Second dispatch with same fingerprint should fail
	marked, err = svc.MarkDispatched(ctx, routine.ID, "fingerprint1", now)
	if err != nil {
		t.Fatalf("MarkDispatched (2nd): %v", err)
	}
	if marked {
		t.Error("expected second dispatch to fail (already dispatched)")
	}
}

func TestDisabledRoutineNotDue(t *testing.T) {
	s := testutil.NewStore(t)
	ctx := context.Background()

	companyID, agentID := setupTestData(t, s)

	// Create and disable routine
	mockTime := time.Date(2024, 5, 4, 9, 0, 0, 0, time.UTC)
	svc := routines.NewWithClock(s, func() time.Time { return mockTime })

	routine, err := svc.Create(ctx, companyID, agentID, "Disabled Task", "0 9 * * *")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Disable routine
	disabled := false
	_, err = svc.Update(ctx, routine.ID, routines.UpdateInput{Enabled: &disabled})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	// Should not appear in due routines
	due, err := svc.DueRoutines(ctx, mockTime)
	if err != nil {
		t.Fatalf("DueRoutines: %v", err)
	}
	if len(due) != 0 {
		t.Errorf("expected disabled routine to not be due, got %d", len(due))
	}
}
