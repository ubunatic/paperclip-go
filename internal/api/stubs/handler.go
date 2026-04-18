// Package stubs provides stub HTTP handlers that return empty item lists.
package stubs

import (
	"net/http"

	"github.com/ubunatic/paperclip-go/internal/respond"
)

// EmptyList returns an http.HandlerFunc that always responds with {"items":[]}.
func EmptyList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		respond.JSON(w, http.StatusOK, map[string]any{"items": []any{}})
	}
}
