package companies_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	apicompanies "github.com/ubunatic/paperclip-go/internal/api/companies"
	"github.com/ubunatic/paperclip-go/internal/companies"
	"github.com/ubunatic/paperclip-go/internal/ids"
	"github.com/ubunatic/paperclip-go/internal/store"
	"github.com/ubunatic/paperclip-go/internal/testutil"
)

func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	return testutil.NewStore(t)
}

func setupTestCompany(t *testing.T, s *store.Store) string {
	t.Helper()
	ctx := context.Background()

	companyID := "company-test-" + ids.NewUUID()
	companyShortname := "test-" + ids.NewUUID()[:8]
	_, err := s.DB.ExecContext(ctx,
		`INSERT INTO companies(id, name, shortname, description, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		companyID, "Test Company", companyShortname, "Test", "2024-01-01T00:00:00Z", "2024-01-01T00:00:00Z",
	)
	if err != nil {
		t.Fatalf("create company: %v", err)
	}

	return companyID
}

func setupTestAgent(t *testing.T, s *store.Store, companyID string) string {
	t.Helper()
	ctx := context.Background()

	agentID := "agent-test-" + ids.NewUUID()
	agentShortname := "test-agent-" + ids.NewUUID()[:8]
	_, err := s.DB.ExecContext(ctx,
		`INSERT INTO agents(id, company_id, shortname, display_name, role, adapter, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		agentID, companyID, agentShortname, "Test Agent", "test", "stub", "2024-01-01T00:00:00Z", "2024-01-01T00:00:00Z",
	)
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}

	return agentID
}

func extractCompanyObject(t *testing.T, body *bytes.Buffer) map[string]any {
	t.Helper()
	var resp map[string]any
	if err := json.NewDecoder(body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	return resp
}

func extractCompanyList(t *testing.T, body *bytes.Buffer) []any {
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

func TestHandlerCreate_InvalidJSON(t *testing.T) {
	s := newTestStore(t)
	svc := companies.New(s)
	handler := apicompanies.Handler(svc)

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

func TestHandlerCreate_MissingFields(t *testing.T) {
	s := newTestStore(t)
	svc := companies.New(s)
	handler := apicompanies.Handler(svc)

	// Missing name
	body, _ := json.Marshal(map[string]string{
		"shortname": "test",
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

func TestHandlerCreate_Success(t *testing.T) {
	s := newTestStore(t)
	svc := companies.New(s)
	handler := apicompanies.Handler(svc)

	body, _ := json.Marshal(map[string]string{
		"name":      "New Company",
		"shortname": "newco",
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

	resp := extractCompanyObject(t, w.Body)
	if resp["name"] != "New Company" {
		t.Errorf("name = %v, want New Company", resp["name"])
	}
	if resp["shortname"] != "newco" {
		t.Errorf("shortname = %v, want newco", resp["shortname"])
	}
	if _, hasID := resp["id"]; !hasID {
		t.Error("response missing id field")
	}
}

func TestHandlerUpdate_MissingFields(t *testing.T) {
	s := newTestStore(t)
	svc := companies.New(s)
	handler := apicompanies.Handler(svc)

	companyID := setupTestCompany(t, s)

	body, _ := json.Marshal(map[string]any{})

	req, err := http.NewRequest("PATCH", "/"+companyID, bytes.NewReader(body))
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

func TestHandlerUpdate_EmptyName(t *testing.T) {
	s := newTestStore(t)
	svc := companies.New(s)
	handler := apicompanies.Handler(svc)

	companyID := setupTestCompany(t, s)

	emptyName := ""
	body, _ := json.Marshal(map[string]any{
		"name": emptyName,
	})

	req, err := http.NewRequest("PATCH", "/"+companyID, bytes.NewReader(body))
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

func TestHandlerGet_NotFound(t *testing.T) {
	s := newTestStore(t)
	svc := companies.New(s)
	handler := apicompanies.Handler(svc)

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

func TestHandlerDelete_HasDependents(t *testing.T) {
	s := newTestStore(t)
	svc := companies.New(s)
	handler := apicompanies.Handler(svc)

	companyID := setupTestCompany(t, s)
	// Add an agent to create a dependent
	setupTestAgent(t, s, companyID)

	req, err := http.NewRequest("DELETE", "/"+companyID, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

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
	if !ok || errObj["code"] != "has_dependents" {
		t.Errorf("error code not has_dependents: %v", resp)
	}
}

func TestHandlerList_Success(t *testing.T) {
	s := newTestStore(t)
	svc := companies.New(s)
	handler := apicompanies.Handler(svc)

	// Create two companies
	setupTestCompany(t, s)
	setupTestCompany(t, s)

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}

	list := extractCompanyList(t, w.Body)
	if len(list) < 2 {
		t.Errorf("len(list) = %d, want at least 2", len(list))
	}
}
