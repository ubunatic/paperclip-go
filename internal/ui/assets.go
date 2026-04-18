package ui

import (
	_ "embed"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

//go:embed landing.html
var landingHTML []byte

// Handler returns an http.Handler that serves the UI.
// If uiDir exists, it serves files from that directory with SPA fallback.
// If uiDir does not exist, it serves the embedded landing.html for all requests.
func Handler(uiDir string) http.Handler {
	_, err := os.Stat(uiDir)
	distExists := err == nil
	var distFS fs.FS
	if distExists {
		distFS = os.DirFS(uiDir)
	}
	var fileServer http.Handler
	if distExists {
		fileServer = http.FileServerFS(distFS)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !distExists {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			w.Write(landingHTML)
			return
		}

		p := strings.TrimPrefix(r.URL.Path, "/")
		if p == "" {
			p = "index.html"
		}

		f, err := distFS.Open(p)
		if err != nil {
			// File not found or dist missing → SPA fallback
			http.ServeFile(w, r, filepath.Join(uiDir, "index.html"))
			return
		}

		// Check if it's a directory
		fi, err := f.Stat()
		f.Close()
		if err != nil || fi.IsDir() {
			// Directory access or stat error → SPA fallback
			http.ServeFile(w, r, filepath.Join(uiDir, "index.html"))
			return
		}

		// Regular file → serve it
		fileServer.ServeHTTP(w, r)
	})
}
