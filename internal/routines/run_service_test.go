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
