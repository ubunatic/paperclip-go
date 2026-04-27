package heartbeat_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ubunatic/paperclip-go/internal/domain"
	"github.com/ubunatic/paperclip-go/internal/heartbeat"
)

func TestMockAdapterSuccess(t *testing.T) {
	// Create a mock adapter that returns a successful result
	adapter := heartbeat.NewMockAdapter(func(agent *domain.Agent, issue *domain.Issue) (*domain.RunResult, error) {
		return &domain.RunResult{
			Status:  "success",
			Summary: "Test success",
			IssueID: nil,
		}, nil
	})

	agent := &domain.Agent{ID: "test"}
	result, err := adapter.Run(context.Background(), agent, nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "success" {
		t.Errorf("Status = %q, want %q", result.Status, "success")
	}
	if result.Summary != "Test success" {
		t.Errorf("Summary = %q, want %q", result.Summary, "Test success")
	}
	if result.IssueID != nil {
		t.Errorf("IssueID = %v, want nil", result.IssueID)
	}
}

func TestMockAdapterError(t *testing.T) {
	// Create a mock adapter that returns an error
	testErr := errors.New("test error")
	adapter := heartbeat.NewMockAdapter(func(agent *domain.Agent, issue *domain.Issue) (*domain.RunResult, error) {
		return nil, testErr
	})

	agent := &domain.Agent{ID: "test"}
	result, err := adapter.Run(context.Background(), agent, nil)

	if !errors.Is(err, testErr) {
		t.Errorf("error = %v, want %v", err, testErr)
	}
	if result != nil {
		t.Errorf("result = %v, want nil", result)
	}
}

func TestMockAdapterWithNilIssue(t *testing.T) {
	// Create a mock adapter that checks for nil issue
	adapter := heartbeat.NewMockAdapter(func(agent *domain.Agent, issue *domain.Issue) (*domain.RunResult, error) {
		if issue != nil {
			return nil, errors.New("expected nil issue")
		}
		return &domain.RunResult{
			Status:  "success",
			Summary: "No issue provided",
			IssueID: nil,
		}, nil
	})

	agent := &domain.Agent{ID: "test"}
	result, err := adapter.Run(context.Background(), agent, nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "success" {
		t.Errorf("Status = %q, want %q", result.Status, "success")
	}
}

func TestMockAdapterNilFunction(t *testing.T) {
	// Verify that NewMockAdapter panics with a clear message when summaryFn is nil
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic, but NewMockAdapter did not panic")
		} else if msg, ok := r.(string); !ok || msg != "NewMockAdapter: summaryFn cannot be nil" {
			t.Errorf("panic message = %v, want %q", r, "NewMockAdapter: summaryFn cannot be nil")
		}
	}()
	heartbeat.NewMockAdapter(nil)
}
