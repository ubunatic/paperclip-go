package respond

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeJSON_Valid(t *testing.T) {
	var body struct {
		Name string `json:"name"`
	}

	req := httptest.NewRequest("POST", "/", strings.NewReader(`{"name":"test"}`))
	w := httptest.NewRecorder()

	ok := DecodeJSON(w, req, &body)
	if !ok {
		t.Errorf("DecodeJSON returned false, want true")
	}
	if body.Name != "test" {
		t.Errorf("Name = %q, want %q", body.Name, "test")
	}
	// When successful, no response is written
	if w.Body.Len() > 0 {
		t.Errorf("Response body written on success: %s", w.Body.String())
	}
}

func TestDecodeJSON_InvalidJSON(t *testing.T) {
	var body struct {
		Name string `json:"name"`
	}

	req := httptest.NewRequest("POST", "/", strings.NewReader(`{invalid json}`))
	w := httptest.NewRecorder()

	ok := DecodeJSON(w, req, &body)
	if ok {
		t.Errorf("DecodeJSON returned true, want false")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var errResp errResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}
	if errResp.Error.Code != "bad_request" {
		t.Errorf("Error code = %q, want %q", errResp.Error.Code, "bad_request")
	}
}

func TestDecodeJSON_BodyTooLarge(t *testing.T) {
	var body struct {
		Name string `json:"name"`
	}

	// Create a body that's valid JSON but much larger than 1 MiB
	// This needs to be properly formatted JSON to trigger MaxBytesError during decode
	largeBody := "{" + string(bytes.Repeat([]byte("\"a\":\"xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx\","), 10000)) + "\"b\":2}"

	req := httptest.NewRequest("POST", "/", bytes.NewReader([]byte(largeBody)))
	w := httptest.NewRecorder()

	ok := DecodeJSON(w, req, &body)
	if ok {
		t.Errorf("DecodeJSON returned true, want false")
	}
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusRequestEntityTooLarge)
	}

	var errResp errResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}
	if errResp.Error.Code != "request_entity_too_large" {
		t.Errorf("Error code = %q, want %q", errResp.Error.Code, "request_entity_too_large")
	}
}

func TestDecodeJSON_EmptyBody(t *testing.T) {
	var body struct {
		Name string `json:"name"`
	}

	req := httptest.NewRequest("POST", "/", strings.NewReader(""))
	w := httptest.NewRecorder()

	ok := DecodeJSON(w, req, &body)
	if ok {
		t.Errorf("DecodeJSON returned true, want false")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var errResp errResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}
	if errResp.Error.Code != "bad_request" {
		t.Errorf("Error code = %q, want %q", errResp.Error.Code, "bad_request")
	}
}
