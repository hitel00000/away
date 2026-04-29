//go:build linux
// +build linux

package ws

// E-002: multi-buffer routing correctness.
//
// Verifies that payload.target set by the browser reaches the irssi
// FIFO unchanged for four scenarios:
//   1. channel send        (#test)
//   2. DM send             (alice)
//   3. rapid buffer switch (switch target then immediately send)
//   4. alternating sends   (#test, alice, #test, alice)
//
// Each test sends messages through a real WebSocket → Handler → FIFO
// path using a named pipe in t.TempDir().

import (
	"bufio"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"away/relayd"
	"github.com/gorilla/websocket"
)

// fifoReader opens a named pipe for reading (blocks until a writer appears)
// and returns lines on the provided channel. Closes ch when the pipe closes.
func fifoReader(t *testing.T, path string, ch chan<- string) {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Errorf("fifoReader: open: %v", err)
		close(ch)
		return
	}
	go func() {
		defer f.Close()
		defer close(ch)
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			ch <- sc.Text()
		}
	}()
}

// extractTarget parses the "target" field from a FIFO NDJSON line.
func extractTarget(t *testing.T, line string) string {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(line), &m); err != nil {
		t.Fatalf("extractTarget: unmarshal %q: %v", line, err)
	}
	v, ok := m["target"].(string)
	if !ok {
		t.Fatalf("extractTarget: no target in %q", line)
	}
	return v
}

// readFifoLine waits up to 2 s for the next FIFO line.
func readFifoLine(t *testing.T, ch <-chan string) string {
	t.Helper()
	select {
	case line, ok := <-ch:
		if !ok {
			t.Fatal("readFifoLine: channel closed unexpectedly")
		}
		return line
	case <-time.After(2 * time.Second):
		t.Fatal("readFifoLine: timeout waiting for FIFO line")
	}
	return ""
}

// sendCmd sends a send_message WebSocket command.
func sendCmd(t *testing.T, conn *websocket.Conn, target, text, clientID string) {
	t.Helper()
	cmd := map[string]any{
		"type": "send_message",
		"payload": map[string]any{
			"client_id": clientID,
			"target":    target,
			"text":      text,
		},
	}
	b, err := json.Marshal(cmd)
	if err != nil {
		t.Fatalf("sendCmd: marshal: %v", err)
	}
	if err := conn.WriteMessage(websocket.TextMessage, b); err != nil {
		t.Fatalf("sendCmd: write: %v", err)
	}
}

func setupRoutingTest(t *testing.T) (conn *websocket.Conn, fifoLines <-chan string, cleanup func()) {
	t.Helper()

	dir := t.TempDir()
	fifo := filepath.Join(dir, "test.cmd")
	if err := syscall.Mkfifo(fifo, 0600); err != nil {
		t.Fatalf("mkfifo: %v", err)
	}

	// Override package-level FIFO path for the duration of this test.
	orig := irssiCommandFifo
	irssiCommandFifo = fifo

	ch := make(chan string, 20)

	hub := NewHub(relayd.NewEventRing(), nil)
	srv := httptest.NewServer(Handler(hub))

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"
	c, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}

	// The reader must open BEFORE (or concurrently with) the first write,
	// because O_NONBLOCK open from the writer fails when there is no reader.
	fifoReader(t, fifo, ch)

	return c, ch, func() {
		c.Close()
		srv.Close()
		irssiCommandFifo = orig
	}
}

// 1. Channel send — target is a "#channel".
func TestRoutingChannelSend(t *testing.T) {
	conn, lines, cleanup := setupRoutingTest(t)
	defer cleanup()

	sendCmd(t, conn, "#test", "hello channel", "cid-1")
	got := extractTarget(t, readFifoLine(t, lines))
	if got != "#test" {
		t.Fatalf("channel send: expected target %q, got %q", "#test", got)
	}
}

// 2. DM send — target is a nick (no "#" prefix).
func TestRoutingDMSend(t *testing.T) {
	conn, lines, cleanup := setupRoutingTest(t)
	defer cleanup()

	sendCmd(t, conn, "alice", "hello alice", "cid-2")
	got := extractTarget(t, readFifoLine(t, lines))
	if got != "alice" {
		t.Fatalf("DM send: expected target %q, got %q", "alice", got)
	}
}

// 3. Rapid buffer switch — switch target immediately before send.
// Simulates: user opens DM window, switches to channel, sends fast.
func TestRoutingRapidBufferSwitch(t *testing.T) {
	conn, lines, cleanup := setupRoutingTest(t)
	defer cleanup()

	// Mimic rapid UI switch: two sends back-to-back with different targets.
	sendCmd(t, conn, "bob", "dm msg", "cid-3a")
	sendCmd(t, conn, "#test", "chan msg after switch", "cid-3b")

	got1 := extractTarget(t, readFifoLine(t, lines))
	got2 := extractTarget(t, readFifoLine(t, lines))

	if got1 != "bob" {
		t.Fatalf("rapid switch [0]: expected %q, got %q", "bob", got1)
	}
	if got2 != "#test" {
		t.Fatalf("rapid switch [1]: expected %q, got %q", "#test", got2)
	}
}

// 4. Alternating sends across two buffers — channel, DM, channel, DM.
func TestRoutingAlternatingSends(t *testing.T) {
	conn, lines, cleanup := setupRoutingTest(t)
	defer cleanup()

	sequence := []string{"#test", "alice", "#test", "alice"}
	for i, tgt := range sequence {
		sendCmd(t, conn, tgt, "msg", "cid-4-"+strings.Repeat("x", i))
	}

	for i, want := range sequence {
		got := extractTarget(t, readFifoLine(t, lines))
		if got != want {
			t.Fatalf("alternating [%d]: expected %q, got %q", i, want, got)
		}
	}
}
