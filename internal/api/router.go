// Package api provides the HTTP API router and middleware.
package api

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	apiactivity "github.com/ubunatic/paperclip-go/internal/api/activity"
	apiagents "github.com/ubunatic/paperclip-go/internal/api/agents"
	apicompanies "github.com/ubunatic/paperclip-go/internal/api/companies"
	apiissues "github.com/ubunatic/paperclip-go/internal/api/issues"
	apiskills "github.com/ubunatic/paperclip-go/internal/api/skills"
	"github.com/ubunatic/paperclip-go/internal/api/health"
	"github.com/ubunatic/paperclip-go/internal/activity"
	"github.com/ubunatic/paperclip-go/internal/agents"
	"github.com/ubunatic/paperclip-go/internal/comments"
	"github.com/ubunatic/paperclip-go/internal/companies"
	"github.com/ubunatic/paperclip-go/internal/domain"
	"github.com/ubunatic/paperclip-go/internal/issues"
	"github.com/ubunatic/paperclip-go/internal/store"
)

// NewRouter creates and returns a chi router with all API routes and middleware.
func NewRouter(s *store.Store, skills []domain.Skill) *chi.Mux {
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)

	// Services
	companySvc := companies.New(s)
	agentSvc := agents.New(s)
	activityLog := activity.New(s)
	issueSvc := issues.New(s)
	commentSvc := comments.New(s)

	// /api routes
	r.Route("/api", func(r chi.Router) {
		r.Get("/health", health.Handler)
		r.Mount("/companies", apicompanies.Handler(companySvc))
		r.Mount("/agents", apiagents.Handler(agentSvc))
		r.Mount("/activity", apiactivity.Handler(activityLog))
		r.Mount("/issues", apiissues.Handler(issueSvc, commentSvc))
		r.Mount("/skills", apiskills.Handler(skills))
	})

	return r
}
