package heartbeat

import "net/http"

// LLMClient defines the interface for making HTTP requests to an LLM service.
// This seam allows testing without making real API calls.
// *http.Client satisfies this interface naturally.
type LLMClient interface {
	Do(req *http.Request) (*http.Response, error)
}
