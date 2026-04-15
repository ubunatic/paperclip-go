// Package api provides the HTTP API router and middleware.
package api

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	apicompanies "github.com/ubunatic/paperclip-go/internal/api/companies"
	"github.com/ubunatic/paperclip-go/internal/api/health"
	"github.com/ubunatic/paperclip-go/internal/companies"
	"github.com/ubunatic/paperclip-go/internal/store"
)

// NewRouter creates and returns a chi router with all API routes and middleware.
func NewRouter(s *store.Store) *chi.Mux {
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)

	// Services
	companySvc := companies.New(s)

	// /api routes
	r.Route("/api", func(r chi.Router) {
		r.Get("/health", health.Handler)
		r.Mount("/companies", apicompanies.Handler(companySvc))
	})

	return r
}
