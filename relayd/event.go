package relayd

import (
	"encoding/json"
	"time"
)

type Event struct {
	Type      string          `json:"type"`
	Version   int             `json:"version"`
	ID        string          `json:"id"`
	Timestamp time.Time       `json:"timestamp"`
	Payload   json.RawMessage `json:"payload"`
}
