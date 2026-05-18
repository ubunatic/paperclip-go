package routines

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/ubunatic/paperclip-go/internal/domain"
	"github.com/ubunatic/paperclip-go/internal/heartbeat"
	"github.com/ubunatic/paperclip-go/internal/issues"
)

// Scheduler implements a background scheduler for running routines on a cron schedule.
type Scheduler struct {
	svc      *Service
	runner   *heartbeat.Runner
	issueSvc *issues.Service
	runSvc   *RunService
	tick     time.Duration
	now      func() time.Time
}

// NewScheduler creates a scheduler with default tick interval (60s) and system clock.
func NewScheduler(svc *Service, runner *heartbeat.Runner, issueSvc *issues.Service) *Scheduler {
	return NewSchedulerWithClock(svc, runner, issueSvc, 60*time.Second, func() time.Time { return time.Now().UTC() })
}

// NewSchedulerWithClock creates a scheduler with custom tick interval and clock function (for testing).
func NewSchedulerWithClock(svc *Service, runner *heartbeat.Runner, issueSvc *issues.Service, tick time.Duration, now func() time.Time) *Scheduler {
	return &Scheduler{svc: svc, runner: runner, issueSvc: issueSvc, tick: tick, now: now}
}

// WithRunService attaches a RunService so dispatches are recorded in routine_runs.
func (sch *Scheduler) WithRunService(rs *RunService) *Scheduler {
	sch.runSvc = rs
	return sch
}

// Start launches the background scheduler loop. Blocks until ctx is cancelled.
func (sch *Scheduler) Start(ctx context.Context) {
	ticker := time.NewTicker(sch.tick)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sch.tick_()
		}
	}
}

// tick_ is the internal tick handler that fires due routines.
func (sch *Scheduler) tick_() {
	ctx := context.Background()
	now := sch.now().Truncate(time.Minute)

	// Get all routines due at this slot
	due, err := sch.svc.DueRoutines(ctx, now)
	if err != nil {
		log.Printf("scheduler: DueRoutines error: %v", err)
		return
	}

	// For each due routine, attempt to dispatch
	for _, routine := range due {
		fingerprint := fmt.Sprintf("routine:%s:%s", routine.ID, now.Format(time.RFC3339))

		// Try to mark this slot as dispatched
		dispatched, err := sch.svc.MarkDispatched(ctx, routine.ID, fingerprint, now)
		if err != nil {
			log.Printf("scheduler: MarkDispatched(%s) error: %v", routine.ID, err)
			continue
		}
		if !dispatched {
			// Another process already claimed this slot; skip
			continue
		}

		// Record the dispatch before firing so the run row exists even if heartbeat fails.
		var routineRunID string
		if sch.runSvc != nil {
			rr, err := sch.runSvc.Record(ctx, routine.ID, routine.AgentID)
			if err != nil {
				log.Printf("scheduler: Record run(%s) error: %v", routine.ID, err)
			} else {
				routineRunID = rr.ID
			}
		}

		// Fire the heartbeat run asynchronously and update the run status when done.
		go func(r *domain.Routine, runID string) {
			runCtx := context.Background()
			_, err := sch.runner.Run(runCtx, r.AgentID)
			if sch.runSvc == nil || runID == "" {
				if err != nil {
					log.Printf("scheduler: Run(%s) error: %v", r.ID, err)
				}
				return
			}
			finCtx := context.Background()
			switch {
			case err == nil:
				if merr := sch.runSvc.MarkSucceeded(finCtx, runID); merr != nil {
					log.Printf("scheduler: MarkSucceeded(%s) error: %v", runID, merr)
				}
			case errors.Is(err, heartbeat.ErrAlreadyRunning),
				errors.Is(err, heartbeat.ErrApprovalPending),
				errors.Is(err, heartbeat.ErrBudgetExceeded):
				if merr := sch.runSvc.MarkSkipped(finCtx, runID); merr != nil {
					log.Printf("scheduler: MarkSkipped(%s) error: %v", runID, merr)
				}
			default:
				if merr := sch.runSvc.MarkFailed(finCtx, runID, err.Error()); merr != nil {
					log.Printf("scheduler: MarkFailed(%s) error: %v", runID, merr)
				}
				log.Printf("scheduler: Run(%s) error: %v", r.ID, err)
			}
		}(routine, routineRunID)
	}
}
