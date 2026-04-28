package ws

import (
	"net/http"

	"github.com/gorilla/websocket"
)

var defaultUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func Handler(hub *Hub) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := defaultUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}

		client := NewClient(conn)
		hub.Register(client)
		defer hub.Unregister(client)

		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	})
}
