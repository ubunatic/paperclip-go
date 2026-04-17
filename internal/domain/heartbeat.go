// Package domain holds pure data types shared across service and API packages.
// No database or HTTP dependencies are allowed here.
package domain

import "time"

// HeartbeatRun represents a single execution of a heartbeat for an agent.
type HeartbeatRun struct {
	ID         string     `json:"id"`
	AgentID    string     `json:"agentId"`
	IssueID    *string    `json:"issueId"`
	Status     string     `json:"status"`
	StartedAt  time.Time  `json:"startedAt"`
	FinishedAt *time.Time `json:"finishedAt"`
	Summary    *string    `json:"summary"`
	Error      *string    `json:"error"`
}

// RunContext holds information about the current agent and issue during a heartbeat run.
type RunContext struct {
	Agent *Agent
	Issue *Issue
}

// RunResult holds the output from a successful heartbeat run.
type RunResult struct {
	Status    string
	Summary   string
	IssueID   *string
	Error     error
}
