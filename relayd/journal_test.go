package relayd

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestEventJournalAppendWritesEvent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "events.ndjson")
	j := NewEventJournal(path, 10)

	ev := Event{Type: "message.created", Version: 1, ID: "evt-1"}
	if err := j.Append(ev); err != nil {
		t.Fatalf("append failed: %v", err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open failed: %v", err)
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	if !sc.Scan() {
		t.Fatal("expected one line")
	}
	var got Event
	if err := json.Unmarshal(sc.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if got.ID != ev.ID {
		t.Fatalf("want %q, got %q", ev.ID, got.ID)
	}
}

func TestEventJournalRestoreSkipsMalformedLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "events.ndjson")
	content := "{bad json}\n" +
		`{"type":"message.created","version":1,"id":"evt-1"}` + "\n" +
		"not-json\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	j := NewEventJournal(path, 10)
	events, err := j.LoadRecent()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("want 1 event, got %d", len(events))
	}
	if events[0].ID != "evt-1" {
		t.Fatalf("want evt-1, got %q", events[0].ID)
	}
}

func TestEventJournalRetentionIsBounded(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "events.ndjson")
	j := NewEventJournal(path, 3)

	for i := 0; i < 5; i++ {
		ev := Event{Type: "message.created", Version: 1, ID: "evt-" + strconv.Itoa(i)}
		if err := j.Append(ev); err != nil {
			t.Fatalf("append %d failed: %v", i, err)
		}
	}

	events, err := j.LoadRecent()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("want 3 events, got %d", len(events))
	}
	if events[0].ID != "evt-2" || events[2].ID != "evt-4" {
		t.Fatalf("unexpected retained IDs: %q .. %q", events[0].ID, events[2].ID)
	}
}

func TestEventJournalRestoreKeepsDuplicateEventIDs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "events.ndjson")
	content := `{"type":"message.created","version":1,"id":"evt-1"}` + "\n" +
		`{"type":"message.created","version":1,"id":"evt-1"}` + "\n" +
		`{"type":"message.created","version":1,"id":"evt-2"}` + "\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	j := NewEventJournal(path, 10)
	events, err := j.LoadRecent()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("want 3 events including duplicates, got %d", len(events))
	}
	if events[0].ID != "evt-1" || events[1].ID != "evt-1" || events[2].ID != "evt-2" {
		t.Fatalf("unexpected IDs: %q, %q, %q", events[0].ID, events[1].ID, events[2].ID)
	}
}
