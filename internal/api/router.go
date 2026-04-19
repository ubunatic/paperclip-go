// Package api provides the HTTP API router and middleware.
package api

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/ubunatic/paperclip-go/internal/activity"
	"github.com/ubunatic/paperclip-go/internal/agents"
	apiactivity "github.com/ubunatic/paperclip-go/internal/api/activity"
	apiagents "github.com/ubunatic/paperclip-go/internal/api/agents"
	apicompanies "github.com/ubunatic/paperclip-go/internal/api/companies"
	"github.com/ubunatic/paperclip-go/internal/api/health"
	apiheartbeat "github.com/ubunatic/paperclip-go/internal/api/heartbeat"
	apiissues "github.com/ubunatic/paperclip-go/internal/api/issues"
	apiskills "github.com/ubunatic/paperclip-go/internal/api/skills"
	apistubs "github.com/ubunatic/paperclip-go/internal/api/stubs"
	"github.com/ubunatic/paperclip-go/internal/comments"
	"github.com/ubunatic/paperclip-go/internal/companies"
	"github.com/ubunatic/paperclip-go/internal/domain"
	"github.com/ubunatic/paperclip-go/internal/heartbeat"
	"github.com/ubunatic/paperclip-go/internal/issues"
	"github.com/ubunatic/paperclip-go/internal/skills"
	"github.com/ubunatic/paperclip-go/internal/store"
	"github.com/ubunatic/paperclip-go/internal/ui"
)

// NewRouter creates and returns a chi router with all API routes and middleware.
func NewRouter(s *store.Store, skillsDir string, uiDir string, version string) *chi.Mux {
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
	heartbeatRegistry := heartbeat.NewDefaultRegistry()
	heartbeatRunner := heartbeat.New(s, agentSvc, issueSvc, commentSvc, activityLog, heartbeatRegistry)

	// Load skills
	skillsList, err := skills.Load(skillsDir)
	if err != nil {
		log.Printf("skills: failed to load from %s: %v", skillsDir, err)
		skillsList = []domain.Skill{}
	}

	// /api routes
	r.Route("/api", func(r chi.Router) {
		r.Get("/health", health.Handler(version))
		r.Mount("/companies", apicompanies.Handler(companySvc))
		r.Mount("/agents", apiagents.Handler(agentSvc))
		r.Mount("/activity", apiactivity.Handler(activityLog))
		r.Mount("/issues", apiissues.Handler(issueSvc, commentSvc))
		r.Mount("/heartbeat", apiheartbeat.Handler(heartbeatRunner))
		// GET /skills is read-only; use Get, not Mount
		r.Get("/skills", apiskills.Handler(skillsList))

		// Stub endpoints
		r.Get("/approvals", apistubs.EmptyList())
		r.Get("/costs", apistubs.EmptyList())
		r.Get("/secrets", apistubs.EmptyList())
		r.Get("/adapters", apistubs.EmptyList())
		r.Get("/company-skills", apistubs.EmptyList())
		r.Get("/dashboard", apistubs.EmptyList())
		r.Get("/goals", apistubs.EmptyList())
		r.Get("/projects", apistubs.EmptyList())
		r.Get("/routines", apistubs.EmptyList())
		r.Get("/plugins", apistubs.EmptyList())
		r.Get("/sidebar-badges", apistubs.EmptyList())
		r.Get("/sidebar-preferences", apistubs.EmptyList())
		r.Get("/inbox-dismissals", apistubs.EmptyList())
		r.Get("/instance-settings", apistubs.EmptyList())
		r.Get("/llms", apistubs.EmptyList())
		r.Get("/access", apistubs.EmptyList())

		// Return JSON 404 for undefined /api/* paths (not HTML)
		r.NotFound(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error":{"code":"NOT_FOUND","message":"endpoint not found"}}`))
		})
	})

	// UI handler (serves non-API routes and SPA fallback)
	r.NotFound(ui.Handler(uiDir).ServeHTTP)

	return r
}
