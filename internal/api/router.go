// Package api provides the HTTP API router and middleware.
package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/ubunatic/paperclip-go/internal/api/health"
)

// NewRouter creates and returns a chi router with all API routes and middleware.
func NewRouter() *chi.Mux {
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(contentTypeJSON)

	// /api routes
	r.Route("/api", func(r chi.Router) {
		r.Get("/health", health.Handler)
	})

	return r
}

// contentTypeJSON is middleware that sets the Content-Type header for API responses.
func contentTypeJSON(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}
