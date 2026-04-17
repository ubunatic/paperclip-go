// Package domain holds pure data types shared across service and API packages.
// No database or HTTP dependencies are allowed here.
package domain

import "time"

// Agent represents an agent within a company in the control plane.
type Agent struct {
	ID          string    `json:"id"`
	CompanyID   string    `json:"companyId"`
	Shortname   string    `json:"shortname"`
	DisplayName string    `json:"displayName"`
	Role        string    `json:"role"`
	ReportsTo   *string   `json:"reportsTo"`
	Adapter     string    `json:"adapter"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}
