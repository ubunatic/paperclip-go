package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	apimiddleware "github.com/ubunatic/paperclip-go/internal/api/middleware"
	"github.com/ubunatic/paperclip-go/internal/apikeys"
	"github.com/ubunatic/paperclip-go/internal/companies"
	"github.com/ubunatic/paperclip-go/internal/testutil"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func TestAPIKeyAuthMissingHeader(t *testing.T) {
	s := testutil.NewStore(t)
	svc := apikeys.New(s)
	mw := apimiddleware.APIKeyAuth(svc)

	req := httptest.NewRequest("GET", "/api/issues", nil)
	rec := httptest.NewRecorder()
	mw(okHandler()).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestAPIKeyAuthInvalidKey(t *testing.T) {
	s := testutil.NewStore(t)
	svc := apikeys.New(s)
	mw := apimiddleware.APIKeyAuth(svc)

	req := httptest.NewRequest("GET", "/api/issues", nil)
	req.Header.Set("X-Api-Key", "totally-invalid-key")
	rec := httptest.NewRecorder()
	mw(okHandler()).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestAPIKeyAuthValidKey(t *testing.T) {
	s := testutil.NewStore(t)
	ctx := context.Background()

	companySvc := companies.New(s)
	c, err := companySvc.Create(ctx, "Test Corp", "test", "")
	if err != nil {
		t.Fatalf("create company: %v", err)
	}

	svc := apikeys.New(s)
	_, rawKey, err := svc.Create(ctx, c.ID, "test-key")
	if err != nil {
		t.Fatalf("create key: %v", err)
	}

	mw := apimiddleware.APIKeyAuth(svc)
	req := httptest.NewRequest("GET", "/api/issues", nil)
	req.Header.Set("X-Api-Key", rawKey)
	rec := httptest.NewRecorder()
	mw(okHandler()).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}

func TestAPIKeyAuthRevokedKey(t *testing.T) {
	s := testutil.NewStore(t)
	ctx := context.Background()

	companySvc := companies.New(s)
	c, err := companySvc.Create(ctx, "Test Corp", "test", "")
	if err != nil {
		t.Fatalf("create company: %v", err)
	}

	svc := apikeys.New(s)
	key, rawKey, err := svc.Create(ctx, c.ID, "test-key")
	if err != nil {
		t.Fatalf("create key: %v", err)
	}
	if err := svc.Revoke(ctx, key.ID); err != nil {
		t.Fatalf("revoke key: %v", err)
	}

	mw := apimiddleware.APIKeyAuth(svc)
	req := httptest.NewRequest("GET", "/api/issues", nil)
	req.Header.Set("X-Api-Key", rawKey)
	rec := httptest.NewRecorder()
	mw(okHandler()).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestAPIKeyAuthSkipPath(t *testing.T) {
	s := testutil.NewStore(t)
	svc := apikeys.New(s)
	// /api/health is in the skip list — no key required
	mw := apimiddleware.APIKeyAuth(svc, "/api/health")

	req := httptest.NewRequest("GET", "/api/health", nil)
	rec := httptest.NewRecorder()
	mw(okHandler()).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("skip path status = %d, want 200", rec.Code)
	}
}

func TestAPIKeyAuthNonSkipPathStillRequiresKey(t *testing.T) {
	s := testutil.NewStore(t)
	svc := apikeys.New(s)
	mw := apimiddleware.APIKeyAuth(svc, "/api/health")

	// /api/issues is NOT in the skip list
	req := httptest.NewRequest("GET", "/api/issues", nil)
	rec := httptest.NewRecorder()
	mw(okHandler()).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("non-skip path without key: status = %d, want 401", rec.Code)
	}
}
