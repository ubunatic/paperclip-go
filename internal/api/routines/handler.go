// Package routines provides HTTP handlers for the /api/routines routes.
package routines

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/ubunatic/paperclip-go/internal/respond"
	"github.com/ubunatic/paperclip-go/internal/routines"
	"github.com/ubunatic/paperclip-go/internal/domain"
)

// Handler returns an http.Handler for the /api/routines sub-router.
func Handler(svc *routines.Service) http.Handler {
	r := chi.NewRouter()
	r.Get("/", list(svc))
	r.Post("/", create(svc))
	r.Get("/{id}", get(svc))
	r.Patch("/{id}", update(svc))
	r.Delete("/{id}", del(svc))
	r.Post("/{id}/trigger", trigger(svc))
	return r
}

func list(svc *routines.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		companyID := r.URL.Query().Get("companyId")
		if strings.TrimSpace(companyID) == "" {
			respond.Error(w, http.StatusUnprocessableEntity, "validation_error", "companyId query parameter is required")
			return
		}

		items, err := svc.ListByCompany(r.Context(), companyID)
		if err != nil {
			log.Printf("routines: error listing by company: %v", err)
			respond.Error(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
			return
		}

		// Ensure items is never nil (return empty array, not null)
		if items == nil {
			items = make([]*domain.Routine, 0)
		}
		respond.JSON(w, http.StatusOK, map[string]any{"items": items})
	}
}

func create(svc *routines.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			CompanyID string `json:"companyId"`
			AgentID   string `json:"agentId"`
			Name      string `json:"name"`
			CronExpr  string `json:"cronExpr"`
		}
		if !respond.DecodeJSON(w, r, &body) {
			return
		}

		if strings.TrimSpace(body.CompanyID) == "" || strings.TrimSpace(body.AgentID) == "" ||
			strings.TrimSpace(body.Name) == "" || strings.TrimSpace(body.CronExpr) == "" {
			respond.Error(w, http.StatusUnprocessableEntity, "validation_error",
				"companyId, agentId, name, and cronExpr are required and must not be blank")
			return
		}

		routine, err := svc.Create(r.Context(), body.CompanyID, body.AgentID, body.Name, body.CronExpr)
		if err != nil {
			if errors.Is(err, routines.ErrInvalidCron) {
				respond.Error(w, http.StatusUnprocessableEntity, "invalid_cron", "cron expression is invalid")
				return
			}
			if errors.Is(err, routines.ErrNameConflict) {
				respond.Error(w, http.StatusConflict, "name_conflict", "routine name already exists for this company")
				return
			}
			log.Printf("routines: error creating routine: %v", err)
			respond.Error(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
			return
		}

		respond.JSON(w, http.StatusCreated, routine)
	}
}

func get(svc *routines.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if strings.TrimSpace(id) == "" {
			respond.Error(w, http.StatusBadRequest, "validation_error", "id is required and must not be blank")
			return
		}

		routine, err := svc.GetByID(r.Context(), id)
		if err != nil {
			if errors.Is(err, routines.ErrNotFound) {
				respond.Error(w, http.StatusNotFound, "not_found", "routine not found")
				return
			}
			log.Printf("routines: error getting routine: %v", err)
			respond.Error(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
			return
		}

		respond.JSON(w, http.StatusOK, routine)
	}
}

func update(svc *routines.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if strings.TrimSpace(id) == "" {
			respond.Error(w, http.StatusBadRequest, "validation_error", "id is required and must not be blank")
			return
		}

		var body struct {
			Name     *string `json:"name"`
			CronExpr *string `json:"cronExpr"`
			Enabled  *bool   `json:"enabled"`
		}
		if !respond.DecodeJSON(w, r, &body) {
			return
		}

		patch := routines.UpdateInput{
			Name:     body.Name,
			CronExpr: body.CronExpr,
			Enabled:  body.Enabled,
		}

		routine, err := svc.Update(r.Context(), id, patch)
		if err != nil {
			if errors.Is(err, routines.ErrNotFound) {
				respond.Error(w, http.StatusNotFound, "not_found", "routine not found")
				return
			}
			if errors.Is(err, routines.ErrInvalidCron) {
				respond.Error(w, http.StatusUnprocessableEntity, "invalid_cron", "cron expression is invalid")
				return
			}
			if errors.Is(err, routines.ErrNameConflict) {
				respond.Error(w, http.StatusConflict, "name_conflict", "routine name already exists for this company")
				return
			}
			log.Printf("routines: error updating routine: %v", err)
			respond.Error(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
			return
		}

		respond.JSON(w, http.StatusOK, routine)
	}
}

func del(svc *routines.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if strings.TrimSpace(id) == "" {
			respond.Error(w, http.StatusBadRequest, "validation_error", "id is required and must not be blank")
			return
		}

		err := svc.Delete(r.Context(), id)
		if err != nil {
			if errors.Is(err, routines.ErrNotFound) {
				respond.Error(w, http.StatusNotFound, "not_found", "routine not found")
				return
			}
			log.Printf("routines: error deleting routine: %v", err)
			respond.Error(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func trigger(svc *routines.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if strings.TrimSpace(id) == "" {
			respond.Error(w, http.StatusBadRequest, "validation_error", "id is required and must not be blank")
			return
		}

		routine, err := svc.Trigger(r.Context(), id)
		if err != nil {
			if errors.Is(err, routines.ErrNotFound) {
				respond.Error(w, http.StatusNotFound, "not_found", "routine not found")
				return
			}
			log.Printf("routines: error triggering routine: %v", err)
			respond.Error(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
			return
		}

		respond.JSON(w, http.StatusOK, routine)
	}
}
