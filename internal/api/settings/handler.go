// Package settings provides HTTP handlers for the /api/instance-settings routes.
package settings

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	settingssvc "github.com/ubunatic/paperclip-go/internal/settings"
	"github.com/ubunatic/paperclip-go/internal/respond"
)

// Handler returns an http.Handler for the /api/instance-settings sub-router.
func Handler(svc *settingssvc.Service) http.Handler {
	r := chi.NewRouter()
	r.Get("/", getAll(svc))
	r.Patch("/", patch(svc))
	return r
}

func getAll(svc *settingssvc.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		settings, err := svc.GetAll(r.Context())
		if err != nil {
			log.Printf("settings: error getting all: %v", err)
			respond.Error(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
			return
		}

		// Ensure settings is never nil (return empty map, not null)
		if settings == nil {
			settings = make(map[string]string)
		}
		// Return flat JSON map, not wrapped in "items"
		respond.JSON(w, http.StatusOK, settings)
	}
}

func patch(svc *settingssvc.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		if !respond.DecodeJSON(w, r, &body) {
			return
		}

		// Validate that we got an object (non-nil after successful decode)
		if body == nil {
			respond.Error(w, http.StatusBadRequest, "bad_request", "request body must be a JSON object")
			return
		}

		updated, err := svc.Patch(r.Context(), body)
		if err != nil {
			log.Printf("settings: error patching: %v", err)
			respond.Error(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
			return
		}

		// Ensure updated is never nil (return empty map, not null)
		if updated == nil {
			updated = make(map[string]string)
		}
		// Return flat JSON map
		respond.JSON(w, http.StatusOK, updated)
	}
}
