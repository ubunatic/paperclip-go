// Package domain holds pure data types shared across service and API packages.
package domain

import "time"

// Label represents a label that can be applied to issues within a company.
type Label struct {
	ID        string    `json:"id"`
	CompanyID string    `json:"companyId"`
	Name      string    `json:"name"`
	Color     string    `json:"color"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
