// Package api provides the HTTP API router and middleware.
package api

import (
	"context"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/ubunatic/paperclip-go/internal/activity"
	"github.com/ubunatic/paperclip-go/internal/agents"
	apiactivity "github.com/ubunatic/paperclip-go/internal/api/activity"
	apiagents "github.com/ubunatic/paperclip-go/internal/api/agents"
	apiapprovals "github.com/ubunatic/paperclip-go/internal/api/approvals"
	apicompanies "github.com/ubunatic/paperclip-go/internal/api/companies"
	aproutines "github.com/ubunatic/paperclip-go/internal/api/routines"
	apiworkspaces "github.com/ubunatic/paperclip-go/internal/api/workspaces"
	"github.com/ubunatic/paperclip-go/internal/api/health"
	apiheartbeat "github.com/ubunatic/paperclip-go/internal/api/heartbeat"
	apiissues "github.com/ubunatic/paperclip-go/internal/api/issues"
	apilabels "github.com/ubunatic/paperclip-go/internal/api/labels"
	apiskills "github.com/ubunatic/paperclip-go/internal/api/skills"
	apisecrets "github.com/ubunatic/paperclip-go/internal/api/secrets"
	apisettings "github.com/ubunatic/paperclip-go/internal/api/settings"
	apistubs "github.com/ubunatic/paperclip-go/internal/api/stubs"
	"github.com/ubunatic/paperclip-go/internal/approvals"
	"github.com/ubunatic/paperclip-go/internal/comments"
	"github.com/ubunatic/paperclip-go/internal/companies"
	"github.com/ubunatic/paperclip-go/internal/interactions"
	"github.com/ubunatic/paperclip-go/internal/routines"
	"github.com/ubunatic/paperclip-go/internal/workspaces"
	"github.com/ubunatic/paperclip-go/internal/domain"
	"github.com/ubunatic/paperclip-go/internal/heartbeat"
	"github.com/ubunatic/paperclip-go/internal/issues"
	"github.com/ubunatic/paperclip-go/internal/labels"
	"github.com/ubunatic/paperclip-go/internal/secrets"
	"github.com/ubunatic/paperclip-go/internal/settings"
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
	activityLog := activity.New(s)
	agentSvc := agents.New(s, activityLog)
	issueSvc := issues.New(s)
	commentSvc := comments.New(s)
	labelSvc := labels.New(s)
	secretSvc := secrets.New(s)
	settingSvc := settings.New(s)
	approvalSvc := approvals.New(s)
	interactionSvc := interactions.New(s)
	heartbeatRegistry := heartbeat.NewDefaultRegistry()
	heartbeatRunner := heartbeat.New(s, agentSvc, issueSvc, commentSvc, activityLog, heartbeatRegistry)
	routineSvc := routines.New(s)
	workspaceSvc := workspaces.New(s)

	// Seed instance settings defaults
	if err := settingSvc.SeedDefaults(context.Background(), map[string]string{
		"deployment_mode": "local_trusted",
		"allowed_origins": "localhost",
	}); err != nil {
		log.Printf("api: error seeding instance settings defaults: %v", err)
	}

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
		r.Mount("/issues", apiissues.Handler(issueSvc, commentSvc, labelSvc, activityLog, interactionSvc))
		r.Mount("/labels", apilabels.Handler(labelSvc))
		r.Mount("/heartbeat", apiheartbeat.Handler(heartbeatRunner))
		// GET /skills is read-only; use Get, not Mount
		r.Get("/skills", apiskills.Handler(skillsList))

		// Stub endpoints
		r.Mount("/approvals", apiapprovals.Handler(approvalSvc))
		r.Get("/costs", apistubs.EmptyList())
		r.Mount("/secrets", apisecrets.Handler(secretSvc))
		r.Get("/adapters", apistubs.EmptyList())
		r.Get("/company-skills", apistubs.EmptyList())
		r.Get("/dashboard", apistubs.EmptyList())
		r.Get("/goals", apistubs.EmptyList())
		r.Get("/projects", apistubs.EmptyList())
		r.Mount("/routines", aproutines.Handler(routineSvc))
		r.Mount("/execution-workspaces", apiworkspaces.Handler(workspaceSvc))
		r.Get("/plugins", apistubs.EmptyList())
		r.Get("/sidebar-badges", apistubs.EmptyList())
		r.Get("/sidebar-preferences", apistubs.EmptyList())
		r.Get("/inbox-dismissals", apistubs.EmptyList())
		r.Mount("/instance-settings", apisettings.Handler(settingSvc))
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
