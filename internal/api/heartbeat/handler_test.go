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
	"github.com/ubunatic/paperclip-go/internal/store"
	"github.com/ubunatic/paperclip-go/internal/testutil"
)

func newTestSetup(t *testing.T) (*heartbeat.Runner, *store.Store, *agents.Service, string) {
	t.Helper()

	s := testutil.NewStore(t)
	ctx := context.Background()

	companySvc := companies.New(s)
	company, err := companySvc.Create(ctx, "Test Corp", "test", "Test company")
	if err != nil {
		t.Fatalf("Create company: %v", err)
	}

	agentSvc := agents.New(s, activity.New(s))
	agent, err := agentSvc.Create(ctx, company.ID, "alice", "Alice", "agent", nil, "stub")
	if err != nil {
		t.Fatalf("Create agent: %v", err)
	}

	actLog := activity.New(s)
	commentSvc := comments.New(s)
	registry := heartbeat.NewDefaultRegistry()
	runner := heartbeat.New(s, agentSvc, nil, commentSvc, actLog, registry)

	return runner, s, agentSvc, agent.ID
}

func extractHeartbeatRun(t *testing.T, body *bytes.Buffer) map[string]any {
	t.Helper()

	var resp map[string]any
	if err := json.NewDecoder(body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}

	return resp
}

func TestGetByID_Found(t *testing.T) {
	runner, _, _, agentID := newTestSetup(t)
	ctx := context.Background()
	handler := apiheartbeat.Handler(runner)

	// Create a heartbeat run
	run, err := runner.Create(ctx, agentID, nil, "running")
	if err != nil {
		t.Fatalf("Create run: %v", err)
	}

	req, err := http.NewRequest("GET", "/runs/"+run.ID, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	resp := extractHeartbeatRun(t, w.Body)
	if resp["id"] != run.ID {
		t.Errorf("id = %v, want %s", resp["id"], run.ID)
	}
	if resp["agentId"] != agentID {
		t.Errorf("agentId = %v, want %s", resp["agentId"], agentID)
	}
	if resp["status"] != "running" {
		t.Errorf("status = %v, want running", resp["status"])
	}
}

func TestGetByID_NotFound(t *testing.T) {
	runner, _, _, _ := newTestSetup(t)
	handler := apiheartbeat.Handler(runner)

	req, err := http.NewRequest("GET", "/runs/nonexistent-id", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	errObj, ok := resp["error"].(map[string]any)
	if !ok || errObj["code"] != "not_found" {
		t.Errorf("error code not not_found: %v", resp)
	}
}

func TestCancel_Running(t *testing.T) {
	runner, _, _, agentID := newTestSetup(t)
	ctx := context.Background()
	handler := apiheartbeat.Handler(runner)

	// Create a heartbeat run
	run, err := runner.Create(ctx, agentID, nil, "running")
	if err != nil {
		t.Fatalf("Create run: %v", err)
	}

	req, err := http.NewRequest("POST", "/runs/"+run.ID+"/cancel", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	resp := extractHeartbeatRun(t, w.Body)
	if resp["status"] != "cancelled" {
		t.Errorf("status = %v, want cancelled", resp["status"])
	}
	if resp["finishedAt"] == nil {
		t.Error("finishedAt should not be nil after cancel")
	}
}

func TestCancel_AlreadyFinished(t *testing.T) {
	runner, _, _, agentID := newTestSetup(t)
	ctx := context.Background()
	handler := apiheartbeat.Handler(runner)

	// Create a heartbeat run with success status (already terminal)
	run, err := runner.Create(ctx, agentID, nil, "success")
	if err != nil {
		t.Fatalf("Create run: %v", err)
	}

	req, err := http.NewRequest("POST", "/runs/"+run.ID+"/cancel", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", w.Code)
	}

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	errObj, ok := resp["error"].(map[string]any)
	if !ok || errObj["code"] != "already_terminal" {
		t.Errorf("error code not already_terminal: %v", resp)
	}
}

func TestCancel_NotFound(t *testing.T) {
	runner, _, _, _ := newTestSetup(t)
	handler := apiheartbeat.Handler(runner)

	req, err := http.NewRequest("POST", "/runs/nonexistent-id/cancel", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	errObj, ok := resp["error"].(map[string]any)
	if !ok || errObj["code"] != "not_found" {
		t.Errorf("error code not not_found: %v", resp)
	}
}
