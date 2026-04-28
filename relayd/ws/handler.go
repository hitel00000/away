package ws

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var defaultUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Command struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type SendMessagePayload struct {
	Text string `json:"text"`
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
			_, data, err := conn.ReadMessage()
			if err != nil {
				return
			}

			var cmd Command
			if err := json.Unmarshal(data, &cmd); err != nil {
				log.Printf("failed to parse command: %v", err)
				continue
			}

			if cmd.Type == "send_message" {
				var payload SendMessagePayload
				if err := json.Unmarshal(cmd.Payload, &payload); err != nil {
					log.Printf("failed to parse send_message payload: %v", err)
					continue
				}
				log.Printf("received send_message: %q", payload.Text)
			}
		}
	})
}
