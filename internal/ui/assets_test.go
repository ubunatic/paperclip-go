package ui_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ubunatic/paperclip-go/internal/ui"
)

// TestHandler_NoDistDir tests that Handler returns landing page when uiDir doesn't exist
func TestHandler_NoDistDir(t *testing.T) {
	handler := ui.Handler("/nonexistent/directory")
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "paperclip-go") {
		t.Errorf("expected landing HTML to contain 'paperclip-go', got: %s", body)
	}
}

// TestHandler_DistExists_IndexHTML tests that Handler serves index.html from dist
func TestHandler_DistExists_IndexHTML(t *testing.T) {
	tempDir := t.TempDir()

	// Create index.html
	indexPath := filepath.Join(tempDir, "index.html")
	if err := os.WriteFile(indexPath, []byte("<html><body>Index</body></html>"), 0644); err != nil {
		t.Fatalf("failed to create index.html: %v", err)
	}

	handler := ui.Handler(tempDir)
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Index") {
		t.Errorf("expected 'Index' in response, got: %s", body)
	}
}

// TestHandler_DistExists_StaticFile tests that Handler serves static files from dist
func TestHandler_DistExists_StaticFile(t *testing.T) {
	tempDir := t.TempDir()

	// Create a static file
	cssPath := filepath.Join(tempDir, "style.css")
	if err := os.WriteFile(cssPath, []byte("body { color: red; }"), 0644); err != nil {
		t.Fatalf("failed to create style.css: %v", err)
	}

	handler := ui.Handler(tempDir)
	req := httptest.NewRequest("GET", "/style.css", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "color: red") {
		t.Errorf("expected CSS content in response, got: %s", body)
	}
}

// TestHandler_DistExists_SPAFallback tests SPA fallback to index.html for non-existent routes
func TestHandler_DistExists_SPAFallback(t *testing.T) {
	tempDir := t.TempDir()

	// Create index.html
	indexPath := filepath.Join(tempDir, "index.html")
	if err := os.WriteFile(indexPath, []byte("<html><body>SPA Root</body></html>"), 0644); err != nil {
		t.Fatalf("failed to create index.html: %v", err)
	}

	handler := ui.Handler(tempDir)
	req := httptest.NewRequest("GET", "/dashboard", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "SPA Root") {
		t.Errorf("expected 'SPA Root' in response, got: %s", body)
	}
}

// TestHandler_ContentType_Landing tests that content type is set correctly for landing page
func TestHandler_ContentType_Landing(t *testing.T) {
	handler := ui.Handler("/nonexistent/directory")
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	contentType := w.Header().Get("Content-Type")
	if contentType != "text/html; charset=utf-8" {
		t.Errorf("expected Content-Type 'text/html; charset=utf-8', got '%s'", contentType)
	}
}
