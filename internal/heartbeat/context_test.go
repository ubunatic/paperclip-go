package heartbeat_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ubunatic/paperclip-go/internal/activity"
	"github.com/ubunatic/paperclip-go/internal/agents"
	"github.com/ubunatic/paperclip-go/internal/comments"
	"github.com/ubunatic/paperclip-go/internal/companies"
	"github.com/ubunatic/paperclip-go/internal/domain"
	"github.com/ubunatic/paperclip-go/internal/heartbeat"
	"github.com/ubunatic/paperclip-go/internal/issues"
	"github.com/ubunatic/paperclip-go/internal/testutil"
)

// blockingAdapter blocks until ctx is cancelled.
type blockingAdapter struct{}

func (a *blockingAdapter) Run(ctx context.Context, agent *domain.Agent, issue *domain.Issue) (*domain.RunResult, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

func newRunnerWithBlockingAdapter(t *testing.T, adapterName string) (*heartbeat.Runner, string) {
	t.Helper()
	s := testutil.NewStore(t)
	ctx := context.Background()

	companySvc := companies.New(s)
	company, err := companySvc.Create(ctx, "Test Corp", "ctx-test-"+adapterName, "Context test company")
	if err != nil {
		t.Fatalf("Create company: %v", err)
	}

	agentSvc := agents.New(s, activity.New(s))
	agent, err := agentSvc.Create(ctx, company.ID, "blocker-"+adapterName, "Blocking Agent", "agent", nil, adapterName)
	if err != nil {
		t.Fatalf("Create agent: %v", err)
	}

	registry := heartbeat.NewRegistry()
	registry.Register(adapterName, &blockingAdapter{})

	issueSvc := issues.New(s)
	commentSvc := comments.New(s)
	actLog := activity.New(s)
	runner := heartbeat.New(s, agentSvc, issueSvc, commentSvc, actLog, registry, nil)

	return runner, agent.ID
}

func TestRunCancelledByContext(t *testing.T) {
	runner, agentID := newRunnerWithBlockingAdapter(t, "block-cancel")

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		_, err := runner.Run(ctx, agentID)
		done <- err
	}()

	// Cancel context after a short delay to let Run() start
	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected error after context cancellation, got nil")
		}
		if !errors.Is(err, context.Canceled) {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Run() did not return after context cancellation")
	}
}

func TestRunTimeout(t *testing.T) {
	runner, agentID := newRunnerWithBlockingAdapter(t, "block-timeout")

	// Set a very short timeout
	runner.Timeout = 50 * time.Millisecond

	_, err := runner.Run(context.Background(), agentID)
	if err == nil {
		t.Fatal("expected error from timeout, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected context.DeadlineExceeded, got %v", err)
	}
}
