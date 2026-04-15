// Package domain holds pure data types shared across service and API packages.
// No database or HTTP dependencies are allowed here.
package domain

import "time"

// Company represents an agent company in the control plane.
type Company struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Shortname   string    `json:"shortname"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}
