// Package domain holds pure data types shared across service and API packages.
// No database or HTTP dependencies are allowed here.
package domain

import "time"

// Agent represents an agent within a company in the control plane.
type Agent struct {
	ID            string         `json:"id"`
	CompanyID     string         `json:"companyId"`
	Shortname     string         `json:"shortname"`
	DisplayName   string         `json:"displayName"`
	Role          string         `json:"role"`
	ReportsTo     *string        `json:"reportsTo"`
	Adapter       string         `json:"adapter"`
	RuntimeState  string         `json:"runtimeState"`
	Configuration map[string]any `json:"configuration"`
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
}

// validRuntimeStates contains the allowed runtime state values for agents.
var validRuntimeStates = map[string]bool{
	"idle":       true,
	"running":    true,
	"paused":     true,
	"terminated": true,
}

// IsValidRuntimeState reports whether state is an allowed agent runtime state.
func IsValidRuntimeState(state string) bool {
	return validRuntimeStates[state]
}
