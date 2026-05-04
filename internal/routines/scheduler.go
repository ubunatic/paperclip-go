package routines

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ubunatic/paperclip-go/internal/domain"
	"github.com/ubunatic/paperclip-go/internal/heartbeat"
	"github.com/ubunatic/paperclip-go/internal/issues"
)

// Scheduler implements a background scheduler for running routines on a cron schedule.
type Scheduler struct {
	svc     *Service
	runner  *heartbeat.Runner
	issueSvc *issues.Service
	tick    time.Duration
	now     func() time.Time
}

// NewScheduler creates a scheduler with default tick interval (60s) and system clock.
func NewScheduler(svc *Service, runner *heartbeat.Runner, issueSvc *issues.Service) *Scheduler {
	return NewSchedulerWithClock(svc, runner, issueSvc, 60*time.Second, func() time.Time { return time.Now().UTC() })
}

// NewSchedulerWithClock creates a scheduler with custom tick interval and clock function (for testing).
func NewSchedulerWithClock(svc *Service, runner *heartbeat.Runner, issueSvc *issues.Service, tick time.Duration, now func() time.Time) *Scheduler {
	return &Scheduler{svc: svc, runner: runner, issueSvc: issueSvc, tick: tick, now: now}
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

		// Fire the heartbeat run asynchronously
		go func(r *domain.Routine) {
			runCtx := context.Background()
			_, err := sch.runner.Run(runCtx, r.AgentID)
			if err != nil {
				log.Printf("scheduler: Run(%s) error: %v", r.ID, err)
				// Leave dispatch fingerprint set to avoid duplicate runs
				// Manual retry via POST /trigger if needed
			}
		}(routine)
	}
}
