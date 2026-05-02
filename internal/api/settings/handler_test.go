package settings_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	apisettings "github.com/ubunatic/paperclip-go/internal/api/settings"
	"github.com/ubunatic/paperclip-go/internal/settings"
	"github.com/ubunatic/paperclip-go/internal/testutil"
)

func TestHandlerGet_ReturnsDefaults(t *testing.T) {
	s := testutil.NewStore(t)
	svc := settings.New(s)
	ctx := context.Background()

	// Seed defaults
	err := svc.SeedDefaults(ctx, map[string]string{
		"deployment_mode": "local_trusted",
		"allowed_origins": "localhost",
	})
	if err != nil {
		t.Fatalf("seeding defaults: %v", err)
	}

	// Make request
	handler := apisettings.Handler(svc)
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Verify response
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var result map[string]string
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("decoding response: %v", err)
	}

	// Verify keys present
	if result["deployment_mode"] != "local_trusted" {
		t.Errorf("deployment_mode = %q, want 'local_trusted'", result["deployment_mode"])
	}
	if result["allowed_origins"] != "localhost" {
		t.Errorf("allowed_origins = %q, want 'localhost'", result["allowed_origins"])
	}
}

func TestHandlerPatch_UpdatesValue(t *testing.T) {
	s := testutil.NewStore(t)
	svc := settings.New(s)
	ctx := context.Background()

	// Seed defaults
	err := svc.SeedDefaults(ctx, map[string]string{
		"deployment_mode": "local_trusted",
		"allowed_origins": "localhost",
	})
	if err != nil {
		t.Fatalf("seeding defaults: %v", err)
	}

	// Make PATCH request
	handler := apisettings.Handler(svc)
	patchBody, _ := json.Marshal(map[string]string{
		"deployment_mode": "cloud",
	})
	req := httptest.NewRequest("PATCH", "/", bytes.NewReader(patchBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Verify response
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var result map[string]string
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("decoding response: %v", err)
	}

	// Verify updated
	if result["deployment_mode"] != "cloud" {
		t.Errorf("deployment_mode = %q, want 'cloud'", result["deployment_mode"])
	}
	if result["allowed_origins"] != "localhost" {
		t.Errorf("allowed_origins = %q, want 'localhost'", result["allowed_origins"])
	}
}

func TestHandlerPatch_InvalidJSON(t *testing.T) {
	s := testutil.NewStore(t)
	svc := settings.New(s)

	// Make PATCH request with malformed JSON
	handler := apisettings.Handler(svc)
	req := httptest.NewRequest("PATCH", "/", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Verify error
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestHandlerPatch_EmptyObject(t *testing.T) {
	s := testutil.NewStore(t)
	svc := settings.New(s)
	ctx := context.Background()

	// Seed defaults
	err := svc.SeedDefaults(ctx, map[string]string{
		"deployment_mode": "local_trusted",
		"allowed_origins": "localhost",
	})
	if err != nil {
		t.Fatalf("seeding defaults: %v", err)
	}

	// Make PATCH request with empty object
	handler := apisettings.Handler(svc)
	req := httptest.NewRequest("PATCH", "/", bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Verify response
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var result map[string]string
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("decoding response: %v", err)
	}

	// Verify full map returned
	if result["deployment_mode"] != "local_trusted" {
		t.Errorf("deployment_mode = %q, want 'local_trusted'", result["deployment_mode"])
	}
	if result["allowed_origins"] != "localhost" {
		t.Errorf("allowed_origins = %q, want 'localhost'", result["allowed_origins"])
	}
}
