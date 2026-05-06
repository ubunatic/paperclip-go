package events

import "sync"

// MemBus is an in-memory implementation of the Bus interface.
type MemBus struct {
	mu    sync.RWMutex
	subs  map[string][]chan Event
}

// NewMemBus returns a new in-memory event bus.
func NewMemBus() *MemBus {
	return &MemBus{
		subs: make(map[string][]chan Event),
	}
}

// Publish sends an event to all subscribers of the given topic.
// Sends are non-blocking; if a subscriber's channel is full, the event is dropped for that subscriber.
func (b *MemBus) Publish(topic string, e Event) {
	b.mu.RLock()
	snapshot := make([]chan Event, len(b.subs[topic]))
	copy(snapshot, b.subs[topic])
	b.mu.RUnlock()

	for _, ch := range snapshot {
		// Non-blocking send; drop if the channel buffer is full.
		select {
		case ch <- e:
		default:
			// Channel buffer full; drop this event for this subscriber.
		}
	}
}

// Subscribe registers a new subscriber on the given topic.
// It returns a channel to receive events and an unsubscribe function.
// The unsubscribe function must be called to clean up the subscription.
func (b *MemBus) Subscribe(topic string) (<-chan Event, func()) {
	ch := make(chan Event, 32)

	b.mu.Lock()
	b.subs[topic] = append(b.subs[topic], ch)
	b.mu.Unlock()

	// Return an unsubscribe closure that removes this channel from the subscription list.
	unsub := func() {
		b.mu.Lock()
		defer b.mu.Unlock()

		chans := b.subs[topic]
		for i, c := range chans {
			if c == ch {
				// Remove by swapping with the last element and truncating.
				b.subs[topic] = append(chans[:i], chans[i+1:]...)
				close(ch)
				break
			}
		}
	}

	return ch, unsub
}
