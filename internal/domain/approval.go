// Package domain holds pure data types shared across service and API packages.
// No database or HTTP dependencies are allowed here.
package domain

import "time"

// ApprovalStatus represents the status of an approval request.
type ApprovalStatus string

const (
	// ApprovalStatusPending represents a pending approval request.
	ApprovalStatusPending ApprovalStatus = "pending"
	// ApprovalStatusApproved represents an approved request.
	ApprovalStatusApproved ApprovalStatus = "approved"
	// ApprovalStatusRejected represents a rejected request.
	ApprovalStatusRejected ApprovalStatus = "rejected"
)

// Approval represents a human-in-loop approval request.
type Approval struct {
	ID            string         `json:"id"`
	CompanyID     string         `json:"companyId"`
	AgentID       string         `json:"agentId"`
	IssueID       string         `json:"issueId"`
	Kind          string         `json:"kind"`
	Status        ApprovalStatus `json:"status"`
	RequestBody   *string        `json:"requestBody"`
	ResponseBody  *string        `json:"responseBody"`
	CreatedAt     time.Time      `json:"createdAt"`
	ResolvedAt    *time.Time     `json:"resolvedAt"`
}
