package domain

import "time"

type APIKey struct {
	ID        string     `json:"id"`
	CompanyID string     `json:"companyId"`
	Name      string     `json:"name"`
	CreatedAt time.Time  `json:"createdAt"`
	RevokedAt *time.Time `json:"revokedAt"`
}
