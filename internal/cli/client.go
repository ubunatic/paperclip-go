package cli

import (
	"fmt"
	"net/http"
	"os"

	"github.com/ubunatic/paperclip-go/internal/config"
)

// HTTPClient wraps an http.Client and base URL for API calls.
type HTTPClient struct {
	client  *http.Client
	baseURL string
}

// NewHTTPClient creates a new HTTP client pointing to the configured server.
func NewHTTPClient() (*HTTPClient, error) {
	cfg, err := config.Load(config.DefaultPath())
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	// Build base URL from listen address
	// Default: http://127.0.0.1:3200
	baseURL := "http://" + cfg.ListenAddr
	if envURL := os.Getenv("PAPERCLIP_API_URL"); envURL != "" {
		baseURL = envURL
	}

	return &HTTPClient{
		client:  &http.Client{},
		baseURL: baseURL,
	}, nil
}

// Do sends an HTTP request and returns the response.
func (c *HTTPClient) Do(req *http.Request) (*http.Response, error) {
	return c.client.Do(req)
}

// BaseURL returns the configured API base URL.
func (c *HTTPClient) BaseURL() string {
	return c.baseURL
}

// Close closes any idle connections.
func (c *HTTPClient) Close() error {
	c.client.CloseIdleConnections()
	return nil
}
