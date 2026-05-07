package routines_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	apiroutines "github.com/ubunatic/paperclip-go/internal/api/routines"
	"github.com/ubunatic/paperclip-go/internal/ids"
	"github.com/ubunatic/paperclip-go/internal/routines"
	"github.com/ubunatic/paperclip-go/internal/store"
	"github.com/ubunatic/paperclip-go/internal/testutil"
)

func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	return testutil.NewStore(t)
}

func setupRoutineTestData(t *testing.T, s *store.Store) (companyID, agentID string) {
	t.Helper()
	ctx := context.Background()

	companyID = "company-test-" + ids.NewUUID()
	_, err := s.DB.ExecContext(ctx,
		`INSERT INTO companies(id, name, shortname, description, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		companyID, "Test Company", "test", "Test", "2024-01-01T00:00:00Z", "2024-01-01T00:00:00Z",
	)
	if err != nil {
		t.Fatalf("create company: %v", err)
	}

	agentID = "agent-test-" + ids.NewUUID()
	_, err = s.DB.ExecContext(ctx,
		`INSERT INTO agents(id, company_id, shortname, display_name, role, adapter, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		agentID, companyID, "test-agent", "Test Agent", "test", "stub", "2024-01-01T00:00:00Z", "2024-01-01T00:00:00Z",
	)
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}

	return companyID, agentID
}

func extractRoutineObject(t *testing.T, body *bytes.Buffer) map[string]any {
	t.Helper()
	var resp map[string]any
	if err := json.NewDecoder(body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	return resp
}

func extractRoutineList(t *testing.T, body *bytes.Buffer) []any {
	t.Helper()
	var resp map[string]any
	if err := json.NewDecoder(body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	list, ok := resp["items"].([]any)
	if !ok {
		t.Fatalf("response does not contain items array: %v", resp)
	}
	return list
}

func TestHandlerListMissingCompanyID(t *testing.T) {
	s := newTestStore(t)
	svc := routines.New(s)
	handler := apiroutines.Handler(svc)

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", w.Code)
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

func TestHandlerCreateInvalidJSON(t *testing.T) {
	s := newTestStore(t)
	svc := routines.New(s)
	handler := apiroutines.Handler(svc)

	body := bytes.NewReader([]byte("invalid json"))
	req, err := http.NewRequest("POST", "/", body)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestHandlerCreateMissingFields(t *testing.T) {
	s := newTestStore(t)
	svc := routines.New(s)
	handler := apiroutines.Handler(svc)

	// Missing cronExpr
	body, _ := json.Marshal(map[string]string{
		"companyId": "c1",
		"agentId":   "a1",
		"name":      "test-routine",
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

func TestHandlerCreateInvalidCron(t *testing.T) {
	s := newTestStore(t)
	svc := routines.New(s)
	handler := apiroutines.Handler(svc)

	companyID, agentID := setupRoutineTestData(t, s)

	body, _ := json.Marshal(map[string]string{
		"companyId": companyID,
		"agentId":   agentID,
		"name":      "test-routine",
		"cronExpr":  "invalid cron",
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

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	errObj, ok := resp["error"].(map[string]any)
	if !ok || errObj["code"] != "invalid_cron" {
		t.Errorf("error code not invalid_cron: %v", resp)
	}
}

func TestHandlerCreateNameConflict(t *testing.T) {
	s := newTestStore(t)
	svc := routines.New(s)
	handler := apiroutines.Handler(svc)

	companyID, agentID := setupRoutineTestData(t, s)

	// Create first routine
	_, err := svc.Create(context.Background(), companyID, agentID, "my-routine", "0 0 * * *")
	if err != nil {
		t.Fatalf("Create first: %v", err)
	}

	// Try to create duplicate name
	body, _ := json.Marshal(map[string]string{
		"companyId": companyID,
		"agentId":   agentID,
		"name":      "my-routine",
		"cronExpr":  "0 1 * * *",
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
	if !ok || errObj["code"] != "name_conflict" {
		t.Errorf("error code not name_conflict: %v", resp)
	}
}

func TestHandlerCreateSuccess(t *testing.T) {
	s := newTestStore(t)
	svc := routines.New(s)
	handler := apiroutines.Handler(svc)

	companyID, agentID := setupRoutineTestData(t, s)

	body, _ := json.Marshal(map[string]string{
		"companyId": companyID,
		"agentId":   agentID,
		"name":      "test-routine",
		"cronExpr":  "0 0 * * *",
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

	resp := extractRoutineObject(t, w.Body)
	if resp["companyId"] != companyID {
		t.Errorf("companyId = %v, want %s", resp["companyId"], companyID)
	}
	if resp["name"] != "test-routine" {
		t.Errorf("name = %v, want test-routine", resp["name"])
	}
	if resp["cronExpr"] != "0 0 * * *" {
		t.Errorf("cronExpr = %v, want 0 0 * * *", resp["cronExpr"])
	}
	// Assert dispatchFingerprint is NOT in the response
	if _, ok := resp["dispatchFingerprint"]; ok {
		t.Errorf("dispatchFingerprint should not be in response, but found: %v", resp["dispatchFingerprint"])
	}
}

func TestHandlerGetNotFound(t *testing.T) {
	s := newTestStore(t)
	svc := routines.New(s)
	handler := apiroutines.Handler(svc)

	req, err := http.NewRequest("GET", "/nonexistent-id", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestHandlerGetSuccess(t *testing.T) {
	s := newTestStore(t)
	svc := routines.New(s)
	handler := apiroutines.Handler(svc)

	companyID, agentID := setupRoutineTestData(t, s)

	created, err := svc.Create(context.Background(), companyID, agentID, "test-routine", "0 0 * * *")
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
		t.Errorf("status = %d, want 200", w.Code)
	}

	resp := extractRoutineObject(t, w.Body)
	if resp["id"] != created.ID {
		t.Errorf("id = %v, want %s", resp["id"], created.ID)
	}
	if resp["name"] != "test-routine" {
		t.Errorf("name = %v, want test-routine", resp["name"])
	}
	// Assert dispatchFingerprint is NOT in the response
	if _, ok := resp["dispatchFingerprint"]; ok {
		t.Errorf("dispatchFingerprint should not be in GET response, but found: %v", resp["dispatchFingerprint"])
	}
}

func TestHandlerUpdateInvalidCron(t *testing.T) {
	s := newTestStore(t)
	svc := routines.New(s)
	handler := apiroutines.Handler(svc)

	companyID, agentID := setupRoutineTestData(t, s)

	created, err := svc.Create(context.Background(), companyID, agentID, "test-routine", "0 0 * * *")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	body, _ := json.Marshal(map[string]string{
		"cronExpr": "invalid cron",
	})

	req, err := http.NewRequest("PATCH", "/"+created.ID, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", w.Code)
	}

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	errObj, ok := resp["error"].(map[string]any)
	if !ok || errObj["code"] != "invalid_cron" {
		t.Errorf("error code not invalid_cron: %v", resp)
	}
}

func TestHandlerDeleteNotFound(t *testing.T) {
	s := newTestStore(t)
	svc := routines.New(s)
	handler := apiroutines.Handler(svc)

	req, err := http.NewRequest("DELETE", "/nonexistent-id", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestHandlerTriggerNotFound(t *testing.T) {
	s := newTestStore(t)
	svc := routines.New(s)
	handler := apiroutines.Handler(svc)

	req, err := http.NewRequest("POST", "/nonexistent-id/trigger", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestHandlerListSuccess(t *testing.T) {
	s := newTestStore(t)
	svc := routines.New(s)
	handler := apiroutines.Handler(svc)

	companyID, agentID := setupRoutineTestData(t, s)

	// Create two routines
	_, err := svc.Create(context.Background(), companyID, agentID, "routine-1", "0 0 * * *")
	if err != nil {
		t.Fatalf("Create 1: %v", err)
	}

	_, err = svc.Create(context.Background(), companyID, agentID, "routine-2", "0 1 * * *")
	if err != nil {
		t.Fatalf("Create 2: %v", err)
	}

	req, err := http.NewRequest("GET", "/?companyId="+companyID, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}

	list := extractRoutineList(t, w.Body)
	if len(list) != 2 {
		t.Errorf("len(list) = %d, want 2", len(list))
	}
}
