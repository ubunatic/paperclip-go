package domain

import "time"

// Setting represents a single instance-level configuration key-value pair.
type Setting struct {
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	UpdatedAt time.Time `json:"updatedAt"`
}
