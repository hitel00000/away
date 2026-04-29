package relayd

import "sync"

const eventRingSize = 500

type EventRing struct {
	mu     sync.RWMutex
	events []Event
	start  int
	count  int
}

func NewEventRing() *EventRing {
	return &EventRing{
		events: make([]Event, eventRingSize),
	}
}

func (r *EventRing) Append(ev Event) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.count < len(r.events) {
		idx := (r.start + r.count) % len(r.events)
		r.events[idx] = ev
		r.count++
		return
	}

	r.events[r.start] = ev
	r.start = (r.start + 1) % len(r.events)
}

func (r *EventRing) ReplayRecent(limit int) []Event {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if limit <= 0 || r.count == 0 {
		return nil
	}
	if limit > r.count {
		limit = r.count
	}

	out := make([]Event, limit)
	begin := r.count - limit
	for i := 0; i < limit; i++ {
		idx := (r.start + begin + i) % len(r.events)
		out[i] = r.events[idx]
	}
	return out
}
