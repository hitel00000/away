package ws

import (
	"sync"

	"github.com/gorilla/websocket"
)

type Client struct {
	conn   *websocket.Conn
	send   chan []byte
	mu     sync.Mutex
	closed bool
	once   sync.Once
}

func NewClient(conn *websocket.Conn) *Client {
	c := &Client{
		conn: conn,
		send: make(chan []byte, 256), // Buffer for backpressure
	}
	go c.writePump()
	return c
}

func (c *Client) Send(payload []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return websocket.ErrCloseSent
	}

	select {
	case c.send <- payload:
		return nil
	default:
		// Slow consumer: drop connection
		c.mu.Unlock()
		c.Close()
		c.mu.Lock()
		return websocket.ErrCloseSent
	}
}

func (c *Client) writePump() {
	for payload := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, payload); err != nil {
			break
		}
	}
	c.Close()
}

func (c *Client) Close() {
	c.once.Do(func() {
		c.mu.Lock()
		c.closed = true
		c.mu.Unlock()
		close(c.send)
		_ = c.conn.Close()
	})
}
