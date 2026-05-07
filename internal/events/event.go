// Package events provides an event bus for publishing and subscribing to domain events.
package events

import "time"

// Event represents a domain event that occurred within the system.
type Event struct {
	Topic     string        // Topic identifies the type of event (e.g., "issue.created", "comment.updated")
	Kind      string        // Kind provides additional classification of the event
	CompanyID string        // CompanyID identifies which company this event belongs to
	Payload   any           // Payload contains the event-specific data
	OccurredAt time.Time    // OccurredAt is when the event occurred
}

// Bus provides pub/sub semantics for domain events.
type Bus interface {
	// Publish sends an event to all subscribers of the given topic.
	Publish(topic string, event Event)

	// Subscribe registers the caller as a subscriber to events on the given topic.
	// It returns a channel to receive events and a function to unsubscribe.
	// Calling the unsubscribe function is the preferred way to clean up the subscription.
	Subscribe(topic string) (<-chan Event, func())
}
