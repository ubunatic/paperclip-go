package heartbeat

import (
	"context"

	"github.com/ubunatic/paperclip-go/internal/domain"
)

// MockAdapter is a test adapter that uses a customizable function to determine
// its behavior. It allows test code to inject deterministic responses without
// creating separate adapter types for each test scenario.
type MockAdapter struct {
	// summaryFn is called to generate the run result.
	summaryFn func(*domain.Agent, *domain.Issue) (*domain.RunResult, error)
}

// NewMockAdapter creates a new MockAdapter with the given function.
// The function is called for each Run() invocation and determines the result.
func NewMockAdapter(summaryFn func(*domain.Agent, *domain.Issue) (*domain.RunResult, error)) *MockAdapter {
	if summaryFn == nil {
		panic("NewMockAdapter: summaryFn cannot be nil")
	}
	return &MockAdapter{
		summaryFn: summaryFn,
	}
}

// Run implements the Adapter interface for MockAdapter.
// It delegates to the summaryFn function passed at construction time.
func (a *MockAdapter) Run(ctx context.Context, agent *domain.Agent, issue *domain.Issue) (*domain.RunResult, error) {
	return a.summaryFn(agent, issue)
}
