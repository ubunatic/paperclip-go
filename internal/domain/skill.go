// Package domain defines pure data types for the control plane.
package domain

// Skill represents a skill definition with metadata and body content.
type Skill struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Path        string `json:"path"`
	Body        string `json:"body"`
}
