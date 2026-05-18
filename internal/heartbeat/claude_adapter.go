package heartbeat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ubunatic/paperclip-go/internal/domain"
)

// ClaudeAdapter implements the Adapter interface for the Anthropic Claude API.
type ClaudeAdapter struct {
	apiKey string
	model  string
	client LLMClient
}

// NewClaudeAdapter creates a new ClaudeAdapter with the given API key, model, and HTTP client.
func NewClaudeAdapter(apiKey, model string, client LLMClient) *ClaudeAdapter {
	return &ClaudeAdapter{
		apiKey: apiKey,
		model:  model,
		client: client,
	}
}

// anthropicRequest represents the request body for the Anthropic Messages API.
type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system,omitempty"`
	Messages  []anthropicMessage `json:"messages"`
}

// anthropicMessage represents a message in the Anthropic request.
type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// anthropicUsage holds token counts from the Anthropic API response.
type anthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// anthropicResponse represents the response from the Anthropic Messages API.
type anthropicResponse struct {
	Content    []anthropicContent `json:"content"`
	StopReason string             `json:"stop_reason"`
	Usage      anthropicUsage     `json:"usage"`
}

// anthropicContent represents a content block in the Anthropic response.
type anthropicContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// anthropicError represents an error response from the Anthropic API.
type anthropicError struct {
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

// Run implements the Adapter interface for ClaudeAdapter.
// It calls the Anthropic Messages API with a prompt based on the agent and optional issue.
// Returns RunResult with status "success" or "error" (never returns a Go error).
func (a *ClaudeAdapter) Run(ctx context.Context, agent *domain.Agent, issue *domain.Issue) (*domain.RunResult, error) {
	// Resolve max_tokens from agent configuration, defaulting to 1024.
	maxTokens := 1024
	if agent != nil {
		if v, ok := agent.Configuration["max_tokens"]; ok {
			if f, ok := v.(float64); ok && f > 0 {
				maxTokens = int(f)
			}
		}
	}

	// Resolve system prompt from agent configuration.
	systemPrompt := ""
	if agent != nil {
		if v, ok := agent.Configuration["system_prompt"]; ok {
			if s, ok := v.(string); ok {
				systemPrompt = s
			}
		}
	}
	if systemPrompt == "" {
		if agent != nil && agent.DisplayName != "" {
			systemPrompt = "You are " + agent.DisplayName + ", an AI agent."
		} else {
			systemPrompt = "You are a helpful AI agent."
		}
	}

	// Build the prompt from the issue or use a default idle prompt.
	prompt := "Check for any work to do and report your status."
	if issue != nil {
		parts := []string{"# " + issue.Title}
		if issue.Body != "" {
			parts = append(parts, issue.Body)
		}
		prompt = strings.Join(parts, "\n\n")
	}

	// Build the Anthropic request.
	req := anthropicRequest{
		Model:     a.model,
		MaxTokens: maxTokens,
		System:    systemPrompt,
		Messages: []anthropicMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	// Execute with retry logic (up to 3 attempts, exponential backoff 1s, 2s).
	var resp *http.Response
	var doErr error
	backoff := time.Second
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
				backoff *= 2
			}
		}

		// Re-marshal request body for each attempt (body is read once per request).
		bodyBytes, err := json.Marshal(req)
		if err != nil {
			return &domain.RunResult{
				Status:  "error",
				Summary: fmt.Sprintf("Failed to marshal request: %v", err),
			}, nil
		}

		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.anthropic.com/v1/messages", bytes.NewReader(bodyBytes))
		if err != nil {
			return &domain.RunResult{
				Status:  "error",
				Summary: fmt.Sprintf("creating request: %v", err),
			}, nil
		}
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("x-api-key", a.apiKey)
		httpReq.Header.Set("anthropic-version", "2023-06-01")

		resp, doErr = a.client.Do(httpReq)
		if doErr != nil {
			continue // network error, retry
		}
		if resp.StatusCode == 429 || resp.StatusCode >= 500 {
			resp.Body.Close()
			resp = nil
			continue // transient, retry
		}
		break // success or non-retryable error
	}

	if doErr != nil {
		return &domain.RunResult{
			Status:  "error",
			Summary: fmt.Sprintf("HTTP request failed after retries: %v", doErr),
		}, nil
	}

	defer resp.Body.Close()

	// Read the response body.
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return &domain.RunResult{
			Status:  "error",
			Summary: fmt.Sprintf("Failed to read response: %v", err),
		}, nil
	}

	// Handle non-200 status codes.
	if resp.StatusCode != 200 {
		// Try to extract error message from response.
		var apiErr anthropicError
		if err := json.Unmarshal(respBody, &apiErr); err == nil && apiErr.Error.Message != "" {
			errMsg := apiErr.Error.Message
			if len(errMsg) > 200 {
				errMsg = errMsg[:200]
			}
			return &domain.RunResult{
				Status:  "error",
				Summary: errMsg,
			}, nil
		}

		// Fallback to raw body excerpt.
		errMsg := string(respBody)
		if len(errMsg) > 200 {
			errMsg = errMsg[:200]
		}
		return &domain.RunResult{
			Status:  "error",
			Summary: fmt.Sprintf("API error %d: %s", resp.StatusCode, errMsg),
		}, nil
	}

	// Parse the successful response.
	var apiResp anthropicResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return &domain.RunResult{
			Status:  "error",
			Summary: fmt.Sprintf("Failed to parse response: %v", err),
		}, nil
	}

	// Extract text from content array.
	summary := ""
	for _, content := range apiResp.Content {
		if content.Type == "text" {
			summary = content.Text
			break
		}
	}

	summary = strings.TrimSpace(summary)

	return &domain.RunResult{
		Status:           "success",
		Summary:          summary,
		PromptTokens:     apiResp.Usage.InputTokens,
		CompletionTokens: apiResp.Usage.OutputTokens,
	}, nil
}
