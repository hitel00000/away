package relayd

import (
	"encoding/json"
	"os"
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
}
