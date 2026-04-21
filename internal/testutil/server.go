// Package testutil provides helpers for integration and end-to-end tests.
package testutil

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/ubunatic/paperclip-go/internal/activity"
	"github.com/ubunatic/paperclip-go/internal/companies"
	"github.com/ubunatic/paperclip-go/internal/issues"
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

// SpawnActivityLog returns a new activity Log using the given store.
func SpawnActivityLog(s *store.Store) *activity.Log {
	return activity.New(s)
}

// CreateTestCompany creates a test company and returns its ID.
func CreateTestCompany(t *testing.T, s *store.Store) string {
	t.Helper()
	companySvc := companies.New(s)
	company, err := companySvc.Create(context.Background(), "Test Corp", "test-corp", "Test company")
	if err != nil {
		t.Fatalf("CreateTestCompany: %v", err)
	}
	return company.ID
}

// CreateTestIssue creates a test issue in a company and returns its ID.
func CreateTestIssue(t *testing.T, s *store.Store, companyID string) string {
	t.Helper()
	issueSvc := issues.New(s)
	issue, err := issueSvc.Create(context.Background(), companyID, "Test Issue", "Test body", nil)
	if err != nil {
		t.Fatalf("CreateTestIssue: %v", err)
	}
	return issue.ID
}
