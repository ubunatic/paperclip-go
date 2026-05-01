// Package heartbeat provides heartbeat execution and adapter infrastructure.
package heartbeat

import (
	"context"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/ubunatic/paperclip-go/internal/domain"
)

// Adapter defines the interface for heartbeat adapters.
// Implementations are responsible for executing a single heartbeat cycle
// for an agent and optionally selecting and executing work on a checked-out issue.
type Adapter interface {
	// Run executes a heartbeat cycle for the given agent and optional issue.
	// It returns a RunResult with status, summary, and optional error.
	Run(ctx context.Context, agent *domain.Agent, issue *domain.Issue) (*domain.RunResult, error)
}

// StubAdapter is a minimal adapter that always succeeds.
// It logs execution and returns success.
type StubAdapter struct{}

// Run implements the Adapter interface for StubAdapter.
// Always returns success with a generic summary.
func (a *StubAdapter) Run(ctx context.Context, agent *domain.Agent, issue *domain.Issue) (*domain.RunResult, error) {
	summary := "Heartbeat executed successfully"
	if issue != nil {
		summary = "Heartbeat executed with issue " + issue.ID
	}

	return &domain.RunResult{
		Status:  "success",
		Summary: summary,
		IssueID: nil, // stub doesn't return an issue
	}, nil
}

// Registry maps adapter names to their implementations.
type Registry struct {
	mu       sync.RWMutex
	adapters map[string]Adapter
}

// NewRegistry creates and returns a new Registry.
func NewRegistry() *Registry {
	return &Registry{
		adapters: make(map[string]Adapter),
	}
}

// Register associates an adapter name with an Adapter implementation.
func (r *Registry) Register(name string, adapter Adapter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.adapters[name] = adapter
}

// Get returns the Adapter for the given name, or nil if not found.
func (r *Registry) Get(name string) Adapter {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.adapters[name]
}

// NewDefaultRegistry creates a Registry with built-in adapters.
func NewDefaultRegistry() *Registry {
	r := NewRegistry()
	r.Register("stub", &StubAdapter{})

	// Register Claude adapter if API key is present
	if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		model := os.Getenv("ANTHROPIC_MODEL")
		if model == "" {
			model = "claude-haiku-4-5"
		}
		r.Register("claude_local", NewClaudeAdapter(key, model, &http.Client{Timeout: 60 * time.Second}))
	}

	return r
}
