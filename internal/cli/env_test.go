package cli

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ubunatic/paperclip-go/internal/companies"
	"github.com/ubunatic/paperclip-go/internal/secrets"
	"github.com/ubunatic/paperclip-go/internal/testutil"
)

func TestEnvListViaDB(t *testing.T) {
	s := testutil.NewStore(t)
	defer s.Close()
	ctx := context.Background()

	// Create a company first (foreign key constraint)
	companySvc := companies.New(s)
	company, err := companySvc.Create(ctx, "Test Company", "test-co", "")
	if err != nil {
		t.Fatalf("creating company: %v", err)
	}
	companyID := company.ID

	// Create some secrets
	svc := secrets.New(s)
	_, err = svc.Create(ctx, companyID, "FOO", "bar")
	if err != nil {
		t.Fatalf("creating secret FOO: %v", err)
	}

	_, err = svc.Create(ctx, companyID, "BAZ", "qux")
	if err != nil {
		t.Fatalf("creating secret BAZ: %v", err)
	}

	// Test that listViaDB returns the secrets properly
	items, err := svc.ListByCompany(ctx, companyID)
	if err != nil {
		t.Fatalf("listing secrets: %v", err)
	}

	if len(items) != 2 {
		t.Errorf("expected 2 secrets, got %d", len(items))
	}

	found := make(map[string]bool)
	for _, item := range items {
		found[item.Name] = true
	}

	if !found["FOO"] {
		t.Errorf("missing FOO secret")
	}
	if !found["BAZ"] {
		t.Errorf("missing BAZ secret")
	}
}

func TestEnvSetViaDB(t *testing.T) {
	s := testutil.NewStore(t)
	defer s.Close()
	ctx := context.Background()

	// Create a company first
	companySvc := companies.New(s)
	company, err := companySvc.Create(ctx, "Test Company", "test-co", "")
	if err != nil {
		t.Fatalf("creating company: %v", err)
	}
	companyID := company.ID

	// Test that setViaDB creates a secret
	svc := secrets.New(s)
	secret, err := svc.Create(ctx, companyID, "TEST_KEY", "test_value")
	if err != nil {
		t.Fatalf("setViaDB failed: %v", err)
	}

	if secret.Name != "TEST_KEY" {
		t.Errorf("expected name 'TEST_KEY', got '%s'", secret.Name)
	}

	// Verify it was actually created
	items, err := svc.ListByCompany(ctx, companyID)
	if err != nil {
		t.Fatalf("listing secrets: %v", err)
	}

	found := false
	for _, item := range items {
		if item.Name == "TEST_KEY" {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("secret TEST_KEY not found in database")
	}
}

func TestEnvGetViaDB(t *testing.T) {
	s := testutil.NewStore(t)
	defer s.Close()
	ctx := context.Background()

	// Create a company first
	companySvc := companies.New(s)
	company, err := companySvc.Create(ctx, "Test Company", "test-co", "")
	if err != nil {
		t.Fatalf("creating company: %v", err)
	}
	companyID := company.ID

	// Create a secret
	svc := secrets.New(s)
	secret, err := svc.Create(ctx, companyID, "MY_VAR", "my_secret_value")
	if err != nil {
		t.Fatalf("creating secret: %v", err)
	}

	// Test that GetByID retrieves the secret correctly
	retrieved, err := svc.GetByID(ctx, secret.ID)
	if err != nil {
		t.Fatalf("getViaDB failed: %v", err)
	}

	if retrieved.Value != "my_secret_value" {
		t.Errorf("expected 'my_secret_value', got: %q", retrieved.Value)
	}

	if retrieved.Name != "MY_VAR" {
		t.Errorf("expected name 'MY_VAR', got: %q", retrieved.Name)
	}
}

func TestEnvListViaHTTP(t *testing.T) {
	ctx := context.Background()
	companyID := "test-company"

	// Create a mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/secrets" && r.Method == "GET" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"items": [
					{"id":"1","companyId":"test-company","name":"FOO","createdAt":"2024-01-01T00:00:00Z","updatedAt":"2024-01-01T00:00:00Z"},
					{"id":"2","companyId":"test-company","name":"BAR","createdAt":"2024-01-02T00:00:00Z","updatedAt":"2024-01-02T00:00:00Z"}
				]
			}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	// Create HTTP client pointing to our mock server
	client := &HTTPClient{
		client:  server.Client(),
		baseURL: server.URL,
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := listViaHTTP(ctx, client, companyID)

	w.Close()
	os.Stdout = oldStdout
	output, _ := io.ReadAll(r)

	if err != nil {
		t.Fatalf("listViaHTTP failed: %v", err)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "FOO") {
		t.Errorf("output missing FOO secret, got: %s", outputStr)
	}
	if !strings.Contains(outputStr, "BAR") {
		t.Errorf("output missing BAR secret, got: %s", outputStr)
	}
}

func TestEnvSetViaHTTP(t *testing.T) {
	ctx := context.Background()
	companyID := "test-company"

	// Create a mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/secrets" && r.Method == "POST" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			now := time.Now().UTC()
			w.Write([]byte(`{
				"id":"new-secret-id",
				"companyId":"test-company",
				"name":"NEW_KEY",
				"value":"new_value",
				"createdAt":"` + now.Format(time.RFC3339) + `",
				"updatedAt":"` + now.Format(time.RFC3339) + `"
			}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	client := &HTTPClient{
		client:  server.Client(),
		baseURL: server.URL,
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := setViaHTTP(ctx, client, companyID, "NEW_KEY", "new_value")

	w.Close()
	os.Stdout = oldStdout
	output, _ := io.ReadAll(r)

	if err != nil {
		t.Fatalf("setViaHTTP failed: %v", err)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "NEW_KEY") {
		t.Errorf("output missing NEW_KEY, got: %s", outputStr)
	}
}

func TestEnvGetViaHTTP(t *testing.T) {
	ctx := context.Background()
	companyID := "test-company"

	// Create a mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/secrets" && r.Method == "GET" && r.URL.RawQuery != "" {
			// List endpoint
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"items": [
					{"id":"secret-id","companyId":"test-company","name":"MY_VAR","createdAt":"2024-01-01T00:00:00Z","updatedAt":"2024-01-01T00:00:00Z"}
				]
			}`))
			return
		}
		if r.URL.Path == "/api/secrets/secret-id" && r.Method == "GET" {
			// Get endpoint
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			now := time.Now().UTC()
			w.Write([]byte(`{
				"id":"secret-id",
				"companyId":"test-company",
				"name":"MY_VAR",
				"value":"my_secret_value",
				"createdAt":"` + now.Format(time.RFC3339) + `",
				"updatedAt":"` + now.Format(time.RFC3339) + `"
			}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	client := &HTTPClient{
		client:  server.Client(),
		baseURL: server.URL,
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := getViaHTTP(ctx, client, companyID, "MY_VAR")

	w.Close()
	os.Stdout = oldStdout
	output, _ := io.ReadAll(r)

	if err != nil {
		t.Fatalf("getViaHTTP failed: %v", err)
	}

	outputStr := strings.TrimSpace(string(output))
	if outputStr != "my_secret_value" {
		t.Errorf("expected 'my_secret_value', got: %q", outputStr)
	}
}

func TestEnvGetViaHTTPNotFound(t *testing.T) {
	ctx := context.Background()
	companyID := "test-company"

	// Create a mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/secrets" && r.Method == "GET" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"items": []}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	client := &HTTPClient{
		client:  server.Client(),
		baseURL: server.URL,
	}

	err := getViaHTTP(ctx, client, companyID, "NONEXISTENT")

	if err == nil {
		t.Errorf("expected error for nonexistent secret, got nil")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestHTTPClientNewHTTPClient(t *testing.T) {
	// Save old env
	oldURL := os.Getenv("PAPERCLIP_API_URL")
	defer func() {
		if oldURL != "" {
			os.Setenv("PAPERCLIP_API_URL", oldURL)
		} else {
			os.Unsetenv("PAPERCLIP_API_URL")
		}
	}()

	// Test default case (no env var)
	os.Unsetenv("PAPERCLIP_API_URL")
	client, err := NewHTTPClient()
	if err != nil {
		t.Fatalf("creating default HTTP client: %v", err)
	}
	if !strings.Contains(client.BaseURL(), "127.0.0.1:3200") {
		t.Errorf("expected default URL to contain 127.0.0.1:3200, got: %s", client.BaseURL())
	}
	client.Close()

	// Test with custom URL
	os.Setenv("PAPERCLIP_API_URL", "http://example.com:8080")
	client2, err := NewHTTPClient()
	if err != nil {
		t.Fatalf("creating custom HTTP client: %v", err)
	}
	if client2.BaseURL() != "http://example.com:8080" {
		t.Errorf("expected URL 'http://example.com:8080', got: %s", client2.BaseURL())
	}
	client2.Close()
}

func TestHTTPClientDo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	}))
	defer server.Close()

	client := &HTTPClient{
		client:  server.Client(),
		baseURL: server.URL,
	}

	req, err := http.NewRequest("GET", server.URL+"/test", nil)
	if err != nil {
		t.Fatalf("creating request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("doing request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "test response" {
		t.Errorf("expected 'test response', got: %s", string(body))
	}
}

func TestEnvSetViaDBDuplicate(t *testing.T) {
	s := testutil.NewStore(t)
	svc := secrets.New(s)
	ctx := context.Background()

	// Create a company first
	companySvc := companies.New(s)
	company, err := companySvc.Create(ctx, "Test Company", "test-co", "")
	if err != nil {
		t.Fatalf("creating company: %v", err)
	}
	companyID := company.ID

	// Create first secret
	_, err = svc.Create(ctx, companyID, "DUP_KEY", "value1")
	if err != nil {
		t.Fatalf("creating first secret: %v", err)
	}

	// Try to create duplicate via setViaDB
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err = setViaDB(ctx, companyID, "DUP_KEY", "value2")

	w.Close()
	os.Stdout = oldStdout
	io.ReadAll(r)

	// Should get an error about duplicate
	if err == nil {
		t.Errorf("expected error creating duplicate secret, got nil")
	}
}
