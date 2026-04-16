// Package domain holds pure data types shared across service and API packages.
// No database or HTTP dependencies are allowed here.
package domain

import "time"

// Comment represents a comment on an issue.
type Comment struct {
	ID            string    `json:"id"`
	IssueID       string    `json:"issueId"`
	AuthorAgentID *string   `json:"authorAgentId"`
	AuthorKind    string    `json:"authorKind"`
	Body          string    `json:"body"`
	CreatedAt     time.Time `json:"createdAt"`
}
