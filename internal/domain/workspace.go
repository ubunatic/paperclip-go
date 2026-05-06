// Package domain holds pure data types shared across service and API packages.
// No database or HTTP dependencies are allowed here.
package domain

import "time"

// WorkspaceStatus represents the status of an execution workspace.
type WorkspaceStatus string

const (
	// WorkspaceStatusActive represents an active workspace.
	WorkspaceStatusActive WorkspaceStatus = "active"
	// WorkspaceStatusInactive represents an inactive workspace.
	WorkspaceStatusInactive WorkspaceStatus = "inactive"
	// WorkspaceStatusError represents a workspace in error state.
	WorkspaceStatusError WorkspaceStatus = "error"
)

// validWorkspaceStatuses contains the allowed status values for workspaces.
var validWorkspaceStatuses = map[string]bool{
	"active":   true,
	"inactive": true,
	"error":    true,
}

// IsValidWorkspaceStatus reports whether status is an allowed workspace status.
func IsValidWorkspaceStatus(status string) bool {
	return validWorkspaceStatuses[status]
}

// Workspace represents an execution workspace.
type Workspace struct {
	ID        string          `json:"id"`
	CompanyID string          `json:"companyId"`
	AgentID   string          `json:"agentId"`
	IssueID   *string         `json:"issueId"`
	Path      string          `json:"path"`
	Status    WorkspaceStatus `json:"status"`
	CreatedAt time.Time       `json:"createdAt"`
	UpdatedAt time.Time       `json:"updatedAt"`
}
