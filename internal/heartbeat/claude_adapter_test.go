package heartbeat_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/ubunatic/paperclip-go/internal/domain"
	"github.com/ubunatic/paperclip-go/internal/heartbeat"
)

// mockLLMClient implements heartbeat.LLMClient for testing.
type mockLLMClient struct {
	body   string
	status int
	err    error
}

// Do implements heartbeat.LLMClient.
func (m *mockLLMClient) Do(req *http.Request) (*http.Response, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &http.Response{
		StatusCode: m.status,
		Body:       io.NopCloser(bytes.NewBufferString(m.body)),
		Header:     make(http.Header),
	}, nil
}

func TestClaudeAdapterSuccess(t *testing.T) {
	// Mock a successful Anthropic response
	mockClient := &mockLLMClient{
		status: 200,
		body: `{
			"content": [{"type": "text", "text": "This is the model response"}],
			"stop_reason": "end_turn"
		}`,
	}

	adapter := heartbeat.NewClaudeAdapter("test-key", "claude-haiku-4-5", mockClient)

	agent := &domain.Agent{ID: "test-agent"}
	issue := &domain.Issue{
		ID:    "test-issue",
		Title: "Test Title",
		Body:  "Test Body",
	}

	result, err := adapter.Run(context.Background(), agent, issue)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "success" {
		t.Errorf("Status = %q, want %q", result.Status, "success")
	}
	if result.Summary != "This is the model response" {
		t.Errorf("Summary = %q, want %q", result.Summary, "This is the model response")
	}
}

func TestClaudeAdapterAPIError(t *testing.T) {
	// Mock a 400 error response
	mockClient := &mockLLMClient{
		status: 400,
		body: `{
			"error": {
				"message": "Invalid API key"
			}
		}`,
	}

	adapter := heartbeat.NewClaudeAdapter("test-key", "claude-haiku-4-5", mockClient)

	agent := &domain.Agent{ID: "test-agent"}
	result, err := adapter.Run(context.Background(), agent, nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "error" {
		t.Errorf("Status = %q, want %q", result.Status, "error")
	}
	if result.Summary != "Invalid API key" {
		t.Errorf("Summary = %q, want %q", result.Summary, "Invalid API key")
	}
}

func TestClaudeAdapterEmptyResponse(t *testing.T) {
	// Mock a successful response with empty content array
	mockClient := &mockLLMClient{
		status: 200,
		body: `{
			"content": [],
			"stop_reason": "end_turn"
		}`,
	}

	adapter := heartbeat.NewClaudeAdapter("test-key", "claude-haiku-4-5", mockClient)

	agent := &domain.Agent{ID: "test-agent"}
	result, err := adapter.Run(context.Background(), agent, nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "success" {
		t.Errorf("Status = %q, want %q", result.Status, "success")
	}
	if result.Summary != "" {
		t.Errorf("Summary = %q, want %q", result.Summary, "")
	}
}

func TestClaudeAdapterWithoutIssue(t *testing.T) {
	// Test that the adapter uses the idle prompt when issue is nil
	mockClient := &mockLLMClient{
		status: 200,
		body: `{
			"content": [{"type": "text", "text": "Idle response"}],
			"stop_reason": "end_turn"
		}`,
	}

	adapter := heartbeat.NewClaudeAdapter("test-key", "claude-haiku-4-5", mockClient)

	agent := &domain.Agent{ID: "test-agent"}
	result, err := adapter.Run(context.Background(), agent, nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "success" {
		t.Errorf("Status = %q, want %q", result.Status, "success")
	}
	if result.Summary != "Idle response" {
		t.Errorf("Summary = %q, want %q", result.Summary, "Idle response")
	}
}

func TestClaudeAdapterTransportError(t *testing.T) {
	// Mock a transport-level error (network failure)
	mockClient := &mockLLMClient{
		err: errors.New("dial tcp: connection refused"),
	}

	adapter := heartbeat.NewClaudeAdapter("test-key", "claude-haiku-4-5", mockClient)

	agent := &domain.Agent{ID: "test-agent"}
	result, err := adapter.Run(context.Background(), agent, nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "error" {
		t.Errorf("Status = %q, want %q", result.Status, "error")
	}
	if result.Summary == "" {
		t.Errorf("Summary should not be empty on transport error")
	}
}
