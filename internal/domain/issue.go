// Package domain holds pure data types shared across service and API packages.
// No database or HTTP dependencies are allowed here.
package domain

import "time"

// Issue represents a task or issue assigned within a company.
type Issue struct {
	ID                 string     `json:"id"`
	CompanyID          string     `json:"companyId"`
	Title              string     `json:"title"`
	Body               string     `json:"body"`
	Status             string     `json:"status"`
	AssigneeID         *string    `json:"assigneeId"`
	CheckedOutBy       *string    `json:"checkedOutBy"`
	CheckedOutAt       *time.Time `json:"checkedOutAt"`
	ParentIssueID      *string    `json:"parentIssueId"`
	OriginFingerprint  string     `json:"originFingerprint"`
	CreatedAt          time.Time  `json:"createdAt"`
	UpdatedAt          time.Time  `json:"updatedAt"`
	ArchivedAt         *time.Time `json:"archivedAt"`
	Documents          []any      `json:"documents"`
	WorkProducts       []any      `json:"workProducts"`
	Priority           string     `json:"priority"`
	Estimate           *int       `json:"estimate"`
}

// validStatuses contains the allowed status values for issues.
var validStatuses = map[string]bool{
	"open":        true,
	"in_progress": true,
	"blocked":     true,
	"done":        true,
	"cancelled":   true,
}

// IsValidIssueStatus reports whether status is an allowed issue status.
func IsValidIssueStatus(status string) bool {
	return validStatuses[status]
}

var validPriorities = map[string]bool{
	"urgent": true,
	"high":   true,
	"medium": true,
	"low":    true,
}

// IsValidIssuePriority reports whether p is an allowed issue priority.
func IsValidIssuePriority(p string) bool {
	return validPriorities[p]
}
