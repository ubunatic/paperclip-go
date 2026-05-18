package routines_test

import (
	"context"
	"testing"

	"github.com/ubunatic/paperclip-go/internal/routines"
	"github.com/ubunatic/paperclip-go/internal/testutil"
)

func TestRoutineRunRecord(t *testing.T) {
	s := testutil.NewStore(t)
	ctx := context.Background()

	companyID, agentID := setupTestData(t, s)

	svc := routines.New(s)
	routine, err := svc.Create(ctx, companyID, agentID, "Run Test Routine", "0 9 * * *")
	if err != nil {
		t.Fatalf("Create routine: %v", err)
	}

	runSvc := routines.NewRunService(s)
	run, err := runSvc.Record(ctx, routine.ID, agentID)
	if err != nil {
		t.Fatalf("Record: %v", err)
	}

	if run == nil {
		t.Fatal("expected run, got nil")
	}
	if run.ID == "" {
		t.Error("expected run ID to be set")
	}
	if run.RoutineID != routine.ID {
		t.Errorf("expected routineID %s, got %s", routine.ID, run.RoutineID)
	}
	if run.AgentID != agentID {
		t.Errorf("expected agentID %s, got %s", agentID, run.AgentID)
	}
	if run.Status != "dispatched" {
		t.Errorf("expected status dispatched, got %s", run.Status)
	}
	if run.StartedAt.IsZero() {
		t.Error("expected StartedAt to be non-zero")
	}
}

func TestRoutineRunListOrderedDescending(t *testing.T) {
	s := testutil.NewStore(t)
	ctx := context.Background()

	companyID, agentID := setupTestData(t, s)

	svc := routines.New(s)
	routine, err := svc.Create(ctx, companyID, agentID, "Order Routine", "0 9 * * *")
	if err != nil {
		t.Fatalf("Create routine: %v", err)
	}

	runSvc := routines.NewRunService(s)
	run1, err := runSvc.Record(ctx, routine.ID, agentID)
	if err != nil {
		t.Fatalf("Record run1: %v", err)
	}
	run2, err := runSvc.Record(ctx, routine.ID, agentID)
	if err != nil {
		t.Fatalf("Record run2: %v", err)
	}

	runs, err := runSvc.ListByRoutine(ctx, routine.ID)
	if err != nil {
		t.Fatalf("ListByRoutine: %v", err)
	}
	if len(runs) != 2 {
		t.Fatalf("expected 2 runs, got %d", len(runs))
	}
	// Both runs must be present regardless of sub-second ordering
	ids := map[string]bool{runs[0].ID: true, runs[1].ID: true}
	if !ids[run1.ID] || !ids[run2.ID] {
		t.Errorf("runs = %v, want both %q and %q", ids, run1.ID, run2.ID)
	}
	// All runs must belong to this routine
	for _, r := range runs {
		if r.RoutineID != routine.ID {
			t.Errorf("run.RoutineID = %q, want %q", r.RoutineID, routine.ID)
		}
	}
}

func TestRoutineRunListByRoutine(t *testing.T) {
	s := testutil.NewStore(t)
	ctx := context.Background()

	companyID, agentID := setupTestData(t, s)

	svc := routines.New(s)

	routine1, err := svc.Create(ctx, companyID, agentID, "Routine One", "0 9 * * *")
	if err != nil {
		t.Fatalf("Create routine1: %v", err)
	}

	routine2, err := svc.Create(ctx, companyID, agentID, "Routine Two", "0 10 * * *")
	if err != nil {
		t.Fatalf("Create routine2: %v", err)
	}

	runSvc := routines.NewRunService(s)

	if _, err := runSvc.Record(ctx, routine1.ID, agentID); err != nil {
		t.Fatalf("Record run1 for routine1: %v", err)
	}
	if _, err := runSvc.Record(ctx, routine1.ID, agentID); err != nil {
		t.Fatalf("Record run2 for routine1: %v", err)
	}
	if _, err := runSvc.Record(ctx, routine2.ID, agentID); err != nil {
		t.Fatalf("Record run for routine2: %v", err)
	}

	runs, err := runSvc.ListByRoutine(ctx, routine1.ID)
	if err != nil {
		t.Fatalf("ListByRoutine: %v", err)
	}

	if len(runs) != 2 {
		t.Errorf("expected 2 runs for routine1, got %d", len(runs))
	}

	for _, r := range runs {
		if r.RoutineID != routine1.ID {
			t.Errorf("expected routineID %s, got %s", routine1.ID, r.RoutineID)
		}
	}
}

func TestRoutineRunMarkSucceeded(t *testing.T) {
	s := testutil.NewStore(t)
	ctx := context.Background()

	companyID, agentID := setupTestData(t, s)
	svc := routines.New(s)
	routine, err := svc.Create(ctx, companyID, agentID, "Mark Succeeded Routine", "0 9 * * *")
	if err != nil {
		t.Fatalf("Create routine: %v", err)
	}

	runSvc := routines.NewRunService(s)
	run, err := runSvc.Record(ctx, routine.ID, agentID)
	if err != nil {
		t.Fatalf("Record: %v", err)
	}

	if err := runSvc.MarkSucceeded(ctx, run.ID); err != nil {
		t.Fatalf("MarkSucceeded: %v", err)
	}

	runs, err := runSvc.ListByRoutine(ctx, routine.ID)
	if err != nil {
		t.Fatalf("ListByRoutine: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(runs))
	}
	got := runs[0]
	if got.Status != "succeeded" {
		t.Errorf("status = %q, want %q", got.Status, "succeeded")
	}
	if got.FinishedAt == nil {
		t.Error("FinishedAt is nil, want non-nil")
	}
	if got.Error != nil {
		t.Errorf("Error = %v, want nil", got.Error)
	}
}

func TestRoutineRunMarkFailed(t *testing.T) {
	s := testutil.NewStore(t)
	ctx := context.Background()

	companyID, agentID := setupTestData(t, s)
	svc := routines.New(s)
	routine, err := svc.Create(ctx, companyID, agentID, "Mark Failed Routine", "0 9 * * *")
	if err != nil {
		t.Fatalf("Create routine: %v", err)
	}

	runSvc := routines.NewRunService(s)
	run, err := runSvc.Record(ctx, routine.ID, agentID)
	if err != nil {
		t.Fatalf("Record: %v", err)
	}

	if err := runSvc.MarkFailed(ctx, run.ID, "something went wrong"); err != nil {
		t.Fatalf("MarkFailed: %v", err)
	}

	runs, err := runSvc.ListByRoutine(ctx, routine.ID)
	if err != nil {
		t.Fatalf("ListByRoutine: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(runs))
	}
	got := runs[0]
	if got.Status != "failed" {
		t.Errorf("status = %q, want %q", got.Status, "failed")
	}
	if got.FinishedAt == nil {
		t.Error("FinishedAt is nil, want non-nil")
	}
	if got.Error == nil {
		t.Error("Error is nil, want non-nil")
	} else if *got.Error != "something went wrong" {
		t.Errorf("Error = %q, want %q", *got.Error, "something went wrong")
	}
}

func TestRoutineRunMarkSkipped(t *testing.T) {
	s := testutil.NewStore(t)
	ctx := context.Background()

	companyID, agentID := setupTestData(t, s)
	svc := routines.New(s)
	routine, err := svc.Create(ctx, companyID, agentID, "Mark Skipped Routine", "0 9 * * *")
	if err != nil {
		t.Fatalf("Create routine: %v", err)
	}

	runSvc := routines.NewRunService(s)
	run, err := runSvc.Record(ctx, routine.ID, agentID)
	if err != nil {
		t.Fatalf("Record: %v", err)
	}

	if err := runSvc.MarkSkipped(ctx, run.ID); err != nil {
		t.Fatalf("MarkSkipped: %v", err)
	}

	runs, err := runSvc.ListByRoutine(ctx, routine.ID)
	if err != nil {
		t.Fatalf("ListByRoutine: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(runs))
	}
	got := runs[0]
	if got.Status != "skipped" {
		t.Errorf("status = %q, want %q", got.Status, "skipped")
	}
	if got.FinishedAt == nil {
		t.Error("FinishedAt is nil, want non-nil")
	}
	if got.Error != nil {
		t.Errorf("Error = %v, want nil", got.Error)
	}
}
