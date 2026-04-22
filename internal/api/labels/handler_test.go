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
	"github.com/ubunatic/paperclip-go/internal/testutil"
)

func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	return testutil.NewStore(t)
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
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
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
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	errObj, ok := resp["error"].(map[string]any)
	if !ok || errObj["code"] != "duplicate_label" {
		t.Errorf("error code not duplicate_label: %v", resp)
	}
}

func createTestCompany(t *testing.T, s *store.Store, id string) {
	t.Helper()

	_, err := s.DB.ExecContext(context.Background(),
		`INSERT INTO companies(id, name, shortname, description, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		id, "Test", "test", "desc", "2024-01-01T00:00:00Z", "2024-01-01T00:00:00Z",
	)
	if err != nil {
		t.Fatalf("setup company: %v", err)
	}
}

func extractLabelList(t *testing.T, body *bytes.Buffer) []any {
	t.Helper()

	var resp any
	if err := json.NewDecoder(body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}

	if list, ok := resp.([]any); ok {
		return list
	}

	obj, ok := resp.(map[string]any)
	if !ok {
		t.Fatalf("response is neither array nor object: %T", resp)
	}

	list, ok := obj["items"].([]any)
	if !ok {
		t.Fatalf("response does not contain items array: %v", obj)
	}

	return list
}

func extractLabelObject(t *testing.T, body *bytes.Buffer) map[string]any {
	t.Helper()

	var resp map[string]any
	if err := json.NewDecoder(body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}

	return resp
}

func TestHandlerList_Success(t *testing.T) {
	s := newTestStore(t)
	svc := labels.New(s)
	handler := apilabels.Handler(svc)

	createTestCompany(t, s, "c1")

	if _, err := svc.Create(context.Background(), "c1", "bug", "#ff0000"); err != nil {
		t.Fatalf("Create bug: %v", err)
	}
	if _, err := svc.Create(context.Background(), "c1", "feature", "#00ff00"); err != nil {
		t.Fatalf("Create feature: %v", err)
	}

	req, err := http.NewRequest("GET", "/?companyId=c1", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	list := extractLabelList(t, w.Body)
	if len(list) != 2 {
		t.Fatalf("len(list) = %d, want 2", len(list))
	}
}

func TestHandlerCreate_Success(t *testing.T) {
	s := newTestStore(t)
	svc := labels.New(s)
	handler := apilabels.Handler(svc)

	createTestCompany(t, s, "c1")

	body, _ := json.Marshal(map[string]string{
		"companyId": "c1",
		"name":      "bug",
		"color":     "#ff0000",
	})

	req, err := http.NewRequest("POST", "/", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", w.Code)
	}

	resp := extractLabelObject(t, w.Body)
	if resp["name"] != "bug" {
		t.Errorf("name = %v, want bug", resp["name"])
	}
	if resp["color"] != "#ff0000" {
		t.Errorf("color = %v, want #ff0000", resp["color"])
	}
	if resp["companyId"] != "c1" {
		t.Errorf("companyId = %v, want c1", resp["companyId"])
	}
}

func TestHandlerGet_Success(t *testing.T) {
	s := newTestStore(t)
	svc := labels.New(s)
	handler := apilabels.Handler(svc)

	createTestCompany(t, s, "c1")

	created, err := svc.Create(context.Background(), "c1", "bug", "#ff0000")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	req, err := http.NewRequest("GET", "/"+created.ID, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	resp := extractLabelObject(t, w.Body)
	if resp["id"] != created.ID {
		t.Errorf("id = %v, want %s", resp["id"], created.ID)
	}
	if resp["name"] != "bug" {
		t.Errorf("name = %v, want bug", resp["name"])
	}
}

func TestHandlerGet_NotFound(t *testing.T) {
	s := newTestStore(t)
	svc := labels.New(s)
	handler := apilabels.Handler(svc)

	req, err := http.NewRequest("GET", "/does-not-exist", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestHandlerDelete_Success(t *testing.T) {
	s := newTestStore(t)
	svc := labels.New(s)
	handler := apilabels.Handler(svc)

	createTestCompany(t, s, "c1")

	created, err := svc.Create(context.Background(), "c1", "bug", "#ff0000")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	req, err := http.NewRequest("DELETE", "/"+created.ID, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", w.Code)
	}
}

func TestHandlerDelete_NotFound(t *testing.T) {
	s := newTestStore(t)
	svc := labels.New(s)
	handler := apilabels.Handler(svc)

	req, err := http.NewRequest("DELETE", "/does-not-exist", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

