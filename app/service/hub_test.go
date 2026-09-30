package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestHub_LifecycleAndClients(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hub := NewHub()
	go hub.Run(ctx)

	if hub.ActiveClients() != 0 {
		t.Fatalf("expected 0 active clients, got %d", hub.ActiveClients())
	}

	client := &Client{
		hub:  hub,
		send: make(chan []byte, 10),
	}

	// Register
	hub.register <- client
	time.Sleep(20 * time.Millisecond)
	if hub.ActiveClients() != 1 {
		t.Fatalf("expected 1 active client, got %d", hub.ActiveClients())
	}

	// Unregister
	hub.unregister <- client
	time.Sleep(20 * time.Millisecond)
	if hub.ActiveClients() != 0 {
		t.Fatalf("expected 0 active clients, got %d", hub.ActiveClients())
	}
}

func TestHub_BroadcastAndSlowClientDrop(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hub := NewHub()
	go hub.Run(ctx)

	// Client with buffer size 1
	slowClient := &Client{
		hub:  hub,
		send: make(chan []byte, 1),
	}
	hub.register <- slowClient
	time.Sleep(20 * time.Millisecond)

	// First broadcast fills client buffer
	hub.BroadcastJSON(map[string]string{"msg": "first"})
	time.Sleep(20 * time.Millisecond)

	// Second broadcast exceeds client buffer, triggering slow-client drop
	hub.BroadcastJSON(map[string]string{"msg": "second"})
	time.Sleep(20 * time.Millisecond)

	if hub.ActiveClients() != 0 {
		t.Fatalf("expected slow client to be dropped, active clients: %d", hub.ActiveClients())
	}
}

func TestHub_ServeWebSocket(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hub := NewHub()
	go hub.Run(ctx)

	server := httptest.NewServer(http.HandlerFunc(hub.ServeWebSocket))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to dial websocket: %v", err)
	}
	defer ws.Close()

	time.Sleep(30 * time.Millisecond)
	if hub.ActiveClients() != 1 {
		t.Fatalf("expected 1 connected client, got %d", hub.ActiveClients())
	}

	// Broadcast test payload
	testMsg := map[string]string{"event": "telemetry", "device": "TURB-001"}
	hub.BroadcastJSON(testMsg)

	_ = ws.SetReadDeadline(time.Now().Add(2 * time.Second))
	var received map[string]string
	if err := ws.ReadJSON(&received); err != nil {
		t.Fatalf("failed to read json from websocket: %v", err)
	}

	if received["device"] != "TURB-001" {
		t.Fatalf("expected TURB-001, got %v", received)
	}
}
