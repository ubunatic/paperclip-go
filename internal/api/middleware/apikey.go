// Package middleware provides reusable HTTP middleware for the API.
package middleware

import (
	"errors"
	"net/http"

	"github.com/ubunatic/paperclip-go/internal/apikeys"
	"github.com/ubunatic/paperclip-go/internal/respond"
)

// APIKeyAuth returns a middleware that validates the X-Api-Key header.
// Requests whose URL.Path is in skipPaths are always passed through without validation,
// which allows unauthenticated access to endpoints like /api/health.
func APIKeyAuth(svc *apikeys.Service, skipPaths ...string) func(http.Handler) http.Handler {
	skip := make(map[string]struct{}, len(skipPaths))
	for _, p := range skipPaths {
		skip[p] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, ok := skip[r.URL.Path]; ok {
				next.ServeHTTP(w, r)
				return
			}

			key := r.Header.Get("X-Api-Key")
			if key == "" {
				respond.Error(w, http.StatusUnauthorized, "unauthorized", "X-Api-Key header required")
				return
			}

			_, err := svc.Validate(r.Context(), key)
			if err != nil {
				if errors.Is(err, apikeys.ErrNotFound) {
					respond.Error(w, http.StatusUnauthorized, "unauthorized", "invalid API key")
					return
				}
				if errors.Is(err, apikeys.ErrRevoked) {
					respond.Error(w, http.StatusUnauthorized, "unauthorized", "API key has been revoked")
					return
				}
				respond.Error(w, http.StatusUnauthorized, "unauthorized", "invalid API key")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
