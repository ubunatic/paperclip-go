package secrets_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	apisecrets "github.com/ubunatic/paperclip-go/internal/api/secrets"
	"github.com/ubunatic/paperclip-go/internal/ids"
	"github.com/ubunatic/paperclip-go/internal/secrets"
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

func extractSecretObject(t *testing.T, body *bytes.Buffer) map[string]any {
	t.Helper()
	var resp map[string]any
	if err := json.NewDecoder(body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	return resp
}

func extractSecretList(t *testing.T, body *bytes.Buffer) []any {
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
	svc := secrets.New(s)
	handler := apisecrets.Handler(svc)

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
	svc := secrets.New(s)
	handler := apisecrets.Handler(svc)

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

	list := extractSecretList(t, w.Body)
	if len(list) != 0 {
		t.Errorf("expected empty list, got %d items", len(list))
	}
}

func TestHandlerList_Success(t *testing.T) {
	s := newTestStore(t)
	svc := secrets.New(s)
	handler := apisecrets.Handler(svc)

	companyID := setupTestCompany(t, s)

	// Create two secrets
	for i := 0; i < 2; i++ {
		body, _ := json.Marshal(map[string]string{
			"companyId": companyID,
			"name":      "secret-" + ids.NewUUID()[:8],
			"value":     "secret-value",
		})

		req, _ := http.NewRequest("POST", "/", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}

	// List secrets
	req, err := http.NewRequest("GET", "/?companyId="+companyID, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}

	list := extractSecretList(t, w.Body)
	if len(list) < 2 {
		t.Errorf("expected at least 2 items, got %d", len(list))
	}

	// Verify items are SecretSummary (no value field)
	if len(list) > 0 {
		item := list[0].(map[string]any)
		if _, hasValue := item["value"]; hasValue {
			t.Error("list should not contain value field (SecretSummary)")
		}
		if _, hasName := item["name"]; !hasName {
			t.Error("list should contain name field")
		}
	}
}

func TestHandlerCreate_InvalidJSON(t *testing.T) {
	s := newTestStore(t)
	svc := secrets.New(s)
	handler := apisecrets.Handler(svc)

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
	svc := secrets.New(s)
	handler := apisecrets.Handler(svc)

	companyID := setupTestCompany(t, s)

	// Missing name
	body, _ := json.Marshal(map[string]string{
		"companyId": companyID,
		"value":     "secret-value",
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

func TestHandlerCreate_WhitespaceValue(t *testing.T) {
	s := newTestStore(t)
	svc := secrets.New(s)
	handler := apisecrets.Handler(svc)

	companyID := setupTestCompany(t, s)

	body, _ := json.Marshal(map[string]string{
		"companyId": companyID,
		"name":      "secret-name",
		"value":     "   ",
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
	svc := secrets.New(s)
	handler := apisecrets.Handler(svc)

	companyID := setupTestCompany(t, s)

	body, _ := json.Marshal(map[string]string{
		"companyId": companyID,
		"name":      "test-secret",
		"value":     "secret-value",
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

	resp := extractSecretObject(t, w.Body)
	if resp["name"] != "test-secret" {
		t.Errorf("name = %v, want test-secret", resp["name"])
	}
	if resp["value"] != "secret-value" {
		t.Errorf("value = %v, want secret-value", resp["value"])
	}
	if _, hasID := resp["id"]; !hasID {
		t.Error("response missing id field")
	}
}

func TestHandlerCreate_Duplicate(t *testing.T) {
	s := newTestStore(t)
	svc := secrets.New(s)
	handler := apisecrets.Handler(svc)

	companyID := setupTestCompany(t, s)

	// Create first secret
	body, _ := json.Marshal(map[string]string{
		"companyId": companyID,
		"name":      "duplicate-secret",
		"value":     "secret-value-1",
	})

	req, _ := http.NewRequest("POST", "/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Try to create duplicate
	body, _ = json.Marshal(map[string]string{
		"companyId": companyID,
		"name":      "duplicate-secret",
		"value":     "secret-value-2",
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
	svc := secrets.New(s)
	handler := apisecrets.Handler(svc)

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
	svc := secrets.New(s)
	handler := apisecrets.Handler(svc)

	companyID := setupTestCompany(t, s)

	// Create a secret
	body, _ := json.Marshal(map[string]string{
		"companyId": companyID,
		"name":      "test-secret",
		"value":     "secret-value",
	})

	req, _ := http.NewRequest("POST", "/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	created := extractSecretObject(t, w.Body)
	secretID := created["id"].(string)

	// Get the secret
	req, err := http.NewRequest("GET", "/"+secretID, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}

	resp := extractSecretObject(t, w.Body)
	if resp["value"] != "secret-value" {
		t.Errorf("value = %v, want secret-value", resp["value"])
	}
}

func TestHandlerUpdate_MissingFields(t *testing.T) {
	s := newTestStore(t)
	svc := secrets.New(s)
	handler := apisecrets.Handler(svc)

	companyID := setupTestCompany(t, s)

	// Create a secret first
	body, _ := json.Marshal(map[string]string{
		"companyId": companyID,
		"name":      "test-secret",
		"value":     "secret-value",
	})

	req, _ := http.NewRequest("POST", "/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	created := extractSecretObject(t, w.Body)
	secretID := created["id"].(string)

	// Update with empty body
	body, _ = json.Marshal(map[string]any{})

	req, err := http.NewRequest("PATCH", "/"+secretID, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", w.Code)
	}
}

func TestHandlerUpdate_EmptyName(t *testing.T) {
	s := newTestStore(t)
	svc := secrets.New(s)
	handler := apisecrets.Handler(svc)

	companyID := setupTestCompany(t, s)

	// Create a secret first
	body, _ := json.Marshal(map[string]string{
		"companyId": companyID,
		"name":      "test-secret",
		"value":     "secret-value",
	})

	req, _ := http.NewRequest("POST", "/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	created := extractSecretObject(t, w.Body)
	secretID := created["id"].(string)

	// Update with empty name
	emptyName := ""
	body, _ = json.Marshal(map[string]any{
		"name": emptyName,
	})

	req, err := http.NewRequest("PATCH", "/"+secretID, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", w.Code)
	}
}

func TestHandlerUpdate_NotFound(t *testing.T) {
	s := newTestStore(t)
	svc := secrets.New(s)
	handler := apisecrets.Handler(svc)

	body, _ := json.Marshal(map[string]string{
		"name": "updated-name",
	})

	req, err := http.NewRequest("PATCH", "/nonexistent", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestHandlerUpdate_Success(t *testing.T) {
	s := newTestStore(t)
	svc := secrets.New(s)
	handler := apisecrets.Handler(svc)

	companyID := setupTestCompany(t, s)

	// Create a secret first
	body, _ := json.Marshal(map[string]string{
		"companyId": companyID,
		"name":      "test-secret",
		"value":     "secret-value",
	})

	req, _ := http.NewRequest("POST", "/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	created := extractSecretObject(t, w.Body)
	secretID := created["id"].(string)

	// Update the name
	body, _ = json.Marshal(map[string]string{
		"name": "updated-name",
	})

	req, err := http.NewRequest("PATCH", "/"+secretID, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}

	resp := extractSecretObject(t, w.Body)
	if resp["name"] != "updated-name" {
		t.Errorf("name = %v, want updated-name", resp["name"])
	}
}

func TestHandlerUpdate_DuplicateName(t *testing.T) {
	s := newTestStore(t)
	svc := secrets.New(s)
	handler := apisecrets.Handler(svc)

	companyID := setupTestCompany(t, s)

	// Create first secret
	body, _ := json.Marshal(map[string]string{
		"companyId": companyID,
		"name":      "secret-one",
		"value":     "value-one",
	})

	req, _ := http.NewRequest("POST", "/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Create second secret
	body, _ = json.Marshal(map[string]string{
		"companyId": companyID,
		"name":      "secret-two",
		"value":     "value-two",
	})

	req, _ = http.NewRequest("POST", "/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	created := extractSecretObject(t, w.Body)
	secretTwoID := created["id"].(string)

	// Try to update second secret to have first's name
	body, _ = json.Marshal(map[string]string{
		"name": "secret-one",
	})

	req, err := http.NewRequest("PATCH", "/"+secretTwoID, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want 409", w.Code)
	}
}

func TestHandlerDelete_NotFound(t *testing.T) {
	s := newTestStore(t)
	svc := secrets.New(s)
	handler := apisecrets.Handler(svc)

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
	svc := secrets.New(s)
	handler := apisecrets.Handler(svc)

	companyID := setupTestCompany(t, s)

	// Create a secret
	body, _ := json.Marshal(map[string]string{
		"companyId": companyID,
		"name":      "test-secret",
		"value":     "secret-value",
	})

	req, _ := http.NewRequest("POST", "/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	created := extractSecretObject(t, w.Body)
	secretID := created["id"].(string)

	// Delete the secret
	req, err := http.NewRequest("DELETE", "/"+secretID, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 204", w.Code)
	}
}
