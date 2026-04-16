// Package domain holds pure data types shared across service and API packages.
package domain

// Skill represents a skill available in the control plane.
type Skill struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Path        string `json:"path"`
	Body        string `json:"body"`
}
