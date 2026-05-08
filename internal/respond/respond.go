// Package respond provides shared JSON response helpers for HTTP handlers.
// It lives outside the api package so handler sub-packages can import it
// without creating import cycles.
package respond

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
)

type errResponse struct {
	Error errBody `json:"error"`
}

type errBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// JSON serializes v as JSON and writes it to w with the given status code.
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("respond: encode error: %v", err)
	}
}

// Error writes a JSON error envelope with the given HTTP status, code, and message.
func Error(w http.ResponseWriter, status int, code, message string) {
	JSON(w, status, errResponse{Error: errBody{Code: code, Message: message}})
}

// DecodeJSON reads the request body (up to 1 MiB), decodes JSON into dst, and writes error response if needed.
// Returns true on success, false on error.
// Differentiates 413 (RequestEntityTooLarge) for oversized bodies vs. 400 for invalid JSON.
func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MiB
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		// Differentiate between body too large and invalid JSON
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			Error(w, http.StatusRequestEntityTooLarge, "request_entity_too_large", "request body exceeds 1 MiB limit")
		} else {
			Error(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		}
		return false
	}
	return true
}
