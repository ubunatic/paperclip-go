package ws

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/ubunatic/paperclip-go/internal/events"
)

// Handler returns an HTTP handler that upgrades to WebSocket and fans out events.
func Handler(bus events.Bus) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract companyId from query string
		companyID := r.URL.Query().Get("companyId")
		if companyID == "" {
			http.Error(w, "companyId query parameter required", http.StatusBadRequest)
			return
		}

		// Upgrade to WebSocket
		conn, err := Upgrade(w, r)
		if err != nil {
			// Upgrade already wrote error response
			return
		}
		defer conn.Close()

		// Subscribe to company events
		topic := "company:" + companyID
		ch, unsub := bus.Subscribe(topic)
		defer unsub()

		// Fan out events to client
		for {
			select {
			case event, ok := <-ch:
				if !ok {
					// Channel closed (unsubscribe called)
					return
				}

				// Marshal event to JSON
				data, err := json.Marshal(event)
				if err != nil {
					// Silently skip bad events
					continue
				}

				// Set write deadline to prevent goroutine leaks on slow clients
				conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
				// Write to WebSocket
				if err := WriteTextFrame(conn, data); err != nil {
					// Client disconnected or write failed
					return
				}

			case <-r.Context().Done():
				// Server shutdown or request cancelled
				return
			}
		}
	}
}
