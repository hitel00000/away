package relayd_test

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

var (
	binaryPath string
	buildOnce  sync.Once
)

func getBinary(t *testing.T) string {
	t.Helper()
	buildOnce.Do(func() {
		exeName := "awayd"
		if runtime.GOOS == "windows" {
			exeName += ".exe"
		}
		// Use os.TempDir instead of t.TempDir to avoid deletion after first test
		tmpDir, err := os.MkdirTemp("", "away-e2e-")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		exePath := filepath.Join(tmpDir, exeName)
		cmdBuild := exec.Command("go", "build", "-o", exePath, "./relayd/cmd/awayd/main.go")
		cmdBuild.Dir = ".."
		if err := cmdBuild.Run(); err != nil {
			t.Fatalf("failed to build: %v", err)
		}
		binaryPath = exePath
	})
	return binaryPath
}

type relayServer struct {
	cmd     *exec.Cmd
	outBuf  *bytes.Buffer
	cleanup func()
}

func startRelay(t *testing.T, exePath string, socketPath, fifoPath string) *relayServer {
	t.Helper()

	cmd := exec.Command(exePath)
	cmd.Dir = ".." // Run from root so "web" exists
	cmd.Env = append(os.Environ(),
		"AWAY_IRC_SOCKET="+socketPath,
		"AWAY_IRC_FIFO="+fifoPath,
	)
	outBuf := new(bytes.Buffer)
	cmd.Stdout = outBuf
	cmd.Stderr = outBuf

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start relay: %v", err)
	}

	// Readiness polling for HTTP/WebSocket port
	wsURL := "ws://localhost:8080/ws"
	waitCondition(t, 2*time.Second, func() bool {
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err == nil {
			conn.Close()
			return true
		}
		return false
	})

	// Readiness polling for UNIX socket
	waitCondition(t, 2*time.Second, func() bool {
		conn, err := net.Dial("unix", socketPath)
		if err == nil {
			conn.Close()
			return true
		}
		return false
	})

	return &relayServer{
		cmd:    cmd,
		outBuf: outBuf,
		cleanup: func() {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		},
	}
}

func waitCondition(t *testing.T, timeout time.Duration, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("condition not met within %v", timeout)
}

func readReplayedEvents(t *testing.T, conn *websocket.Conn) []map[string]interface{} {
	t.Helper()
	var events []map[string]interface{}
	// Replayed events should arrive almost immediately. 
	// We read until a small timeout occurs.
	for {
		conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
		var msg map[string]interface{}
		if err := conn.ReadJSON(&msg); err != nil {
			break
		}
		events = append(events, msg)
	}
	return events
}

func TestE2E_Reconnect_Replay(t *testing.T) {
	exePath := getBinary(t)
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "irc.sock")
	fifoPath := filepath.Join(tmpDir, "irc.fifo")

	server := startRelay(t, exePath, socketPath, fifoPath)
	defer server.cleanup()

	// 1. IRSSI sends a message before any client connects
	ircConn, err := net.Dial("unix", socketPath)
	if err != nil {
		t.Fatalf("irssi dial failed: %v", err)
	}
	defer ircConn.Close()

	evID := "evt-replay-1"
	evLine := fmt.Sprintf(`{"id":"%s","type":"message.created","payload":{"nick":"alice","text":"msg1"}}`, evID) + "\n"
	if _, err := ircConn.Write([]byte(evLine)); err != nil {
		t.Fatalf("irssi write failed: %v", err)
	}

	// 2. Client connects and should receive replayed event
	wsURL := "ws://localhost:8080/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()

	replayed := readReplayedEvents(t, conn)
	found := false
	for _, ev := range replayed {
		if ev["id"] == evID {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected event %s in replay, not found", evID)
	}

	// 3. Reconnect and verify no duplicates in replay
	conn2, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial 2 failed: %v", err)
	}
	defer conn2.Close()

	replayed2 := readReplayedEvents(t, conn2)
	count := 0
	for _, ev := range replayed2 {
		if ev["id"] == evID {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected event %s exactly once in re-replay, found %d", evID, count)
	}
}

func TestE2E_Reconnect_RelayRestart(t *testing.T) {
	exePath := getBinary(t)
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "irc.sock")
	fifoPath := filepath.Join(tmpDir, "irc.fifo")

	// Start server 1
	server1 := startRelay(t, exePath, socketPath, fifoPath)
	
	// IRSSI sends msg1
	ircConn, err := net.Dial("unix", socketPath)
	if err != nil {
		t.Fatalf("irssi dial failed: %v", err)
	}
	evLine1 := `{"id":"evt-1","type":"message.created","payload":{"nick":"alice","text":"msg1"}}` + "\n"
	ircConn.Write([]byte(evLine1))
	ircConn.Close()

	// Restart server
	server1.cleanup()
	
	server2 := startRelay(t, exePath, socketPath, fifoPath)
	defer server2.cleanup()

	// IRSSI reconnects and sends msg2
	ircConn2, err := net.Dial("unix", socketPath)
	if err != nil {
		t.Fatalf("irssi dial 2 failed: %v", err)
	}
	defer ircConn2.Close()

	evLine2 := `{"id":"evt-2","type":"message.created","payload":{"nick":"alice","text":"msg2"}}` + "\n"
	ircConn2.Write([]byte(evLine2))

	// Client connects
	wsURL := "ws://localhost:8080/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()

	// Snapshot/Replay should contain msg2
	replayed := readReplayedEvents(t, conn)
	found2 := false
	for _, ev := range replayed {
		if ev["id"] == "evt-2" {
			found2 = true
		}
		if ev["id"] == "evt-1" {
			t.Errorf("did not expect msg1 after restart (no persistence yet)")
		}
	}
	if !found2 {
		t.Errorf("expected msg2 in replay after restart")
	}
}
