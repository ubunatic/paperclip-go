package health_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	apihealth "github.com/ubunatic/paperclip-go/internal/api/health"
)

func TestHandlerHealth_OK(t *testing.T) {
	handler := apihealth.Handler("test-version")

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}

	if resp["status"] != "ok" {
		t.Errorf("status = %v, want ok", resp["status"])
	}
	if resp["version"] != "test-version" {
		t.Errorf("version = %v, want test-version", resp["version"])
	}
}

func TestHandlerHealth_Fields(t *testing.T) {
	handler := apihealth.Handler("test-version")

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}

	requiredFields := []string{
		"status",
		"version",
		"deploymentMode",
		"deploymentExposure",
		"authReady",
		"bootstrapStatus",
		"bootstrapInviteActive",
		"features",
	}

	for _, field := range requiredFields {
		if _, hasField := resp[field]; !hasField {
			t.Errorf("response missing field: %s", field)
		}
	}
}

func TestHandlerHealth_Features(t *testing.T) {
	handler := apihealth.Handler("test-version")

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}

	features, ok := resp["features"].(map[string]any)
	if !ok {
		t.Fatalf("features is not a map: %v", resp["features"])
	}

	if features["companyDeletionEnabled"] != true {
		t.Errorf("companyDeletionEnabled = %v, want true", features["companyDeletionEnabled"])
	}
}

func TestHandlerHealth_VersionEmbedded(t *testing.T) {
	versionString := "v1.2.3-test"
	handler := apihealth.Handler(versionString)

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}

	if resp["version"] != versionString {
		t.Errorf("version = %v, want %v", resp["version"], versionString)
	}
}
