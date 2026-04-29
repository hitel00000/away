package relayd

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

const defaultJournalMaxEntries = 500

type EventJournal struct {
	mu         sync.Mutex
	path       string
	maxEntries int
	writeCount int
}

func NewEventJournal(path string, maxEntries int) *EventJournal {
	if maxEntries <= 0 {
		maxEntries = defaultJournalMaxEntries
	}
	return &EventJournal{path: path, maxEntries: maxEntries}
}

func (j *EventJournal) LoadRecent() ([]Event, error) {
	f, err := os.Open(j.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var events []Event
	seen := make(map[string]struct{})
	for scanner.Scan() {
		var ev Event
		if err := json.Unmarshal(scanner.Bytes(), &ev); err != nil {
			continue
		}
		if ev.ID != "" {
			if _, ok := seen[ev.ID]; ok {
				continue
			}
			seen[ev.ID] = struct{}{}
		}
		events = append(events, ev)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if len(events) > j.maxEntries {
		events = events[len(events)-j.maxEntries:]
	}
	return events, nil
}

func (j *EventJournal) Append(ev Event) error {
	j.mu.Lock()
	defer j.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(j.path), 0755); err != nil {
		return err
	}

	line, err := json.Marshal(ev)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(j.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	if _, err := f.Write(append(line, '\n')); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}

	j.writeCount++
	if j.writeCount%50 == 0 {
		return j.compactLocked()
	}
	return nil
}

func (j *EventJournal) compactLocked() error {
	f, err := os.Open(j.path)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	lines := make([][]byte, 0, j.maxEntries+1)
	for scanner.Scan() {
		line := append([]byte(nil), scanner.Bytes()...)
		lines = append(lines, line)
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	if len(lines) <= j.maxEntries {
		return nil
	}

	keep := lines[len(lines)-j.maxEntries:]
	tmp := j.path + ".tmp"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	for _, line := range keep {
		if _, err := out.Write(append(line, '\n')); err != nil {
			out.Close()
			return err
		}
	}
	if err := out.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, j.path)
}
