package relayd

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestEventRingReplayRecentFixture(t *testing.T) {
	dmEvents := loadFixtureEvents(t, filepath.Join("..", "fixtures", "dm.ndjson"))
	chatEvents := loadFixtureEvents(t, filepath.Join("..", "fixtures", "simple_chat.ndjson"))

	ring := NewEventRing()
	for _, ev := range chatEvents {
		ring.Append(ev)
	}
	for _, ev := range dmEvents {
		ring.Append(ev)
	}

	recent := ring.ReplayRecent(2)
	if got, want := len(recent), 2; got != want {
		t.Fatalf("expected %d events, got %d", want, got)
	}
	if recent[0].ID != "evt-1" || recent[1].ID != "evt-2" {
		t.Fatalf("unexpected recent IDs: %q, %q", recent[0].ID, recent[1].ID)
	}
}

func TestEventRingKeepsLast500(t *testing.T) {
	events := loadFixtureEvents(t, filepath.Join("..", "fixtures", "dm.ndjson"))
	if len(events) == 0 {
		t.Fatal("fixture must include at least one event")
	}

	ring := NewEventRing()
	total := 650
	for i := 0; i < total; i++ {
		ev := events[0]
		ev.ID = "evt-overflow-" + strconv.Itoa(i)
		ring.Append(ev)
	}

	recent := ring.ReplayRecent(500)
	if got, want := len(recent), 500; got != want {
		t.Fatalf("expected %d events, got %d", want, got)
	}
	if recent[0].ID != "evt-overflow-150" {
		t.Fatalf("expected oldest kept event evt-overflow-150, got %q", recent[0].ID)
	}
	if recent[499].ID != "evt-overflow-649" {
		t.Fatalf("expected newest event evt-overflow-649, got %q", recent[499].ID)
	}
}

func loadFixtureEvents(t *testing.T, path string) []Event {
	t.Helper()

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("failed to open fixture %q: %v", path, err)
	}
	defer f.Close()

	var events []Event
	scanner := bufio.NewScanner(f)
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
	return events
}
