// Package api provides the HTTP API router and middleware.
package api

import (
	"log"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/ubunatic/paperclip-go/internal/activity"
	"github.com/ubunatic/paperclip-go/internal/agents"
	apiactivity "github.com/ubunatic/paperclip-go/internal/api/activity"
	apiagents "github.com/ubunatic/paperclip-go/internal/api/agents"
	apicompanies "github.com/ubunatic/paperclip-go/internal/api/companies"
	"github.com/ubunatic/paperclip-go/internal/api/health"
	apiissues "github.com/ubunatic/paperclip-go/internal/api/issues"
	apiskills "github.com/ubunatic/paperclip-go/internal/api/skills"
	"github.com/ubunatic/paperclip-go/internal/comments"
	"github.com/ubunatic/paperclip-go/internal/companies"
	"github.com/ubunatic/paperclip-go/internal/issues"
	"github.com/ubunatic/paperclip-go/internal/skills"
	"github.com/ubunatic/paperclip-go/internal/store"
)

// NewRouter creates and returns a chi router with all API routes and middleware.
func NewRouter(s *store.Store, skillsDir string) *chi.Mux {
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

	// Load skills
	skillsList, err := skills.Load(skillsDir)
	if err != nil {
		log.Printf("skills: failed to load from %s: %v", skillsDir, err)
	}

	// /api routes
	r.Route("/api", func(r chi.Router) {
		r.Get("/health", health.Handler)
		r.Mount("/companies", apicompanies.Handler(companySvc))
		r.Mount("/agents", apiagents.Handler(agentSvc))
		r.Mount("/activity", apiactivity.Handler(activityLog))
		r.Mount("/issues", apiissues.Handler(issueSvc, commentSvc))
		r.Get("/skills", apiskills.Handler(skillsList))
	})

	return r
}
