package main

import (
	"away/relayd"
	"away/relayd/ws"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

func main() {
	hub := ws.NewHub()
	ring := relayd.NewEventRing()

	// Websocket handler
	http.Handle("/ws", ws.Handler(hub))

	// Event injection endpoint for dev-feed
	http.HandleFunc("/inject", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		var ev relayd.Event
		if err := json.Unmarshal(body, &ev); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		ring.Append(ev)
		if err := hub.BroadcastEvent(ev); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	})

	addr := ":8080"
	fmt.Printf("awayd listening on %s\n", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}
