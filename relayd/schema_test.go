package relayd_test

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
	"time"
)

type Event struct {
	Type      string          `json:"type"`
	Version   int             `json:"version"`
	ID        string          `json:"id"`
	Timestamp time.Time       `json:"timestamp"`
	Payload   json.RawMessage `json:"payload"`
}

type MessageCreatedPayload struct {
	Network    string   `json:"network"`
	BufferID   string   `json:"buffer_id"`
	BufferType string   `json:"buffer_type"`
	Nick       string   `json:"nick"`
	Text       string   `json:"text"`
	Highlight  bool     `json:"highlight"`
	Tags       []string `json:"tags"`
	ClientID   string   `json:"client_id"`
}

type DMCreatedPayload struct {
	Network  string `json:"network"`
	Peer     string `json:"peer"`
	Text     string `json:"text"`
	ClientID string `json:"client_id"`
}

type Command struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type SendMessagePayload struct {
	ClientID string `json:"client_id"`
	Text     string `json:"text"`
	Target   string `json:"target"`
}

func TestSchema_MessageCreated(t *testing.T) {
	b, err := os.ReadFile("../fixtures/events_v1/message.created.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	var ev Event
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&ev); err != nil {
		t.Fatalf("failed to unmarshal Event: %v", err)
	}

	if ev.Type != "message.created" || ev.Version != 1 || ev.ID == "" {
		t.Errorf("missing core event fields: %+v", ev)
	}

	var payload MessageCreatedPayload
	pDec := json.NewDecoder(bytes.NewReader(ev.Payload))
	pDec.DisallowUnknownFields()
	if err := pDec.Decode(&payload); err != nil {
		t.Fatalf("failed to unmarshal MessageCreatedPayload: %v", err)
	}

	if payload.BufferID == "" || payload.Nick == "" || payload.Text == "" {
		t.Errorf("missing required payload fields in message.created: %+v", payload)
	}
}

func TestSchema_DMCreated(t *testing.T) {
	b, err := os.ReadFile("../fixtures/events_v1/dm.created.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	var ev Event
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&ev); err != nil {
		t.Fatalf("failed to unmarshal Event: %v", err)
	}

	if ev.Type != "dm.created" || ev.Version != 1 || ev.ID == "" {
		t.Errorf("missing core event fields: %+v", ev)
	}

	var payload DMCreatedPayload
	pDec := json.NewDecoder(bytes.NewReader(ev.Payload))
	pDec.DisallowUnknownFields()
	if err := pDec.Decode(&payload); err != nil {
		t.Fatalf("failed to unmarshal DMCreatedPayload: %v", err)
	}

	if payload.Peer == "" || payload.Text == "" {
		t.Errorf("missing required payload fields in dm.created: %+v", payload)
	}
}

func TestSchema_SendMessage(t *testing.T) {
	b, err := os.ReadFile("../fixtures/events_v1/send_message.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	var cmd Command
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&cmd); err != nil {
		t.Fatalf("failed to unmarshal Command: %v", err)
	}

	if cmd.Type != "send_message" {
		t.Errorf("missing core command fields: %+v", cmd)
	}

	var payload SendMessagePayload
	pDec := json.NewDecoder(bytes.NewReader(cmd.Payload))
	pDec.DisallowUnknownFields()
	if err := pDec.Decode(&payload); err != nil {
		t.Fatalf("failed to unmarshal SendMessagePayload: %v", err)
	}

	if payload.Target == "" || payload.Text == "" {
		t.Errorf("missing required payload fields in send_message: %+v", payload)
	}
}
