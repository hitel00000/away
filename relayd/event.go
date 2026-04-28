package relayd

import (
    "encoding/json"
    "time"
)

type Event struct {
    Type      string          `json:"type"`
    Version   string          `json:"version"`
    ID        string          `json:"id"`
    Timestamp time.Time       `json:"timestamp"`
    Payload   json.RawMessage `json:"payload"`
}
