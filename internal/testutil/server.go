// Package testutil provides helpers for integration and end-to-end tests.
package testutil

import (
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/ubunatic/paperclip-go/internal/activity"
	"github.com/ubunatic/paperclip-go/internal/api"
	"github.com/ubunatic/paperclip-go/internal/domain"
	"github.com/ubunatic/paperclip-go/internal/store"
)

// NewStore opens a temporary SQLite store (auto-migrated) and registers
// cleanup with t. Use this in unit tests that need a real database.
func NewStore(t *testing.T) *store.Store {
	t.Helper()
	dsn := filepath.Join(t.TempDir(), "test.db")
	s, err := store.Open(dsn)
	if err != nil {
		t.Fatalf("testutil.NewStore: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

// SpawnTestServer starts a full httptest.Server backed by a temp SQLite store.
// Both the server and the store are closed when t finishes.
func SpawnTestServer(t *testing.T) (*httptest.Server, *store.Store) {
	t.Helper()
	return SpawnTestServerWithSkills(t, nil)
}

// SpawnTestServerWithSkills starts a full httptest.Server with provided skills.
// Both the server and the store are closed when t finishes.
func SpawnTestServerWithSkills(t *testing.T, skills []domain.Skill) (*httptest.Server, *store.Store) {
	t.Helper()
	s := NewStore(t)
	router := api.NewRouter(s, skills)
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)
	return srv, s
}

// SpawnActivityLog returns a new activity Log using the given store.
func SpawnActivityLog(s *store.Store) *activity.Log {
	return activity.New(s)
}
