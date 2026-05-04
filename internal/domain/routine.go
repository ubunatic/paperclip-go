// Package domain holds pure data types shared across service and API packages.
// No database or HTTP dependencies are allowed here.
package domain

import "time"

type Routine struct {
	ID                  string     `json:"id"`
	CompanyID           string     `json:"companyId"`
	AgentID             string     `json:"agentId"`
	Name                string     `json:"name"`
	CronExpr            string     `json:"cronExpr"`
	Enabled             bool       `json:"enabled"`
	LastRunAt           *time.Time `json:"lastRunAt"`
	DispatchFingerprint *string    `json:"dispatchFingerprint,omitempty"`
	CreatedAt           time.Time  `json:"createdAt"`
	UpdatedAt           time.Time  `json:"updatedAt"`
}
