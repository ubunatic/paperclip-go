package heartbeat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

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
	Messages  []anthropicMessage `json:"messages"`
}

// anthropicMessage represents a message in the Anthropic request.
type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// anthropicResponse represents the response from the Anthropic Messages API.
type anthropicResponse struct {
	Content    []anthropicContent `json:"content"`
	StopReason string             `json:"stop_reason"`
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
	_ = agent // reserved for future prompt personalisation
	// Build the prompt from the issue or use a default idle prompt
	prompt := "Check for any work to do."
	if issue != nil {
		prompt = issue.Title + "\n\n" + issue.Body
	}

	// Build the Anthropic request
	req := anthropicRequest{
		Model:     a.model,
		MaxTokens: 1024,
		Messages: []anthropicMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	// Marshal the request to JSON
	reqBody, err := json.Marshal(req)
	if err != nil {
		return &domain.RunResult{
			Status:  "error",
			Summary: fmt.Sprintf("Failed to marshal request: %v", err),
		}, nil
	}

	// Create the HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(reqBody))
	if err != nil {
		return &domain.RunResult{
			Status:  "error",
			Summary: fmt.Sprintf("Failed to create request: %v", err),
		}, nil
	}

	// Set required headers
	httpReq.Header.Set("x-api-key", a.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("Content-Type", "application/json")

	// Execute the request
	resp, err := a.client.Do(httpReq)
	if err != nil {
		return &domain.RunResult{
			Status:  "error",
			Summary: fmt.Sprintf("Failed to call API: %v", err),
		}, nil
	}
	defer resp.Body.Close()

	// Read the response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return &domain.RunResult{
			Status:  "error",
			Summary: fmt.Sprintf("Failed to read response: %v", err),
		}, nil
	}

	// Handle non-200 status codes
	if resp.StatusCode != 200 {
		// Try to extract error message from response
		var apiErr anthropicError
		if err := json.Unmarshal(respBody, &apiErr); err == nil && apiErr.Error.Message != "" {
			// Return first 200 chars of error message
			errMsg := apiErr.Error.Message
			if len(errMsg) > 200 {
				errMsg = errMsg[:200]
			}
			return &domain.RunResult{
				Status:  "error",
				Summary: errMsg,
			}, nil
		}

		// Fallback to raw body excerpt
		errMsg := string(respBody)
		if len(errMsg) > 200 {
			errMsg = errMsg[:200]
		}
		return &domain.RunResult{
			Status:  "error",
			Summary: fmt.Sprintf("API error %d: %s", resp.StatusCode, errMsg),
		}, nil
	}

	// Parse the successful response
	var apiResp anthropicResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return &domain.RunResult{
			Status:  "error",
			Summary: fmt.Sprintf("Failed to parse response: %v", err),
		}, nil
	}

	// Extract text from content array
	summary := ""
	if len(apiResp.Content) > 0 {
		// Find the first text content block
		for _, content := range apiResp.Content {
			if content.Type == "text" {
				summary = content.Text
				break
			}
		}
	}

	// Trim whitespace
	summary = strings.TrimSpace(summary)

	return &domain.RunResult{
		Status:  "success",
		Summary: summary,
	}, nil
}
