package agents_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	apiagents "github.com/ubunatic/paperclip-go/internal/api/agents"
	"github.com/ubunatic/paperclip-go/internal/activity"
	"github.com/ubunatic/paperclip-go/internal/agents"
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

func setupTestIssue(t *testing.T, s *store.Store, companyID string, assigneeID *string) string {
	t.Helper()
	ctx := context.Background()

	issueID := "issue-test-" + ids.NewUUID()
	_, err := s.DB.ExecContext(ctx,
		`INSERT INTO issues(id, company_id, title, status, assignee_id, created_at, updated_at, documents, work_products) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		issueID, companyID, "Test Issue", "open", assigneeID, "2024-01-01T00:00:00Z", "2024-01-01T00:00:00Z", "[]", "[]",
	)
	if err != nil {
		t.Fatalf("create issue: %v", err)
	}

	return issueID
}

func extractAgentObject(t *testing.T, body *bytes.Buffer) map[string]any {
	t.Helper()
	var resp map[string]any
	if err := json.NewDecoder(body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	return resp
}

func extractAgentList(t *testing.T, body *bytes.Buffer) []any {
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
	log := activity.New(s)
	svc := agents.New(s, log)
	handler := apiagents.Handler(svc)

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
	log := activity.New(s)
	svc := agents.New(s, log)
	handler := apiagents.Handler(svc)

	// Missing companyId
	body, _ := json.Marshal(map[string]string{
		"shortname":   "test",
		"displayName": "Test Agent",
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
	log := activity.New(s)
	svc := agents.New(s, log)
	handler := apiagents.Handler(svc)

	companyID := setupTestCompany(t, s)

	body, _ := json.Marshal(map[string]string{
		"companyId":   companyID,
		"shortname":   "newagent",
		"displayName": "New Agent",
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

	resp := extractAgentObject(t, w.Body)
	if resp["companyId"] != companyID {
		t.Errorf("companyId = %v, want %s", resp["companyId"], companyID)
	}
	if resp["shortname"] != "newagent" {
		t.Errorf("shortname = %v, want newagent", resp["shortname"])
	}
	if resp["adapter"] != "stub" {
		t.Errorf("adapter = %v, want stub (default)", resp["adapter"])
	}
	if _, hasID := resp["id"]; !hasID {
		t.Error("response missing id field")
	}
}

func TestHandlerGet_NotFound(t *testing.T) {
	s := newTestStore(t)
	log := activity.New(s)
	svc := agents.New(s, log)
	handler := apiagents.Handler(svc)

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

func TestHandlerUpdate_MissingFields(t *testing.T) {
	s := newTestStore(t)
	log := activity.New(s)
	svc := agents.New(s, log)
	handler := apiagents.Handler(svc)

	companyID := setupTestCompany(t, s)
	agentID := setupTestAgent(t, s, companyID)

	// Empty patch body with no fields
	body, _ := json.Marshal(map[string]any{})

	req, err := http.NewRequest("PATCH", "/"+agentID, bytes.NewReader(body))
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

func TestHandlerUpdate_NullConfiguration(t *testing.T) {
	s := newTestStore(t)
	log := activity.New(s)
	svc := agents.New(s, log)
	handler := apiagents.Handler(svc)

	companyID := setupTestCompany(t, s)
	agentID := setupTestAgent(t, s, companyID)

	body, _ := json.Marshal(map[string]any{
		"configuration": nil,
	})

	req, err := http.NewRequest("PATCH", "/"+agentID, bytes.NewReader(body))
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

func TestHandlerPause_NotFound(t *testing.T) {
	s := newTestStore(t)
	log := activity.New(s)
	svc := agents.New(s, log)
	handler := apiagents.Handler(svc)

	req, err := http.NewRequest("POST", "/nonexistent-id/pause", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestHandlerPause_InvalidTransition(t *testing.T) {
	s := newTestStore(t)
	log := activity.New(s)
	svc := agents.New(s, log)
	handler := apiagents.Handler(svc)

	companyID := setupTestCompany(t, s)
	agentID := setupTestAgent(t, s, companyID)

	// Pause the agent first
	_, err := svc.Pause(context.Background(), agentID)
	if err != nil {
		t.Fatalf("First Pause: %v", err)
	}

	// Try to pause again
	req, err := http.NewRequest("POST", "/"+agentID+"/pause", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", w.Code)
	}
}

func TestHandlerDelete_HasActiveDependents(t *testing.T) {
	s := newTestStore(t)
	log := activity.New(s)
	svc := agents.New(s, log)
	handler := apiagents.Handler(svc)

	companyID := setupTestCompany(t, s)
	agentID := setupTestAgent(t, s, companyID)

	// Create an issue assigned to the agent to create a dependent
	setupTestIssue(t, s, companyID, &agentID)

	req, err := http.NewRequest("DELETE", "/"+agentID, nil)
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
	if !ok || errObj["code"] != "has_active_dependents" {
		t.Errorf("error code not has_active_dependents: %v", resp)
	}
}

func TestHandlerList_Success(t *testing.T) {
	s := newTestStore(t)
	log := activity.New(s)
	svc := agents.New(s, log)
	handler := apiagents.Handler(svc)

	companyID := setupTestCompany(t, s)

	// Create two agents in the same company
	setupTestAgent(t, s, companyID)
	setupTestAgent(t, s, companyID)

	req, err := http.NewRequest("GET", "/?companyId="+companyID, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}

	list := extractAgentList(t, w.Body)
	if len(list) != 2 {
		t.Errorf("len(list) = %d, want 2", len(list))
	}
}
