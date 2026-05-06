package events

import (
	"testing"
	"time"
)

// TestPublishReceived verifies that published events are received by subscribers.
func TestPublishReceived(t *testing.T) {
	bus := NewMemBus()
	ch, unsub := bus.Subscribe("test.topic")
	defer unsub()

	event := Event{
		Topic:     "test.topic",
		Kind:      "created",
		CompanyID: "company-1",
		Payload:   "test payload",
		OccurredAt: time.Now(),
	}

	bus.Publish("test.topic", event)

	select {
	case received := <-ch:
		if received.Topic != event.Topic {
			t.Errorf("expected topic %q, got %q", event.Topic, received.Topic)
		}
		if received.CompanyID != event.CompanyID {
			t.Errorf("expected companyID %q, got %q", event.CompanyID, received.CompanyID)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("did not receive published event within 100ms")
	}
}

// TestUnsubscribeStopsDelivery verifies that unsubscribing stops event delivery.
func TestUnsubscribeStopsDelivery(t *testing.T) {
	bus := NewMemBus()
	ch, unsub := bus.Subscribe("test.topic")

	unsub()

	// Channel should be closed after unsubscribe.
	_, ok := <-ch
	if ok {
		t.Fatal("expected channel to be closed after unsubscribe")
	}
}

// TestMultipleSubscribers verifies that multiple subscribers receive the same published event.
func TestMultipleSubscribers(t *testing.T) {
	bus := NewMemBus()
	ch1, unsub1 := bus.Subscribe("test.topic")
	defer unsub1()
	ch2, unsub2 := bus.Subscribe("test.topic")
	defer unsub2()

	event := Event{
		Topic:     "test.topic",
		Kind:      "created",
		CompanyID: "company-1",
		Payload:   "test payload",
		OccurredAt: time.Now(),
	}

	bus.Publish("test.topic", event)

	// Both subscribers should receive the event.
	select {
	case received1 := <-ch1:
		if received1.Topic != event.Topic {
			t.Errorf("subscriber 1: expected topic %q, got %q", event.Topic, received1.Topic)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("subscriber 1 did not receive published event within 100ms")
	}

	select {
	case received2 := <-ch2:
		if received2.Topic != event.Topic {
			t.Errorf("subscriber 2: expected topic %q, got %q", event.Topic, received2.Topic)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("subscriber 2 did not receive published event within 100ms")
	}
}

// TestPublishDropsOnFullBuffer verifies that publishes don't block when a subscriber's buffer is full.
func TestPublishDropsOnFullBuffer(t *testing.T) {
	bus := NewMemBus()
	ch, unsub := bus.Subscribe("test.topic")
	defer unsub()

	// Fill the channel buffer (capacity is 32).
	for i := 0; i < 32; i++ {
		event := Event{
			Topic:     "test.topic",
			Kind:      "created",
			CompanyID: "company-1",
			Payload:   i,
			OccurredAt: time.Now(),
		}
		bus.Publish("test.topic", event)
	}

	// Publish one more event; should not block even though buffer is full.
	// This will be dropped, but Publish should return immediately.
	done := make(chan struct{})
	go func() {
		event := Event{
			Topic:     "test.topic",
			Kind:      "created",
			CompanyID: "company-1",
			Payload:   "overflow",
			OccurredAt: time.Now(),
		}
		bus.Publish("test.topic", event)
		close(done)
	}()

	select {
	case <-done:
		// Expected: Publish returned without blocking
	case <-time.After(1 * time.Second):
		t.Fatal("Publish blocked when channel buffer was full")
	}

	// Verify we can still receive the 32 buffered events.
	for i := 0; i < 32; i++ {
		select {
		case received := <-ch:
			if payload, ok := received.Payload.(int); ok && payload != i {
				t.Errorf("expected payload %d, got %d", i, payload)
			}
		case <-time.After(100 * time.Millisecond):
			t.Fatalf("did not receive buffered event %d", i)
		}
	}
}
