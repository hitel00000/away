package main

import (
	"away/relayd"
	"away/relayd/ws"
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
)

func main() {
	ring := relayd.NewEventRing()
	hub := ws.NewHub(ring)

	const ircSocket = "/tmp/away/irc-companion.sock"

	_ = os.Remove(ircSocket)

	ln, err := net.Listen("unix", ircSocket)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		log.Printf("irssi ingest listening on %s", ircSocket)

		for {
			conn, err := ln.Accept()
			if err != nil {
				continue
			}

			go func(c net.Conn) {
				defer c.Close()

				scanner := bufio.NewScanner(c)

				for scanner.Scan() {

					var ev relayd.Event

					line := scanner.Bytes()

					if err := json.Unmarshal(line, &ev); err != nil {
						log.Printf(
							"invalid irssi event: %v",
							err,
						)
						continue
					}

					if err := hub.BroadcastEvent(ev); err != nil {
						log.Printf(
							"broadcast failed: %v",
							err,
						)
					}
				}
			}(conn)
		}
	}()

	// Static file serving for web UI
	http.Handle("/", http.FileServer(http.Dir("web")))

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
