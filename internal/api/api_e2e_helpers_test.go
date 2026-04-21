package api_test

import (
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/ubunatic/paperclip-go/internal/api"
	"github.com/ubunatic/paperclip-go/internal/store"
	"github.com/ubunatic/paperclip-go/internal/testutil"
)

func spawnTestServer(t *testing.T) (*httptest.Server, *store.Store) {
	t.Helper()
	s := testutil.NewStore(t)
	skillsDir := filepath.Join(t.TempDir(), "skills")
	router := api.NewRouter(s, skillsDir, "", "test")
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)
	return srv, s
}

func spawnTestServerWithSkillsDir(t *testing.T, skillsDir string) (*httptest.Server, *store.Store) {
	t.Helper()
	s := testutil.NewStore(t)
	router := api.NewRouter(s, skillsDir, "", "test")
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)
	return srv, s
}
