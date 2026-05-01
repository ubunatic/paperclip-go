// Package domain holds pure data types shared across service and API packages.
// No database or HTTP dependencies are allowed here.
package domain

import "time"

// Secret represents a stored secret value for a company.
type Secret struct {
	ID        string    `json:"id"`
	CompanyID string    `json:"companyId"`
	Name      string    `json:"name"`
	Value     string    `json:"value"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// SecretSummary represents a secret without its value (for list responses).
type SecretSummary struct {
	ID        string    `json:"id"`
	CompanyID string    `json:"companyId"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
