// Package domain holds pure data types shared across service and API packages.
// No database or HTTP dependencies are allowed here.
package domain

import "time"

// Issue represents a task or issue assigned within a company.
type Issue struct {
	ID             string     `json:"id"`
	CompanyID      string     `json:"companyId"`
	Title          string     `json:"title"`
	Body           string     `json:"body"`
	Status         string     `json:"status"`
	AssigneeID     *string    `json:"assigneeId"`
	CheckedOutBy   *string    `json:"checkedOutBy"`
	CheckedOutAt   *time.Time `json:"checkedOutAt"`
	ParentIssueID  *string    `json:"parentIssueId"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}
