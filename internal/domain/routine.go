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
	DispatchFingerprint *string    `json:"-"`
	CreatedAt           time.Time  `json:"createdAt"`
	UpdatedAt           time.Time  `json:"updatedAt"`
}

// RoutineRun records a single dispatch of a routine.
type RoutineRun struct {
	ID         string     `json:"id"`
	RoutineID  string     `json:"routineId"`
	AgentID    string     `json:"agentId"`
	Status     string     `json:"status"`
	StartedAt  time.Time  `json:"startedAt"`
	FinishedAt *time.Time `json:"finishedAt"`
	Error      *string    `json:"error"`
}
