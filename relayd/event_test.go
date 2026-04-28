package relayd

import (
    "bufio"
    "encoding/json"
    "os"
    "path/filepath"
    "testing"
)

type messagePayload struct {
    Nick string `json:"nick"`
    Text string `json:"text"`
}

func TestParseFixtureEvents(t *testing.T) {
    path := filepath.Join("..", "fixtures", "simple_chat.ndjson")
    f, err := os.Open(path)
    if err != nil {
        t.Fatalf("failed to open fixture %q: %v", path, err)
    }
    defer f.Close()

    scanner := bufio.NewScanner(f)
    var events []Event
    for line := 1; scanner.Scan(); line++ {
        var ev Event
        if err := json.Unmarshal(scanner.Bytes(), &ev); err != nil {
            t.Fatalf("failed to parse fixture line %d: %v", line, err)
        }
        events = append(events, ev)
    }
    if err := scanner.Err(); err != nil {
        t.Fatalf("scanner error: %v", err)
    }

    if got, want := len(events), 1; got != want {
        t.Fatalf("expected %d event, got %d", want, got)
    }

    ev := events[0]
    if ev.Type != "message.created" {
        t.Fatalf("expected type message.created, got %q", ev.Type)
    }
    if ev.Version != "1" {
        t.Fatalf("expected version 1, got %q", ev.Version)
    }
    if ev.ID != "evt-1" {
        t.Fatalf("expected id evt-1, got %q", ev.ID)
    }
    if ev.Timestamp.IsZero() {
        t.Fatal("expected timestamp to be set")
    }

    var payload messagePayload
    if err := json.Unmarshal(ev.Payload, &payload); err != nil {
        t.Fatalf("failed to decode payload: %v", err)
    }
    if payload.Nick != "alice" || payload.Text != "hello" {
        t.Fatalf("unexpected payload: %#v", payload)
    }
}
