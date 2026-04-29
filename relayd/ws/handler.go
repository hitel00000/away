package ws

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

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
	ClientID string `json:"client_id"`
	Text     string `json:"text"`
	Target   string `json:"target"`
}

const irssiCommandFifo = "/tmp/away/irc-companion.cmd"

// writeFifo writes a single NDJSON line to the irssi command FIFO.
// Opens and closes the file per call to avoid fd leaks inside the read loop.
func writeFifo(line []byte) error {
	f, err := os.OpenFile(irssiCommandFifo, os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(line, '\n'))
	return err
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
				log.Printf("received send_message: %q %q", payload.Target, payload.Text)

				line, err := json.Marshal(map[string]any{
					"action":    "send_message",
					"client_id": payload.ClientID,
					"target":    payload.Target,
					"text":      payload.Text,
				})

				if err != nil {
					return
				}

				if err := writeFifo(line); err != nil {
					log.Printf("fifo write failed: %v", err)
				}

			}
		}
	})
}
