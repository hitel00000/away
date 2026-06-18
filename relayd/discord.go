package relayd

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

type DiscordBridge struct {
	mu            sync.Mutex
	session       *discordgo.Session
	token         string
	guildID       string
	activeCatID   string
	inactiveCatID string
	fifoPath      string
}

type BufferInfo struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	Label string `json:"label"`
}

type SnapshotPayload struct {
	Buffers []BufferInfo `json:"buffers"`
}

type MessagePayload struct {
	Network    string   `json:"network"`
	BufferID   string   `json:"buffer_id"`
	BufferType string   `json:"buffer_type"`
	Nick       string   `json:"nick"`
	Text       string   `json:"text"`
	Highlight  bool     `json:"highlight"`
	Tags       []string `json:"tags"`
	ClientID   string   `json:"client_id"`
}

type DMPayload struct {
	Network  string `json:"network"`
	Peer     string `json:"peer"`
	Text     string `json:"text"`
	ClientID string `json:"client_id"`
}

func NewDiscordBridge() *DiscordBridge {
	token := os.Getenv("AWAY_DISCORD_TOKEN")
	guildID := os.Getenv("AWAY_DISCORD_GUILD_ID")
	if token == "" || guildID == "" {
		log.Println("AWAY_DISCORD_TOKEN or AWAY_DISCORD_GUILD_ID not set. Discord bridge is disabled.")
		return nil
	}

	fifoPath := "/tmp/away/irc-companion.cmd"
	if val := os.Getenv("AWAY_IRC_FIFO"); val != "" {
		fifoPath = val
	}

	return &DiscordBridge{
		token:    token,
		guildID:  guildID,
		fifoPath: fifoPath,
	}
}

func (b *DiscordBridge) Start() error {
	dg, err := discordgo.New("Bot " + b.token)
	if err != nil {
		return err
	}
	b.session = dg

	// Register event handlers
	dg.AddHandler(b.messageCreate)

	err = dg.Open()
	if err != nil {
		return err
	}

	// Resolve active/inactive categories
	activeCat, err := b.getOrCreateCategory("💬 ACTIVE CHANNELS")
	if err != nil {
		return fmt.Errorf("failed to create active category: %w", err)
	}
	b.activeCatID = activeCat

	inactiveCat, err := b.getOrCreateCategory("💤 INACTIVE CHANNELS")
	if err != nil {
		return fmt.Errorf("failed to create inactive category: %w", err)
	}
	b.inactiveCatID = inactiveCat

	log.Println("Discord bot connected and running.")
	return nil
}

func (b *DiscordBridge) Close() {
	if b.session != nil {
		b.session.Close()
	}
}

func (b *DiscordBridge) HandleEvent(ev Event) {
	switch ev.Type {
	case "message.created":
		var p MessagePayload
		if err := json.Unmarshal(ev.Payload, &p); err != nil {
			log.Printf("discord: failed to unmarshal message payload: %v", err)
			return
		}
		b.handleIRCMessage(p)

	case "dm.created":
		var p DMPayload
		if err := json.Unmarshal(ev.Payload, &p); err != nil {
			log.Printf("discord: failed to unmarshal dm payload: %v", err)
			return
		}
		b.handleIRCDM(p)

	case "sync.snapshot":
		var p SnapshotPayload
		if err := json.Unmarshal(ev.Payload, &p); err != nil {
			log.Printf("discord: failed to unmarshal snapshot payload: %v", err)
			return
		}
		b.reconcileChannels(p.Buffers)
	}
}

func (b *DiscordBridge) handleIRCMessage(p MessagePayload) {
	// Only bridge channel messages (DM has its own event type)
	if p.BufferType != "channel" {
		return
	}
	// Sanitize channel name
	rawChanName := strings.TrimPrefix(p.BufferID, "chan:")
	chanName := sanitizeChannelName("🟢", rawChanName)

	chID, err := b.getOrCreateTextChannel(chanName, b.activeCatID)
	if err != nil {
		log.Printf("discord: failed to get/create channel %s: %v", chanName, err)
		return
	}

	content := fmt.Sprintf("**<%s>** %s", p.Nick, p.Text)
	if _, err := b.session.ChannelMessageSend(chID, content); err != nil {
		log.Printf("discord: failed to send message: %v", err)
	}
}

func (b *DiscordBridge) handleIRCDM(p DMPayload) {
	chanName := sanitizeChannelName("👤", p.Peer)

	chID, err := b.getOrCreateTextChannel(chanName, b.activeCatID)
	if err != nil {
		log.Printf("discord: failed to get/create dm channel %s: %v", chanName, err)
		return
	}

	content := fmt.Sprintf("**<%s>** %s", p.Peer, p.Text)
	if _, err := b.session.ChannelMessageSend(chID, content); err != nil {
		log.Printf("discord: failed to send dm message: %v", err)
	}
}

func (b *DiscordBridge) reconcileChannels(activeBuffers []BufferInfo) {
	b.mu.Lock()
	defer b.mu.Unlock()

	channels, err := b.session.GuildChannels(b.guildID)
	if err != nil {
		log.Printf("discord: failed to fetch guild channels for reconcile: %v", err)
		return
	}

	// Build map of expected active channel names
	expectedActive := make(map[string]bool)
	for _, buf := range activeBuffers {
		var name string
		if buf.Type == "channel" {
			name = sanitizeChannelName("🟢", strings.TrimPrefix(buf.ID, "chan:"))
		} else if buf.Type == "dm" {
			name = sanitizeChannelName("👤", strings.TrimPrefix(buf.ID, "dm:"))
		}
		if name != "" {
			expectedActive[name] = true
		}
	}

	for _, c := range channels {
		if c.Type != discordgo.ChannelTypeGuildText {
			continue
		}

		isIRCChannel := strings.HasPrefix(c.Name, "🟢-") || strings.HasPrefix(c.Name, "⚪-") || strings.HasPrefix(c.Name, "👤-")
		if !isIRCChannel {
			continue
		}

		cleanName := c.Name
		var baseName string
		var isDM bool

		if strings.HasPrefix(cleanName, "🟢-") {
			baseName = strings.TrimPrefix(cleanName, "🟢-")
		} else if strings.HasPrefix(cleanName, "⚪-") {
			baseName = strings.TrimPrefix(cleanName, "⚪-")
		} else if strings.HasPrefix(cleanName, "👤-") {
			baseName = strings.TrimPrefix(cleanName, "👤-")
			isDM = true
		}

		expectedActiveName := cleanName
		if !isDM {
			expectedActiveName = "🟢-" + baseName
		}

		if expectedActive[expectedActiveName] {
			// Should be active
			if strings.HasPrefix(c.Name, "⚪-") || c.ParentID != b.activeCatID {
				newName := c.Name
				if !isDM {
					newName = "🟢-" + baseName
				}
				log.Printf("discord: activating channel %s -> %s", c.Name, newName)
				_, _ = b.session.ChannelEditComplex(c.ID, &discordgo.ChannelEdit{
					Name:     newName,
					ParentID: b.activeCatID,
				})
			}
		} else {
			// Should be inactive
			if strings.HasPrefix(c.Name, "🟢-") || c.ParentID != b.inactiveCatID {
				newName := c.Name
				if !isDM {
					newName = "⚪-" + baseName
				}
				log.Printf("discord: deactivating channel %s -> %s", c.Name, newName)
				_, _ = b.session.ChannelEditComplex(c.ID, &discordgo.ChannelEdit{
					Name:     newName,
					ParentID: b.inactiveCatID,
				})
			}
		}
	}
}

func (b *DiscordBridge) getOrCreateCategory(name string) (string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	channels, err := b.session.GuildChannels(b.guildID)
	if err != nil {
		return "", err
	}
	for _, c := range channels {
		if c.Type == discordgo.ChannelTypeGuildCategory && c.Name == name {
			return c.ID, nil
		}
	}

	newChan, err := b.session.GuildChannelCreate(b.guildID, name, discordgo.ChannelTypeGuildCategory)
	if err != nil {
		return "", err
	}
	return newChan.ID, nil
}

func (b *DiscordBridge) getOrCreateTextChannel(name, categoryID string) (string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	channels, err := b.session.GuildChannels(b.guildID)
	if err != nil {
		return "", err
	}

	// Try matching active/inactive states
	baseName := name
	if strings.HasPrefix(name, "🟢-") {
		baseName = strings.TrimPrefix(name, "🟢-")
	} else if strings.HasPrefix(name, "👤-") {
		baseName = strings.TrimPrefix(name, "👤-")
	}

	for _, c := range channels {
		if c.Type != discordgo.ChannelTypeGuildText {
			continue
		}
		if c.Name == name || c.Name == "🟢-"+baseName || c.Name == "⚪-"+baseName {
			// Channel exists, ensure it is in the target category and named correctly
			if c.ParentID != categoryID || c.Name != name {
				_, _ = b.session.ChannelEditComplex(c.ID, &discordgo.ChannelEdit{
					Name:     name,
					ParentID: categoryID,
				})
			}
			return c.ID, nil
		}
	}

	// Create new channel
	newChan, err := b.session.GuildChannelCreate(b.guildID, name, discordgo.ChannelTypeGuildText)
	if err != nil {
		return "", err
	}

	_, _ = b.session.ChannelEditComplex(newChan.ID, &discordgo.ChannelEdit{
		ParentID: categoryID,
	})

	return newChan.ID, nil
}

func (b *DiscordBridge) messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	// Check if message is in our target guild
	if m.GuildID != b.guildID {
		return
	}

	ch, err := s.Channel(m.ChannelID)
	if err != nil {
		return
	}

	// Match channel prefixes to determine targets
	var target string
	if strings.HasPrefix(ch.Name, "🟢-") {
		target = "#" + strings.TrimPrefix(ch.Name, "🟢-")
	} else if strings.HasPrefix(ch.Name, "⚪-") {
		target = "#" + strings.TrimPrefix(ch.Name, "⚪-")
	} else if strings.HasPrefix(ch.Name, "👤-") {
		target = strings.TrimPrefix(ch.Name, "👤-")
	} else {
		return
	}

	// Forward message to IRC command FIFO
	line, err := json.Marshal(map[string]any{
		"action": "send_message",
		"target": target,
		"text":   m.Content,
	})
	if err != nil {
		return
	}

	if err := b.writeFifo(line); err != nil {
		log.Printf("discord: failed to write to command FIFO: %v", err)
	}
}

func (b *DiscordBridge) writeFifo(line []byte) error {
	f, err := os.OpenFile(b.fifoPath, os.O_WRONLY|syscall.O_NONBLOCK, 0600)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	defer f.Close()
	if _, err = f.Write(append(line, '\n')); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	return nil
}

func sanitizeChannelName(prefix, name string) string {
	name = strings.ReplaceAll(name, "#", "")
	name = strings.ToLower(name)
	var cleaned strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			cleaned.WriteRune(r)
		} else if r == '_' || r == ' ' {
			cleaned.WriteRune('-')
		}
	}
	return prefix + "-" + cleaned.String()
}
