// Package respond provides shared JSON response helpers for HTTP handlers.
// It lives outside the api package so handler sub-packages can import it
// without creating import cycles.
package respond

import (
	"encoding/json"
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
