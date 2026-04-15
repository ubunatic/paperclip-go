// Package ids generates unique identifiers for domain entities.
package ids

import "github.com/google/uuid"

// NewUUID returns a new random UUID string.
func NewUUID() string {
	return uuid.New().String()
}
