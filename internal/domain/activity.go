// Package domain holds pure data types shared across service and API packages.
// No database or HTTP dependencies are allowed here.
package domain

import (
	"encoding/json"
	"time"
)

// Activity represents a log entry of an action taken by an actor on an entity.
type Activity struct {
	ID         string          `json:"id"`
	CompanyID  string          `json:"companyId"`
	ActorKind  string          `json:"actorKind"`
	ActorID    string          `json:"actorId"`
	Action     string          `json:"action"`
	EntityKind string          `json:"entityKind"`
	EntityID   string          `json:"entityId"`
	MetaJSON   json.RawMessage `json:"metaJson"`
	CreatedAt  time.Time       `json:"createdAt"`
}
