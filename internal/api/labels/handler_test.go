package labels_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	apilabels "github.com/ubunatic/paperclip-go/internal/api/labels"
	"github.com/ubunatic/paperclip-go/internal/labels"
	"github.com/ubunatic/paperclip-go/internal/store"
)

func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatalf("newTestStore: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestHandlerList_MissingCompanyID(t *testing.T) {
	s := newTestStore(t)
	svc := labels.New(s)
	handler := apilabels.Handler(svc)

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	errObj, ok := resp["error"].(map[string]any)
	if !ok || errObj["code"] != "validation_error" {
		t.Errorf("error code not validation_error: %v", resp)
	}
}

func TestHandlerCreate_MissingFields(t *testing.T) {
	s := newTestStore(t)
	svc := labels.New(s)
	handler := apilabels.Handler(svc)

	// Missing color
	body, _ := json.Marshal(map[string]string{
		"companyId": "c1",
		"name":      "bug",
	})

	req, err := http.NewRequest("POST", "/", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", w.Code)
	}
}

func TestHandlerCreate_DuplicateLabel(t *testing.T) {
	s := newTestStore(t)
	svc := labels.New(s)
	handler := apilabels.Handler(svc)
	ctx := context.Background()

	// Create company first
	_, err := s.DB.ExecContext(ctx,
		`INSERT INTO companies(id, name, shortname, description, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		"c1", "Test", "test", "desc", "2024-01-01T00:00:00Z", "2024-01-01T00:00:00Z",
	)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	// Create first label
	_, err = svc.Create(ctx, "c1", "bug", "#ff0000")
	if err != nil {
		t.Fatalf("Create first: %v", err)
	}

	// Try to create duplicate
	body, _ := json.Marshal(map[string]string{
		"companyId": "c1",
		"name":      "bug",
		"color":     "#00ff00",
	})

	req, err := http.NewRequest("POST", "/", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want 409", w.Code)
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	errObj, ok := resp["error"].(map[string]any)
	if !ok || errObj["code"] != "duplicate_label" {
		t.Errorf("error code not duplicate_label: %v", resp)
	}
}

