package ws

import (
	"encoding/json"
	"sync"

	"away/relayd"
)

type Hub struct {
	mu      sync.RWMutex
	clients map[*Client]struct{}
}

func NewHub() *Hub {
	return &Hub{clients: make(map[*Client]struct{})}
}

func (h *Hub) Register(c *Client) {
	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()
}

func (h *Hub) Unregister(c *Client) {
	h.mu.Lock()
	if _, ok := h.clients[c]; ok {
		delete(h.clients, c)
	}
	h.mu.Unlock()
	c.Close()
}

func (h *Hub) BroadcastEvent(ev relayd.Event) error {
	payload, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	h.Broadcast(payload)
	return nil
}

func (h *Hub) Broadcast(payload []byte) {
	h.mu.RLock()
	clients := make([]*Client, 0, len(h.clients))
	for c := range h.clients {
		clients = append(clients, c)
	}
	h.mu.RUnlock()

	for _, c := range clients {
		if err := c.Send(payload); err != nil {
			h.Unregister(c)
		}
	}
}

func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
