package ws

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"away/relayd"
	"github.com/gorilla/websocket"
)

func TestHubBroadcastToTwoClients(t *testing.T) {
	hub := NewHub()
	srv := httptest.NewServer(Handler(hub))
	defer srv.Close()

	c1 := mustDialWS(t, srv.URL)
	defer c1.Close()
	c2 := mustDialWS(t, srv.URL)
	defer c2.Close()

	event := relayd.Event{Type: "message.created", Version: "1", ID: "evt-1", Timestamp: time.Now().UTC()}
	if err := hub.BroadcastEvent(event); err != nil {
		t.Fatalf("broadcast failed: %v", err)
	}

	got1 := mustReadEvent(t, c1)
	got2 := mustReadEvent(t, c2)
	if got1.ID != event.ID || got2.ID != event.ID {
		t.Fatalf("expected both clients to receive event %q, got %q and %q", event.ID, got1.ID, got2.ID)
	}
}

func TestHubDisconnectedClientCleanup(t *testing.T) {
	hub := NewHub()
	srv := httptest.NewServer(Handler(hub))
	defer srv.Close()

	c := mustDialWS(t, srv.URL)
	if hub.ClientCount() != 1 {
		t.Fatalf("expected one connected client, got %d", hub.ClientCount())
	}
	if err := c.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if hub.ClientCount() == 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("expected disconnected client to be removed, got %d clients", hub.ClientCount())
}

func mustDialWS(t *testing.T, serverURL string) *websocket.Conn {
	t.Helper()
	wsURL := "ws" + strings.TrimPrefix(serverURL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	return conn
}

func mustReadEvent(t *testing.T, conn *websocket.Conn) relayd.Event {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	var ev relayd.Event
	if err := json.Unmarshal(msg, &ev); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	return ev
}
