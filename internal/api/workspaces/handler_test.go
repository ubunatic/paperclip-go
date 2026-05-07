package workspaces_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	apiworkspaces "github.com/ubunatic/paperclip-go/internal/api/workspaces"
	"github.com/ubunatic/paperclip-go/internal/ids"
	"github.com/ubunatic/paperclip-go/internal/store"
	"github.com/ubunatic/paperclip-go/internal/testutil"
	"github.com/ubunatic/paperclip-go/internal/workspaces"
)

func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	return testutil.NewStore(t)
}

func setupTestCompany(t *testing.T, s *store.Store) string {
	t.Helper()
	ctx := context.Background()

	companyID := "company-test-" + ids.NewUUID()
	_, err := s.DB.ExecContext(ctx,
		`INSERT INTO companies(id, name, shortname, description, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		companyID, "Test Company", "test-"+ids.NewUUID()[:8], "Test", "2024-01-01T00:00:00Z", "2024-01-01T00:00:00Z",
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
	_, err := s.DB.ExecContext(ctx,
		`INSERT INTO agents(id, company_id, shortname, display_name, role, adapter, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		agentID, companyID, "test-agent-"+ids.NewUUID()[:8], "Test Agent", "test", "stub", "2024-01-01T00:00:00Z", "2024-01-01T00:00:00Z",
	)
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}

	return agentID
}

func extractWorkspaceObject(t *testing.T, body *bytes.Buffer) map[string]any {
	t.Helper()
	var resp map[string]any
	if err := json.NewDecoder(body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	return resp
}

func extractWorkspaceList(t *testing.T, body *bytes.Buffer) []any {
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

func TestHandlerList_MissingCompanyID(t *testing.T) {
	s := newTestStore(t)
	svc := workspaces.New(s)
	handler := apiworkspaces.Handler(svc)

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", w.Code)
	}
}

func TestHandlerList_EmptyForCompany(t *testing.T) {
	s := newTestStore(t)
	svc := workspaces.New(s)
	handler := apiworkspaces.Handler(svc)

	companyID := setupTestCompany(t, s)

	req, err := http.NewRequest("GET", "/?companyId="+companyID, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}

	list := extractWorkspaceList(t, w.Body)
	if len(list) != 0 {
		t.Errorf("expected empty list, got %d items", len(list))
	}
}

func TestHandlerList_Success(t *testing.T) {
	s := newTestStore(t)
	svc := workspaces.New(s)
	handler := apiworkspaces.Handler(svc)

	companyID := setupTestCompany(t, s)
	agentID := setupTestAgent(t, s, companyID)

	// Create two workspaces
	for i := 0; i < 2; i++ {
		body, _ := json.Marshal(map[string]string{
			"companyId": companyID,
			"agentId":   agentID,
			"path":      "/workspace-" + ids.NewUUID()[:8],
		})

		req, _ := http.NewRequest("POST", "/", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}

	// List workspaces
	req, err := http.NewRequest("GET", "/?companyId="+companyID, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}

	list := extractWorkspaceList(t, w.Body)
	if len(list) < 2 {
		t.Errorf("expected at least 2 items, got %d", len(list))
	}
}

func TestHandlerCreate_InvalidJSON(t *testing.T) {
	s := newTestStore(t)
	svc := workspaces.New(s)
	handler := apiworkspaces.Handler(svc)

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
	svc := workspaces.New(s)
	handler := apiworkspaces.Handler(svc)

	companyID := setupTestCompany(t, s)

	// Missing path
	body, _ := json.Marshal(map[string]string{
		"companyId": companyID,
		"agentId":   "test-agent",
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

func TestHandlerCreate_BlankAgentID(t *testing.T) {
	s := newTestStore(t)
	svc := workspaces.New(s)
	handler := apiworkspaces.Handler(svc)

	companyID := setupTestCompany(t, s)

	body, _ := json.Marshal(map[string]string{
		"companyId": companyID,
		"agentId":   "   ",
		"path":      "/workspace",
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

func TestHandlerCreate_InvalidStatus(t *testing.T) {
	s := newTestStore(t)
	svc := workspaces.New(s)
	handler := apiworkspaces.Handler(svc)

	companyID := setupTestCompany(t, s)
	agentID := setupTestAgent(t, s, companyID)

	body, _ := json.Marshal(map[string]string{
		"companyId": companyID,
		"agentId":   agentID,
		"path":      "/workspace",
		"status":    "unknown",
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
	svc := workspaces.New(s)
	handler := apiworkspaces.Handler(svc)

	companyID := setupTestCompany(t, s)
	agentID := setupTestAgent(t, s, companyID)

	body, _ := json.Marshal(map[string]string{
		"companyId": companyID,
		"agentId":   agentID,
		"path":      "/workspace",
		"status":    "active",
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

	resp := extractWorkspaceObject(t, w.Body)
	if resp["agentId"] != agentID {
		t.Errorf("agentId = %v, want %v", resp["agentId"], agentID)
	}
	if resp["path"] != "/workspace" {
		t.Errorf("path = %v, want /workspace", resp["path"])
	}
	if resp["status"] != "active" {
		t.Errorf("status = %v, want active", resp["status"])
	}
	if _, hasID := resp["id"]; !hasID {
		t.Error("response missing id field")
	}
}

func TestHandlerCreate_DefaultStatus(t *testing.T) {
	s := newTestStore(t)
	svc := workspaces.New(s)
	handler := apiworkspaces.Handler(svc)

	companyID := setupTestCompany(t, s)
	agentID := setupTestAgent(t, s, companyID)

	// Create without status (should default to "active")
	body, _ := json.Marshal(map[string]string{
		"companyId": companyID,
		"agentId":   agentID,
		"path":      "/workspace",
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

	resp := extractWorkspaceObject(t, w.Body)
	if resp["status"] != "active" {
		t.Errorf("status = %v, want active", resp["status"])
	}
}

func TestHandlerCreate_DuplicateAgentPath(t *testing.T) {
	s := newTestStore(t)
	svc := workspaces.New(s)
	handler := apiworkspaces.Handler(svc)

	companyID := setupTestCompany(t, s)
	agentID := setupTestAgent(t, s, companyID)

	// Create first workspace
	body, _ := json.Marshal(map[string]string{
		"companyId": companyID,
		"agentId":   agentID,
		"path":      "/workspace",
	})

	req, _ := http.NewRequest("POST", "/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Try to create duplicate (same agent + path)
	body, _ = json.Marshal(map[string]string{
		"companyId": companyID,
		"agentId":   agentID,
		"path":      "/workspace",
	})

	req, _ = http.NewRequest("POST", "/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want 409", w.Code)
	}
}

func TestHandlerGet_NotFound(t *testing.T) {
	s := newTestStore(t)
	svc := workspaces.New(s)
	handler := apiworkspaces.Handler(svc)

	req, err := http.NewRequest("GET", "/nonexistent", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestHandlerGet_Success(t *testing.T) {
	s := newTestStore(t)
	svc := workspaces.New(s)
	handler := apiworkspaces.Handler(svc)

	companyID := setupTestCompany(t, s)
	agentID := setupTestAgent(t, s, companyID)

	// Create a workspace
	body, _ := json.Marshal(map[string]string{
		"companyId": companyID,
		"agentId":   agentID,
		"path":      "/workspace",
	})

	req, _ := http.NewRequest("POST", "/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	created := extractWorkspaceObject(t, w.Body)
	workspaceID := created["id"].(string)

	// Get the workspace
	req, err := http.NewRequest("GET", "/"+workspaceID, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}

	resp := extractWorkspaceObject(t, w.Body)
	if resp["id"] != workspaceID {
		t.Errorf("id = %v, want %v", resp["id"], workspaceID)
	}
	if resp["agentId"] != agentID {
		t.Errorf("agentId = %v, want %v", resp["agentId"], agentID)
	}
}

func TestHandlerDelete_NotFound(t *testing.T) {
	s := newTestStore(t)
	svc := workspaces.New(s)
	handler := apiworkspaces.Handler(svc)

	req, err := http.NewRequest("DELETE", "/nonexistent", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestHandlerDelete_Success(t *testing.T) {
	s := newTestStore(t)
	svc := workspaces.New(s)
	handler := apiworkspaces.Handler(svc)

	companyID := setupTestCompany(t, s)
	agentID := setupTestAgent(t, s, companyID)

	// Create a workspace
	body, _ := json.Marshal(map[string]string{
		"companyId": companyID,
		"agentId":   agentID,
		"path":      "/workspace",
	})

	req, _ := http.NewRequest("POST", "/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	created := extractWorkspaceObject(t, w.Body)
	workspaceID := created["id"].(string)

	// Delete the workspace
	req, err := http.NewRequest("DELETE", "/"+workspaceID, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 204", w.Code)
	}
}
