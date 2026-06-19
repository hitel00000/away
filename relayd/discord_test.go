package relayd

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestSanitizeTargetName(t *testing.T) {
	tests := []struct {
		target   string
		expected string
	}{
		{"#away", "🟢-away"},
		{"#Away-Channel", "🟢-away-channel"},
		{"dm:bob", "👤-bob"},
		{"dm:Alice_Smith", "👤-alice-smith"},
		{"#test_space", "🟢-test-space"},
	}

	for _, tc := range tests {
		actual := sanitizeTargetName(tc.target)
		if actual != tc.expected {
			t.Errorf("sanitizeTargetName(%q) = %q, expected %q", tc.target, actual, tc.expected)
		}
	}
}

func TestDiscordBridge_MessageCreate(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "away-fifo-test-*")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close() // Close it so writeFifo can write to it

	b := &DiscordBridge{
		guildID:    "mock-guild-id",
		fifoPath:   tmpFile.Name(),
		cache:      make(map[string]string),
		idToTarget: make(map[string]string),
	}

	b.cache["#test-chan"] = "chan-123"
	b.idToTarget["chan-123"] = "#test-chan"

	b.cache["dm:alice"] = "dm-456"
	b.idToTarget["dm-456"] = "dm:alice"

	s := &discordgo.Session{
		State: discordgo.NewState(),
	}
	s.State.User = &discordgo.User{ID: "bot-id"}

	// 1. Test public channel message egress
	mPublic := &discordgo.MessageCreate{
		Message: &discordgo.Message{
			Author:    &discordgo.User{ID: "user-id"},
			GuildID:   "mock-guild-id",
			ChannelID: "chan-123",
			Content:   "hello channel",
		},
	}

	b.messageCreate(s, mPublic)

	content, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("failed to read temp file: %v", err)
	}

	var parsedPublic map[string]any
	if err := json.Unmarshal(content, &parsedPublic); err != nil {
		t.Fatalf("failed to parse FIFO output: %v", err)
	}

	if parsedPublic["action"] != "send_message" {
		t.Errorf("expected action send_message, got %v", parsedPublic["action"])
	}
	if parsedPublic["target"] != "#test-chan" {
		t.Errorf("expected target #test-chan, got %v", parsedPublic["target"])
	}
	if parsedPublic["text"] != "hello channel" {
		t.Errorf("expected text 'hello channel', got %v", parsedPublic["text"])
	}
	if parsedPublic["client_id"] != "discord" {
		t.Errorf("expected client_id discord, got %v", parsedPublic["client_id"])
	}

	// Clear temp file content
	_ = os.WriteFile(tmpFile.Name(), []byte{}, 0644)

	// 2. Test private query message egress
	mDM := &discordgo.MessageCreate{
		Message: &discordgo.Message{
			Author:    &discordgo.User{ID: "user-id"},
			GuildID:   "mock-guild-id",
			ChannelID: "dm-456",
			Content:   "hello alice",
		},
	}

	b.messageCreate(s, mDM)

	contentDM, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("failed to read temp file: %v", err)
	}

	var parsedDM map[string]any
	if err := json.Unmarshal(contentDM, &parsedDM); err != nil {
		t.Fatalf("failed to parse FIFO output: %v", err)
	}

	if parsedDM["action"] != "send_message" {
		t.Errorf("expected action send_message, got %v", parsedDM["action"])
	}
	if parsedDM["target"] != "alice" {
		t.Errorf("expected target alice, got %v", parsedDM["target"])
	}
	if parsedDM["text"] != "hello alice" {
		t.Errorf("expected text 'hello alice', got %v", parsedDM["text"])
	}
	if parsedDM["client_id"] != "discord" {
		t.Errorf("expected client_id discord, got %v", parsedDM["client_id"])
	}
}

type mockTransport struct {
	roundTrip func(req *http.Request) (*http.Response, error)
}

func (t *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return t.roundTrip(req)
}

func TestDiscordBridge_WebhookSend(t *testing.T) {
	s, err := discordgo.New("Bot token")
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}
	s.State.User = &discordgo.User{ID: "bot-id"}

	var apiCalls []string
	var mockWebhooks []*discordgo.Webhook

	s.Client = &http.Client{
		Transport: &mockTransport{
			roundTrip: func(req *http.Request) (*http.Response, error) {
				apiCalls = append(apiCalls, req.Method+" "+req.URL.Path)

				if req.Method == "GET" && strings.HasSuffix(req.URL.Path, "/webhooks") {
					respData, _ := json.Marshal(mockWebhooks)
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewReader(respData)),
						Header:     make(http.Header),
					}, nil
				}

				if req.Method == "POST" && strings.HasSuffix(req.URL.Path, "/webhooks") {
					wh := &discordgo.Webhook{
						ID:        "wh-id-123",
						Token:     "wh-token-abc",
						Name:      "Away-Bridge",
						ChannelID: "chan-123",
					}
					respData, _ := json.Marshal(wh)
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewReader(respData)),
						Header:     make(http.Header),
					}, nil
				}

				if req.Method == "POST" && strings.Contains(req.URL.Path, "/webhooks/wh-id-123/wh-token-abc") {
					return &http.Response{
						StatusCode: http.StatusNoContent,
						Body:       io.NopCloser(bytes.NewReader([]byte{})),
						Header:     make(http.Header),
					}, nil
				}

				return &http.Response{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(bytes.NewReader([]byte(`{"message": "Unknown Webhook", "code": 10015}`))),
					Header:     make(http.Header),
				}, nil
			},
		},
	}

	b := &DiscordBridge{
		session:  s,
		webhooks: make(map[string]*discordgo.Webhook),
	}

	err = b.sendWebhookMessage("chan-123", "alice", "hello")
	if err != nil {
		t.Fatalf("sendWebhookMessage failed: %v", err)
	}

	apiCalls = nil
	err = b.sendWebhookMessage("chan-123", "alice", "hello again")
	if err != nil {
		t.Fatalf("sendWebhookMessage failed: %v", err)
	}

	if len(apiCalls) != 1 || !strings.Contains(apiCalls[0], "POST") {
		t.Errorf("expected only 1 WebhookExecute call, got: %v", apiCalls)
	}
}

func TestCleanHTMLForDiscord(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "hello"},
		{"<b>bold</b>", "**bold**"},
		{"<i>italic</i>", "*italic*"},
		{"<u>underline</u>", "__underline__"},
		{"&lt;escaped&gt; &amp; &quot;quotes&quot;", "<escaped> & \"quotes\""},
		{"<b>bold <i>italic</i></b>", "**bold *italic***"},
		{"<span class=\"irc-fg-1 irc-bg-2\">colored</span> text", "colored text"},
		{"<b>bold <span class=\"irc-fg-1\">colored</span></b>", "**bold colored**"},
	}

	for _, tc := range tests {
		actual := CleanHTMLForDiscord(tc.input)
		if actual != tc.expected {
			t.Errorf("CleanHTMLForDiscord(%q) = %q, expected %q", tc.input, actual, tc.expected)
		}
	}
}

func TestParseNestedSender(t *testing.T) {
	tests := []struct {
		input     string
		expectedN string
		expectedM string
		expectedO bool
	}{
		{"<jw> hello", "jw", "hello", true},
		{"[jw] hello", "jw", "hello", true},
		{"(jw) hello", "jw", "hello", true},
		{"jw: hello", "jw", "hello", true},
		{"<jw_123> hello world", "jw_123", "hello world", true},
		{"<jw (via ||)> hello", "jw (via ||)", "hello", true},
		{"just a regular message", "", "", false},
		{"no_colon_no_brackets", "", "", false},
	}

	for _, tc := range tests {
		n, m, ok := parseNestedSender(tc.input)
		if ok != tc.expectedO {
			t.Errorf("parseNestedSender(%q) ok = %t, expected %t", tc.input, ok, tc.expectedO)
		}
		if n != tc.expectedN || m != tc.expectedM {
			t.Errorf("parseNestedSender(%q) = (%q, %q), expected (%q, %q)", tc.input, n, m, tc.expectedN, tc.expectedM)
		}
	}
}
