package heartbeat_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sync/atomic"
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

// capturingLLMClient captures the raw request body on each call.
type capturingLLMClient struct {
	body   string
	status int
	bodies [][]byte
}

func (c *capturingLLMClient) Do(req *http.Request) (*http.Response, error) {
	data, _ := io.ReadAll(req.Body)
	c.bodies = append(c.bodies, data)
	return &http.Response{
		StatusCode: c.status,
		Body:       io.NopCloser(bytes.NewBufferString(c.body)),
		Header:     make(http.Header),
	}, nil
}

// sequentialLLMClient returns different responses per call using a counter.
type sequentialLLMClient struct {
	responses []mockLLMClient
	callCount int32
}

func (s *sequentialLLMClient) Do(req *http.Request) (*http.Response, error) {
	idx := int(atomic.AddInt32(&s.callCount, 1)) - 1
	if idx >= len(s.responses) {
		idx = len(s.responses) - 1
	}
	m := s.responses[idx]
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

func TestClaudeAdapterUsesConfiguration(t *testing.T) {
	successBody := `{"content": [{"type": "text", "text": "ok"}], "stop_reason": "end_turn"}`
	client := &capturingLLMClient{
		status: 200,
		body:   successBody,
	}

	adapter := heartbeat.NewClaudeAdapter("test-key", "claude-haiku-4-5", client)

	agent := &domain.Agent{
		ID:          "cfg-agent",
		DisplayName: "MyAgent",
		Configuration: map[string]any{
			"max_tokens":    float64(512),
			"system_prompt": "Custom prompt",
		},
	}

	result, err := adapter.Run(context.Background(), agent, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "success" {
		t.Errorf("Status = %q, want %q", result.Status, "success")
	}
	if len(client.bodies) == 0 {
		t.Fatal("no request body captured")
	}

	var captured map[string]json.RawMessage
	if err := json.Unmarshal(client.bodies[0], &captured); err != nil {
		t.Fatalf("failed to unmarshal captured body: %v", err)
	}

	var maxTokens int
	if err := json.Unmarshal(captured["max_tokens"], &maxTokens); err != nil {
		t.Fatalf("failed to unmarshal max_tokens: %v", err)
	}
	if maxTokens != 512 {
		t.Errorf("max_tokens = %d, want 512", maxTokens)
	}

	var system string
	if err := json.Unmarshal(captured["system"], &system); err != nil {
		t.Fatalf("failed to unmarshal system: %v", err)
	}
	if system != "Custom prompt" {
		t.Errorf("system = %q, want %q", system, "Custom prompt")
	}
}

func TestClaudeAdapterRetries429(t *testing.T) {
	successBody := `{"content": [{"type": "text", "text": "eventual success"}], "stop_reason": "end_turn"}`
	client := &sequentialLLMClient{
		responses: []mockLLMClient{
			{status: 429, body: `{"error":{"message":"rate limited"}}`},
			{status: 429, body: `{"error":{"message":"rate limited"}}`},
			{status: 200, body: successBody},
		},
	}

	adapter := heartbeat.NewClaudeAdapter("test-key", "claude-haiku-4-5", client)
	agent := &domain.Agent{ID: "retry-agent"}

	result, err := adapter.Run(context.Background(), agent, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "success" {
		t.Errorf("Status = %q, want %q", result.Status, "success")
	}
	if result.Summary != "eventual success" {
		t.Errorf("Summary = %q, want %q", result.Summary, "eventual success")
	}
	if int(atomic.LoadInt32(&client.callCount)) != 3 {
		t.Errorf("callCount = %d, want 3", client.callCount)
	}
}

func TestClaudeAdapterNoRetryOn400(t *testing.T) {
	client := &sequentialLLMClient{
		responses: []mockLLMClient{
			{status: 400, body: `{"error":{"message":"bad request"}}`},
		},
	}

	adapter := heartbeat.NewClaudeAdapter("test-key", "claude-haiku-4-5", client)
	agent := &domain.Agent{ID: "no-retry-agent"}

	result, err := adapter.Run(context.Background(), agent, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "error" {
		t.Errorf("Status = %q, want %q", result.Status, "error")
	}
	count := int(atomic.LoadInt32(&client.callCount))
	if count != 1 {
		t.Errorf("callCount = %d, want 1 (no retry on 400)", count)
	}
}
