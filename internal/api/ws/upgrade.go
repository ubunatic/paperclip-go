package ws

import (
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"strings"
)

// hasToken checks if a comma-separated header contains a specific token (case-insensitive).
// RFC 6455 defines these headers as token lists separated by commas.
func hasToken(header, token string) bool {
	for _, v := range strings.Split(header, ",") {
		if strings.EqualFold(strings.TrimSpace(v), token) {
			return true
		}
	}
	return false
}

// Upgrade performs the HTTP → WebSocket handshake using stdlib only.
// Returns the raw net.Conn on success, or writes an error response and returns nil.
func Upgrade(w http.ResponseWriter, r *http.Request) (net.Conn, error) {
	// Check required headers (RFC 6455)
	if !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		http.Error(w, "Upgrade header missing or not 'websocket'", http.StatusBadRequest)
		return nil, fmt.Errorf("upgrade header not websocket")
	}
	if !hasToken(r.Header.Get("Connection"), "Upgrade") {
		http.Error(w, "Connection header must contain 'Upgrade'", http.StatusBadRequest)
		return nil, fmt.Errorf("connection header missing upgrade token")
	}

	// Check WebSocket version (RFC 6455 requires version 13)
	if r.Header.Get("Sec-WebSocket-Version") != "13" {
		w.Header().Set("Sec-WebSocket-Version", "13")
		http.Error(w, "WebSocket version 13 required", http.StatusBadRequest)
		return nil, fmt.Errorf("websocket version not 13")
	}

	// Get Sec-WebSocket-Key
	key := r.Header.Get("Sec-WebSocket-Key")
	if key == "" {
		http.Error(w, "Sec-WebSocket-Key header missing", http.StatusBadRequest)
		return nil, fmt.Errorf("sec-websocket-key missing")
	}

	// Compute accept key: base64(sha1(key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	const magicString = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
	h := sha1.New()
	h.Write([]byte(key + magicString))
	acceptKey := base64.StdEncoding.EncodeToString(h.Sum(nil))

	// Hijack the connection
	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "hijack not supported", http.StatusInternalServerError)
		return nil, fmt.Errorf("hijacker not available")
	}
	conn, rw, err := hj.Hijack()
	if err != nil {
		return nil, fmt.Errorf("hijack failed: %w", err)
	}

	// Write 101 response
	response := fmt.Sprintf(
		"HTTP/1.1 101 Switching Protocols\r\n"+
			"Upgrade: websocket\r\n"+
			"Connection: Upgrade\r\n"+
			"Sec-WebSocket-Accept: %s\r\n"+
			"\r\n",
		acceptKey,
	)
	if _, err := rw.WriteString(response); err != nil {
		conn.Close()
		return nil, fmt.Errorf("write 101 response failed: %w", err)
	}
	if err := rw.Flush(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("flush 101 response failed: %w", err)
	}

	return conn, nil
}

// WriteTextFrame writes data as a WebSocket text frame to conn.
// Handles payloads up to 64KB. Server-to-client frames are never masked.
func WriteTextFrame(conn net.Conn, data []byte) error {
	frame := make([]byte, 0, len(data)+10)

	// First byte: FIN=1 (0x80), RSV=0, opcode=1 (0x01) → 0x81
	frame = append(frame, 0x81)

	// Second byte: MASK=0 (server doesn't mask), payload length
	payloadLen := len(data)
	if payloadLen <= 125 {
		frame = append(frame, byte(payloadLen))
	} else if payloadLen <= 0xFFFF {
		frame = append(frame, 126, byte(payloadLen>>8), byte(payloadLen))
	} else {
		frame = append(frame, 127,
			byte(payloadLen>>56), byte(payloadLen>>48),
			byte(payloadLen>>40), byte(payloadLen>>32),
			byte(payloadLen>>24), byte(payloadLen>>16),
			byte(payloadLen>>8),  byte(payloadLen),
		)
	}

	// Append payload
	frame = append(frame, data...)

	// Write to connection
	_, err := conn.Write(frame)
	return err
}
