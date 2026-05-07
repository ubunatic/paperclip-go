package issues_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	apiissues "github.com/ubunatic/paperclip-go/internal/api/issues"
	"github.com/ubunatic/paperclip-go/internal/activity"
	"github.com/ubunatic/paperclip-go/internal/comments"
	"github.com/ubunatic/paperclip-go/internal/ids"
	"github.com/ubunatic/paperclip-go/internal/interactions"
	"github.com/ubunatic/paperclip-go/internal/issues"
	"github.com/ubunatic/paperclip-go/internal/labels"
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

func setupTestIssue(t *testing.T, s *store.Store, companyID string) string {
	t.Helper()
	ctx := context.Background()

	issueID := "issue-test-" + ids.NewUUID()
	_, err := s.DB.ExecContext(ctx,
		`INSERT INTO issues(id, company_id, title, status, created_at, updated_at, documents, work_products) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		issueID, companyID, "Test Issue", "open", "2024-01-01T00:00:00Z", "2024-01-01T00:00:00Z", "[]", "[]",
	)
	if err != nil {
		t.Fatalf("create issue: %v", err)
	}

	return issueID
}

func newTestHandler(t *testing.T, s *store.Store) http.Handler {
	t.Helper()
	return apiissues.Handler(
		issues.New(s), comments.New(s), labels.New(s),
		activity.New(s), interactions.New(s),
	)
}

func extractIssueObject(t *testing.T, body *bytes.Buffer) map[string]any {
	t.Helper()
	var resp map[string]any
	if err := json.NewDecoder(body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	return resp
}

func extractIssueList(t *testing.T, body *bytes.Buffer) []any {
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
	handler := newTestHandler(t, s)

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

func TestHandlerList_InvalidStatus(t *testing.T) {
	s := newTestStore(t)
	handler := newTestHandler(t, s)

	companyID := setupTestCompany(t, s)

	req, err := http.NewRequest("GET", "/?companyId="+companyID+"&status=invalid_status", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", w.Code)
	}
}

func TestHandlerCreate_InvalidJSON(t *testing.T) {
	s := newTestStore(t)
	handler := newTestHandler(t, s)

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
	handler := newTestHandler(t, s)

	// Missing title
	body, _ := json.Marshal(map[string]string{
		"companyId": "c1",
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
	handler := newTestHandler(t, s)

	companyID := setupTestCompany(t, s)

	body, _ := json.Marshal(map[string]string{
		"companyId": companyID,
		"title":     "Test Issue",
		"status":    "invalid_status",
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
	handler := newTestHandler(t, s)

	companyID := setupTestCompany(t, s)

	body, _ := json.Marshal(map[string]string{
		"companyId": companyID,
		"title":     "New Issue",
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

	resp := extractIssueObject(t, w.Body)
	if resp["companyId"] != companyID {
		t.Errorf("companyId = %v, want %s", resp["companyId"], companyID)
	}
	if resp["title"] != "New Issue" {
		t.Errorf("title = %v, want New Issue", resp["title"])
	}
	if _, hasID := resp["id"]; !hasID {
		t.Error("response missing id field")
	}
}

func TestHandlerGet_NotFound(t *testing.T) {
	s := newTestStore(t)
	handler := newTestHandler(t, s)

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
	handler := newTestHandler(t, s)

	companyID := setupTestCompany(t, s)
	issueID := setupTestIssue(t, s, companyID)

	// Empty patch body with no fields
	body, _ := json.Marshal(map[string]any{})

	req, err := http.NewRequest("PATCH", "/"+issueID, bytes.NewReader(body))
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

func TestHandlerCheckout_MissingAgentID(t *testing.T) {
	s := newTestStore(t)
	handler := newTestHandler(t, s)

	companyID := setupTestCompany(t, s)
	issueID := setupTestIssue(t, s, companyID)

	body, _ := json.Marshal(map[string]string{})

	req, err := http.NewRequest("POST", "/"+issueID+"/checkout", bytes.NewReader(body))
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

func TestHandlerCheckout_Conflict(t *testing.T) {
	s := newTestStore(t)
	handler := newTestHandler(t, s)
	issueSvc := issues.New(s)

	companyID := setupTestCompany(t, s)
	agentID := setupTestAgent(t, s, companyID)
	issueID := setupTestIssue(t, s, companyID)

	// First checkout succeeds
	err := issueSvc.Checkout(context.Background(), issueID, agentID)
	if err != nil {
		t.Fatalf("First Checkout: %v", err)
	}

	// Second checkout should fail
	body, _ := json.Marshal(map[string]string{
		"agentId": "another-agent-id",
	})

	req, err := http.NewRequest("POST", "/"+issueID+"/checkout", bytes.NewReader(body))
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
	if !ok || errObj["code"] != "checkout_conflict" {
		t.Errorf("error code not checkout_conflict: %v", resp)
	}
}
