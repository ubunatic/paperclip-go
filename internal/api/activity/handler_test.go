package activity_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	apiactivity "github.com/ubunatic/paperclip-go/internal/api/activity"
	"github.com/ubunatic/paperclip-go/internal/activity"
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
	_, err := s.DB.ExecContext(ctx,
		`INSERT INTO companies(id, name, shortname, description, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		companyID, "Test Company", "test-"+ids.NewUUID()[:8], "Test", "2024-01-01T00:00:00Z", "2024-01-01T00:00:00Z",
	)
	if err != nil {
		t.Fatalf("create company: %v", err)
	}

	return companyID
}

func extractActivityObject(t *testing.T, body *bytes.Buffer) map[string]any {
	t.Helper()
	var resp map[string]any
	if err := json.NewDecoder(body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	return resp
}

func extractActivityList(t *testing.T, body *bytes.Buffer) []any {
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
	svc := activity.New(s)
	handler := apiactivity.Handler(svc)

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
	svc := activity.New(s)
	handler := apiactivity.Handler(svc)

	// Missing actorType
	body, _ := json.Marshal(map[string]string{
		"companyId": "test-company",
		"actorId":   "test-actor",
		"action":    "create",
		"entityType": "issue",
		"entityId":  "test-entity",
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

func TestHandlerCreate_MissingActorID(t *testing.T) {
	s := newTestStore(t)
	svc := activity.New(s)
	handler := apiactivity.Handler(svc)

	// Missing actorId
	body, _ := json.Marshal(map[string]string{
		"companyId": "test-company",
		"actorType": "agent",
		"action":    "create",
		"entityType": "issue",
		"entityId":  "test-entity",
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
	svc := activity.New(s)
	handler := apiactivity.Handler(svc)

	companyID := setupTestCompany(t, s)

	body, _ := json.Marshal(map[string]string{
		"companyId": companyID,
		"actorType": "agent",
		"actorId":   "test-actor",
		"action":    "create",
		"entityType": "issue",
		"entityId":  "test-entity",
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

	resp := extractActivityObject(t, w.Body)
	if resp["companyId"] != companyID {
		t.Errorf("companyId = %v, want %v", resp["companyId"], companyID)
	}
	if resp["action"] != "create" {
		t.Errorf("action = %v, want create", resp["action"])
	}
	if resp["entityType"] != "issue" {
		t.Errorf("entityType = %v, want issue", resp["entityType"])
	}
	if resp["entityId"] != "test-entity" {
		t.Errorf("entityId = %v, want test-entity", resp["entityId"])
	}
	if _, hasID := resp["id"]; !hasID {
		t.Error("response missing id field")
	}
}

func TestHandlerList_MissingCompanyID(t *testing.T) {
	s := newTestStore(t)
	svc := activity.New(s)
	handler := apiactivity.Handler(svc)

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestHandlerList_EmptyForCompany(t *testing.T) {
	s := newTestStore(t)
	svc := activity.New(s)
	handler := apiactivity.Handler(svc)

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

	list := extractActivityList(t, w.Body)
	if len(list) != 0 {
		t.Errorf("expected empty list, got %d items", len(list))
	}
}

func TestHandlerList_Success(t *testing.T) {
	s := newTestStore(t)
	svc := activity.New(s)
	handler := apiactivity.Handler(svc)

	companyID := setupTestCompany(t, s)

	// Create two activities
	for i := 0; i < 2; i++ {
		body, _ := json.Marshal(map[string]string{
			"companyId": companyID,
			"actorType": "agent",
			"actorId":   "test-actor",
			"action":    "create",
			"entityType": "issue",
			"entityId":  "test-entity",
		})

		req, _ := http.NewRequest("POST", "/", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}

	// List activities
	req, err := http.NewRequest("GET", "/?companyId="+companyID, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}

	list := extractActivityList(t, w.Body)
	if len(list) != 2 {
		t.Errorf("expected 2 items, got %d", len(list))
	}
}
