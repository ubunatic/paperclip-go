// Package domain holds pure data types shared across service and API packages.
// No database or HTTP dependencies are allowed here.
package domain

import "time"

// InteractionStatus represents the status of an issue thread interaction.
type InteractionStatus string

const (
	// InteractionStatusPending represents a pending interaction.
	InteractionStatusPending InteractionStatus = "pending"
	// InteractionStatusResolved represents a resolved interaction.
	InteractionStatusResolved InteractionStatus = "resolved"
)

// Interaction represents an issue thread interaction for agent continuation loops.
type Interaction struct {
	ID               string             `json:"id"`
	CompanyID        string             `json:"companyId"`
	IssueID          string             `json:"issueId"`
	AgentID          *string            `json:"agentId"`
	CommentID        *string            `json:"commentId"`
	RunID            *string            `json:"runId"`
	Kind             string             `json:"kind"`
	Status           InteractionStatus  `json:"status"`
	IdempotencyKey   string             `json:"idempotencyKey"`
	Result           *string            `json:"result"`
	ResolvedAt       *time.Time         `json:"resolvedAt"`
	ResolvedByAgentID *string           `json:"resolvedByAgentId"`
	CreatedAt        time.Time          `json:"createdAt"`
	UpdatedAt        time.Time          `json:"updatedAt"`
}
