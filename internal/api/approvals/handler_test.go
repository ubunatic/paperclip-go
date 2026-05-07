package approvals_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	apiapprovals "github.com/ubunatic/paperclip-go/internal/api/approvals"
	"github.com/ubunatic/paperclip-go/internal/approvals"
	"github.com/ubunatic/paperclip-go/internal/domain"
	"github.com/ubunatic/paperclip-go/internal/ids"
	"github.com/ubunatic/paperclip-go/internal/store"
	"github.com/ubunatic/paperclip-go/internal/testutil"
)

func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	return testutil.NewStore(t)
}

func setupTestData(t *testing.T, s *store.Store) (companyID, agentID, issueID string) {
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

	issueID = "issue-test-" + ids.NewUUID()
	_, err = s.DB.ExecContext(ctx,
		`INSERT INTO issues(id, company_id, title, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		issueID, companyID, "Test Issue", "open", "2024-01-01T00:00:00Z", "2024-01-01T00:00:00Z",
	)
	if err != nil {
		t.Fatalf("create issue: %v", err)
	}

	return companyID, agentID, issueID
}

func extractApprovalObject(t *testing.T, body *bytes.Buffer) map[string]any {
	t.Helper()
	var resp map[string]any
	if err := json.NewDecoder(body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	return resp
}

func extractApprovalList(t *testing.T, body *bytes.Buffer) []any {
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
	svc := approvals.New(s)
	handler := apiapprovals.Handler(svc)

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

func TestHandlerCreate_InvalidJSON(t *testing.T) {
	s := newTestStore(t)
	svc := approvals.New(s)
	handler := apiapprovals.Handler(svc)

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
	svc := approvals.New(s)
	handler := apiapprovals.Handler(svc)

	// Missing kind
	body, _ := json.Marshal(map[string]string{
		"companyId": "c1",
		"agentId":   "a1",
		"issueId":   "i1",
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
	svc := approvals.New(s)
	handler := apiapprovals.Handler(svc)

	companyID, agentID, issueID := setupTestData(t, s)

	body, _ := json.Marshal(map[string]string{
		"companyId": companyID,
		"agentId":   agentID,
		"issueId":   issueID,
		"kind":      "delete_file",
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

	resp := extractApprovalObject(t, w.Body)
	if resp["companyId"] != companyID {
		t.Errorf("companyId = %v, want %s", resp["companyId"], companyID)
	}
	if resp["status"] != string(domain.ApprovalStatusPending) {
		t.Errorf("status = %v, want pending", resp["status"])
	}
	if resp["kind"] != "delete_file" {
		t.Errorf("kind = %v, want delete_file", resp["kind"])
	}
}

func TestHandlerGet_NotFound(t *testing.T) {
	s := newTestStore(t)
	svc := approvals.New(s)
	handler := apiapprovals.Handler(svc)

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

func TestHandlerGet_Success(t *testing.T) {
	s := newTestStore(t)
	svc := approvals.New(s)
	handler := apiapprovals.Handler(svc)

	companyID, agentID, issueID := setupTestData(t, s)

	created, err := svc.Create(context.Background(), companyID, agentID, issueID, "test_kind", nil)
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

	resp := extractApprovalObject(t, w.Body)
	if resp["id"] != created.ID {
		t.Errorf("id = %v, want %s", resp["id"], created.ID)
	}
}

func TestHandlerApprove_Success(t *testing.T) {
	s := newTestStore(t)
	svc := approvals.New(s)
	handler := apiapprovals.Handler(svc)

	companyID, agentID, issueID := setupTestData(t, s)

	created, err := svc.Create(context.Background(), companyID, agentID, issueID, "test_kind", nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	req, err := http.NewRequest("POST", "/"+created.ID+"/approve", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}

	resp := extractApprovalObject(t, w.Body)
	if resp["status"] != string(domain.ApprovalStatusApproved) {
		t.Errorf("status = %v, want approved", resp["status"])
	}
}

func TestHandlerApprove_AlreadyResolved(t *testing.T) {
	s := newTestStore(t)
	svc := approvals.New(s)
	handler := apiapprovals.Handler(svc)

	companyID, agentID, issueID := setupTestData(t, s)

	created, err := svc.Create(context.Background(), companyID, agentID, issueID, "test_kind", nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// First approve succeeds
	_, err = svc.Approve(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("First Approve: %v", err)
	}

	// Second approve via handler should fail
	req, err := http.NewRequest("POST", "/"+created.ID+"/approve", nil)
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
	if !ok || errObj["code"] != "conflict" {
		t.Errorf("error code not conflict: %v", resp)
	}
}

func TestHandlerReject_Success(t *testing.T) {
	s := newTestStore(t)
	svc := approvals.New(s)
	handler := apiapprovals.Handler(svc)

	companyID, agentID, issueID := setupTestData(t, s)

	created, err := svc.Create(context.Background(), companyID, agentID, issueID, "test_kind", nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	req, err := http.NewRequest("POST", "/"+created.ID+"/reject", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}

	resp := extractApprovalObject(t, w.Body)
	if resp["status"] != string(domain.ApprovalStatusRejected) {
		t.Errorf("status = %v, want rejected", resp["status"])
	}
}

func TestHandlerReject_AlreadyResolved(t *testing.T) {
	s := newTestStore(t)
	svc := approvals.New(s)
	handler := apiapprovals.Handler(svc)

	companyID, agentID, issueID := setupTestData(t, s)

	created, err := svc.Create(context.Background(), companyID, agentID, issueID, "test_kind", nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// First reject succeeds
	_, err = svc.Reject(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("First Reject: %v", err)
	}

	// Second reject via handler should fail
	req, err := http.NewRequest("POST", "/"+created.ID+"/reject", nil)
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
	if !ok || errObj["code"] != "conflict" {
		t.Errorf("error code not conflict: %v", resp)
	}
}

func TestHandlerList_Success(t *testing.T) {
	s := newTestStore(t)
	svc := approvals.New(s)
	handler := apiapprovals.Handler(svc)

	companyID, agentID, issueID := setupTestData(t, s)

	// Create two approvals
	_, err := svc.Create(context.Background(), companyID, agentID, issueID, "kind1", nil)
	if err != nil {
		t.Fatalf("Create 1: %v", err)
	}

	_, err = svc.Create(context.Background(), companyID, agentID, issueID, "kind2", nil)
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

	list := extractApprovalList(t, w.Body)
	if len(list) != 2 {
		t.Errorf("len(list) = %d, want 2", len(list))
	}
}
