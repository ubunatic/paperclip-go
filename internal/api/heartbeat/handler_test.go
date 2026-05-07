package heartbeat_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	apiheartbeat "github.com/ubunatic/paperclip-go/internal/api/heartbeat"
	"github.com/ubunatic/paperclip-go/internal/activity"
	"github.com/ubunatic/paperclip-go/internal/agents"
	"github.com/ubunatic/paperclip-go/internal/comments"
	"github.com/ubunatic/paperclip-go/internal/companies"
	"github.com/ubunatic/paperclip-go/internal/heartbeat"
	"github.com/ubunatic/paperclip-go/internal/ids"
	"github.com/ubunatic/paperclip-go/internal/issues"
	"github.com/ubunatic/paperclip-go/internal/store"
	"github.com/ubunatic/paperclip-go/internal/testutil"
)

func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	return testutil.NewStore(t)
}

func setupTestCompanyAndAgent(t *testing.T, s *store.Store) (string, string) {
	t.Helper()
	ctx := context.Background()

	// Create company
	companySvc := companies.New(s)
	company, err := companySvc.Create(ctx, "Test Company", "test", "Test company")
	if err != nil {
		t.Fatalf("create company: %v", err)
	}

	// Create agent
	agentSvc := agents.New(s, activity.New(s))
	agent, err := agentSvc.Create(ctx, company.ID, "test-agent-"+ids.NewUUID()[:8], "Test Agent", "agent", nil, "stub")
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}

	return company.ID, agent.ID
}

func newTestRunner(t *testing.T, s *store.Store) *heartbeat.Runner {
	t.Helper()
	actLog := activity.New(s)
	commentSvc := comments.New(s)
	agentSvc := agents.New(s, actLog)
	issueSvc := issues.New(s)
	registry := heartbeat.NewDefaultRegistry()
	return heartbeat.New(s, agentSvc, issueSvc, commentSvc, actLog, registry)
}

func extractHeartbeatObject(t *testing.T, body *bytes.Buffer) map[string]any {
	t.Helper()
	var resp map[string]any
	if err := json.NewDecoder(body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	return resp
}

func extractHeartbeatList(t *testing.T, body *bytes.Buffer) []any {
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
	ctx := context.Background()

	runner := newTestRunner(t, s)

	handler := apiheartbeat.Handler(runner)

	body := bytes.NewReader([]byte("invalid json"))
	req, err := http.NewRequest("POST", "/runs", body)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestHandlerCreate_MissingAgentID(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	runner := newTestRunner(t, s)

	handler := apiheartbeat.Handler(runner)

	body, _ := json.Marshal(map[string]string{})

	req, err := http.NewRequest("POST", "/runs", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", w.Code)
	}
}

func TestHandlerCreate_AgentNotFound(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	runner := newTestRunner(t, s)

	handler := apiheartbeat.Handler(runner)

	body, _ := json.Marshal(map[string]string{
		"agentId": "nonexistent",
	})

	req, err := http.NewRequest("POST", "/runs", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestHandlerCreate_Success(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	_, agentID := setupTestCompanyAndAgent(t, s)

	runner := newTestRunner(t, s)

	handler := apiheartbeat.Handler(runner)

	body, _ := json.Marshal(map[string]string{
		"agentId": agentID,
	})

	req, err := http.NewRequest("POST", "/runs", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", w.Code)
	}

	resp := extractHeartbeatObject(t, w.Body)
	if _, hasID := resp["id"]; !hasID {
		t.Error("response missing id field")
	}
	if resp["agentId"] != agentID {
		t.Errorf("agentId = %v, want %v", resp["agentId"], agentID)
	}
	if _, hasStatus := resp["status"]; !hasStatus {
		t.Error("response missing status field")
	}
}

func TestHandlerList_MissingAgentID(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	runner := newTestRunner(t, s)

	handler := apiheartbeat.Handler(runner)

	req, err := http.NewRequest("GET", "/runs", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestHandlerList_Success(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	_, agentID := setupTestCompanyAndAgent(t, s)

	runner := newTestRunner(t, s)

	handler := apiheartbeat.Handler(runner)

	// Create 2 runs via runner.Create()
	for i := 0; i < 2; i++ {
		_, err := runner.Create(ctx, agentID, nil, "running")
		if err != nil {
			t.Fatalf("Create run: %v", err)
		}
	}

	// List runs
	req, err := http.NewRequest("GET", "/runs?agentId="+agentID, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}

	list := extractHeartbeatList(t, w.Body)
	if len(list) < 2 {
		t.Errorf("expected at least 2 items, got %d", len(list))
	}
}

func TestHandlerGet_NotFound(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	runner := newTestRunner(t, s)

	handler := apiheartbeat.Handler(runner)

	req, err := http.NewRequest("GET", "/runs/nonexistent", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestHandlerGet_Success(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	_, agentID := setupTestCompanyAndAgent(t, s)

	runner := newTestRunner(t, s)

	// Create a run
	run, err := runner.Create(ctx, agentID, nil, "running")
	if err != nil {
		t.Fatalf("Create run: %v", err)
	}

	handler := apiheartbeat.Handler(runner)

	req, err := http.NewRequest("GET", "/runs/"+run.ID, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}

	resp := extractHeartbeatObject(t, w.Body)
	if resp["id"] != run.ID {
		t.Errorf("id = %v, want %v", resp["id"], run.ID)
	}
}

func TestHandlerCancel_NotFound(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	runner := newTestRunner(t, s)

	handler := apiheartbeat.Handler(runner)

	body, _ := json.Marshal(map[string]string{})

	req, err := http.NewRequest("POST", "/runs/nonexistent/cancel", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestHandlerCancel_TerminalStatus(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	_, agentID := setupTestCompanyAndAgent(t, s)

	runner := newTestRunner(t, s)

	// Create a run and update it to success (terminal status)
	run, err := runner.Create(ctx, agentID, nil, "running")
	if err != nil {
		t.Fatalf("Create run: %v", err)
	}

	// Update run to terminal status
	successMsg := "test output"
	_, err = runner.Update(ctx, run.ID, "success", &successMsg, nil)
	if err != nil {
		t.Fatalf("Update run: %v", err)
	}

	handler := apiheartbeat.Handler(runner)

	body, _ := json.Marshal(map[string]string{})

	req, err := http.NewRequest("POST", "/runs/"+run.ID+"/cancel", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want 409", w.Code)
	}
}

func TestHandlerCancel_Success(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	_, agentID := setupTestCompanyAndAgent(t, s)

	runner := newTestRunner(t, s)

	// Create a running run
	run, err := runner.Create(ctx, agentID, nil, "running")
	if err != nil {
		t.Fatalf("Create run: %v", err)
	}

	handler := apiheartbeat.Handler(runner)

	body, _ := json.Marshal(map[string]string{})

	req, err := http.NewRequest("POST", "/runs/"+run.ID+"/cancel", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}

	resp := extractHeartbeatObject(t, w.Body)
	if resp["status"] != "cancelled" {
		t.Errorf("status = %v, want cancelled", resp["status"])
	}
}
