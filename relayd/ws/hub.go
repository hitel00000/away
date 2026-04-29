package ws

import (
	"encoding/json"
	"sync"

	"away/relayd"
)

const replayLimit = 50

type Hub struct {
	mu      sync.Mutex
	clients map[*Client]struct{}
	ring    *relayd.EventRing
}

func NewHub(ring *relayd.EventRing) *Hub {
	return &Hub{
		clients: make(map[*Client]struct{}),
		ring:    ring,
	}
}

func (h *Hub) Register(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Replay history while holding the lock to ensure cutover consistency.
	// Any event appended after this point will be sent via BroadcastEvent.
	recent := h.ring.ReplayRecent(replayLimit)
	for _, ev := range recent {
		payload, err := json.Marshal(ev)
		if err == nil {
			_ = c.Send(payload)
		}
	}

	h.clients[c] = struct{}{}
}

func (h *Hub) Unregister(c *Client) {
	h.mu.Lock()
	delete(h.clients, c)
	h.mu.Unlock()
	c.Close()
}

func (h *Hub) BroadcastEvent(ev relayd.Event) error {
	h.mu.Lock()
	h.ring.Append(ev)

	payload, err := json.Marshal(ev)
	if err != nil {
		h.mu.Unlock()
		return err
	}

	clients := make([]*Client, 0, len(h.clients))
	for c := range h.clients {
		clients = append(clients, c)
	}
	h.mu.Unlock()

	for _, c := range clients {
		if err := c.Send(payload); err != nil {
			h.Unregister(c)
		}
	}
	return nil
}

func (h *Hub) ClientCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.clients)
}
