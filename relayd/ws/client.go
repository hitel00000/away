package ws

import (
	"sync"

	"github.com/gorilla/websocket"
)

type Client struct {
	conn   *websocket.Conn
	mu     sync.Mutex
	closed bool
}

func NewClient(conn *websocket.Conn) *Client {
	return &Client{conn: conn}
}

func (c *Client) Send(payload []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return websocket.ErrCloseSent
	}
	return c.conn.WriteMessage(websocket.TextMessage, payload)
}

func (c *Client) Close() {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	c.closed = true
	c.mu.Unlock()
	_ = c.conn.Close()
}
