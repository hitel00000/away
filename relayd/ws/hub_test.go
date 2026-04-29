package ws

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"away/relayd"
	"github.com/gorilla/websocket"
)

func TestHubBroadcastToTwoClients(t *testing.T) {
	hub := NewHub(relayd.NewEventRing())
	srv := httptest.NewServer(Handler(hub))
	defer srv.Close()

	c1 := mustDialWS(t, srv.URL)
	defer c1.Close()
	c2 := mustDialWS(t, srv.URL)
	defer c2.Close()

	event := relayd.Event{Type: "message.created", Version: 1, ID: "evt-1", Timestamp: time.Now().UTC()}
	if err := hub.BroadcastEvent(event); err != nil {
		t.Fatalf("broadcast failed: %v", err)
	}

	got1 := mustReadEvent(t, c1)
	got2 := mustReadEvent(t, c2)
	if got1.ID != event.ID || got2.ID != event.ID {
		t.Fatalf("expected both clients to receive event %q, got %q and %q", event.ID, got1.ID, got2.ID)
	}
}

func TestHandlerReplaysRecentEvents(t *testing.T) {
	ring := relayd.NewEventRing()
	// Pre-fill ring buffer
	e1 := relayd.Event{Type: "message.created", Version: 1, ID: "evt-1"}
	e2 := relayd.Event{Type: "message.created", Version: 1, ID: "evt-2"}
	ring.Append(e1)
	ring.Append(e2)

	hub := NewHub(ring)
	srv := httptest.NewServer(Handler(hub))
	defer srv.Close()

	c := mustDialWS(t, srv.URL)
	defer c.Close()

	// Client should receive e1 and e2
	g1 := mustReadEvent(t, c)
	g2 := mustReadEvent(t, c)

	if g1.ID != e1.ID || g2.ID != e2.ID {
		t.Fatalf("expected e1 and e2, got %q and %q", g1.ID, g2.ID)
	}
}

func TestHandlerReplaysLast20Events(t *testing.T) {
	ring := relayd.NewEventRing()
	for i := 0; i < 25; i++ {
		ring.Append(relayd.Event{
			Type:    "message.created",
			Version: 1,
			ID:      "evt-" + strings.Repeat("x", i+1),
		})
	}

	hub := NewHub(ring)
	srv := httptest.NewServer(Handler(hub))
	defer srv.Close()

	c := mustDialWS(t, srv.URL)
	defer c.Close()

	for i := 0; i < 20; i++ {
		ev := mustReadEvent(t, c)
		want := "evt-" + strings.Repeat("x", i+6)
		if ev.ID != want {
			t.Fatalf("replay[%d]: want %q, got %q", i, want, ev.ID)
		}
	}

	_ = c.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	if _, _, err := c.ReadMessage(); err == nil {
		t.Fatalf("expected only 20 replayed events, but received more")
	}
}

func TestHubDisconnectedClientCleanup(t *testing.T) {
	hub := NewHub(relayd.NewEventRing())
	srv := httptest.NewServer(Handler(hub))
	defer srv.Close()

	c := mustDialWS(t, srv.URL)
	
	// Wait for registration
	deadline := time.Now().Add(2 * time.Second)
	registered := false
	for time.Now().Before(deadline) {
		if hub.ClientCount() == 1 {
			registered = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !registered {
		t.Fatalf("expected one connected client, got %d", hub.ClientCount())
	}
	if err := c.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}

	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if hub.ClientCount() == 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("expected disconnected client to be removed, got %d clients", hub.ClientCount())
}

func TestConcurrentReconnectDuplicate(t *testing.T) {
	ring := relayd.NewEventRing()
	hub := NewHub(ring)
	srv := httptest.NewServer(Handler(hub))
	defer srv.Close()

	const clientCount = 10
	const eventCount = 50

	var wg sync.WaitGroup
	wg.Add(clientCount)

	for i := 0; i < clientCount; i++ {
		go func(cid int) {
			defer wg.Done()
			// Random delay to stagger connections
			time.Sleep(time.Duration(cid*5) * time.Millisecond)

			conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http")+"/ws", nil)
			if err != nil {
				t.Errorf("dial failed: %v", err)
				return
			}
			defer conn.Close()

			seen := make(map[string]int)
			for j := 0; j < 10; j++ {
				_, msgData, err := conn.ReadMessage()
				if err != nil {
					return
				}
				var ev relayd.Event
				if err := json.Unmarshal(msgData, &ev); err != nil {
					t.Errorf("client %d: failed to unmarshal: %v", cid, err)
					return
				}
				if seen[ev.ID] > 0 {
					t.Errorf("client %d: duplicate event detected: %s", cid, ev.ID)
				}
				seen[ev.ID]++
			}
		}(i)
	}

	for i := 0; i < eventCount; i++ {
		ev := relayd.Event{
			Type:    "message.created",
			Version: 1,
			ID:      "evt-" + strings.Repeat("x", i), // Unique ID
		}
		hub.BroadcastEvent(ev)
		time.Sleep(2 * time.Millisecond)
	}

	wg.Wait()
}

func TestHubPreservesClientID(t *testing.T) {
	hub := NewHub(relayd.NewEventRing())
	srv := httptest.NewServer(Handler(hub))
	defer srv.Close()

	c := mustDialWS(t, srv.URL)
	defer c.Close()

	clientID := "test-client-id"
	event := relayd.Event{
		Type:    "message.created",
		Version: 1,
		ID:      "evt-1",
		Payload: json.RawMessage(`{"text":"hello","client_id":"` + clientID + `"}`),
	}

	if err := hub.BroadcastEvent(event); err != nil {
		t.Fatalf("broadcast failed: %v", err)
	}

	got := mustReadEvent(t, c)
	var payload struct {
		ClientID string `json:"client_id"`
	}
	if err := json.Unmarshal(got.Payload, &payload); err != nil {
		t.Fatalf("unmarshal payload failed: %v", err)
	}

	if payload.ClientID != clientID {
		t.Fatalf("expected client_id %q, got %q", clientID, payload.ClientID)
	}
}

func TestIdenticalMessagesWithDifferentClientIDs(t *testing.T) {
	hub := NewHub(relayd.NewEventRing())
	srv := httptest.NewServer(Handler(hub))
	defer srv.Close()

	c := mustDialWS(t, srv.URL)
	defer c.Close()

	ev1 := relayd.Event{
		Type:    "message.created",
		Version: 1,
		ID:      "evt-1",
		Payload: json.RawMessage(`{"text":"hello","client_id":"cl-1"}`),
	}
	ev2 := relayd.Event{
		Type:    "message.created",
		Version: 1,
		ID:      "evt-2",
		Payload: json.RawMessage(`{"text":"hello","client_id":"cl-2"}`),
	}

	hub.BroadcastEvent(ev1)
	hub.BroadcastEvent(ev2)

	g1 := mustReadEvent(t, c)
	g2 := mustReadEvent(t, c)

	if g1.ID != ev1.ID || g2.ID != ev2.ID {
		t.Fatalf("expected ev1 and ev2, got %q and %q", g1.ID, g2.ID)
	}

	var p1, p2 struct {
		ClientID string `json:"client_id"`
	}
	json.Unmarshal(g1.Payload, &p1)
	json.Unmarshal(g2.Payload, &p2)

	if p1.ClientID != "cl-1" || p2.ClientID != "cl-2" {
		t.Fatalf("expected cl-1 and cl-2, got %q and %q", p1.ClientID, p2.ClientID)
	}
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
