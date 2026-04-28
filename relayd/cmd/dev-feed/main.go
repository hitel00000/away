package main

import (
	"away/relayd"
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	fixture := flag.String("fixture", "", "path to fixture file (reads stdin if not provided)")
	relay := flag.String("relay", "http://localhost:8080", "relay server URL")
	delay := flag.Duration("delay", 500*time.Millisecond, "delay between events")
	flag.Parse()

	var input *os.File
	if *fixture == "" {
		input = os.Stdin
	} else {
		f, err := os.Open(*fixture)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		input = f
	}

	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var ev relayd.Event
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			fmt.Fprintf(os.Stderr, "failed to parse event: %v\n", err)
			continue
		}

		payload, err := json.Marshal(ev)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to marshal event: %v\n", err)
			continue
		}

		resp, err := http.Post(
			*relay+"/inject",
			"application/json",
			bytes.NewReader(payload),
		)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to send event: %v\n", err)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			fmt.Fprintf(os.Stderr, "inject returned %d\n", resp.StatusCode)
		}
		resp.Body.Close()

		fmt.Printf("sent: %s\n", ev.ID)
		time.Sleep(*delay)
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}
