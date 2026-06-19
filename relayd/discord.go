package relayd

import (
	"encoding/json"
	"fmt"
	"html"
	"log"
	"net/url"
	"os"
	"regexp"
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
	relayBots     []string // Known relay bots list

	// Cache structures: target <-> channel ID mapping
	cache      map[string]string             // ircTarget -> channelID
	idToTarget map[string]string             // channelID -> ircTarget
	webhooks   map[string]*discordgo.Webhook // channelID -> Webhook
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

	// Load relay bots list
	relayBots := []string{"||"} // Default
	if val := os.Getenv("AWAY_DISCORD_RELAY_BOTS"); val != "" {
		parts := strings.Split(val, ",")
		relayBots = make([]string, 0, len(parts))
		for _, p := range parts {
			trimmed := strings.TrimSpace(p)
			if trimmed != "" {
				relayBots = append(relayBots, trimmed)
			}
		}
	}

	return &DiscordBridge{
		token:      token,
		guildID:    guildID,
		fifoPath:   fifoPath,
		cache:      make(map[string]string),
		idToTarget: make(map[string]string),
		webhooks:   make(map[string]*discordgo.Webhook),
		relayBots:  relayBots,
	}
}

func (b *DiscordBridge) Start() error {
	dg, err := discordgo.New("Bot " + b.token)
	if err != nil {
		return err
	}
	b.session = dg
	dg.AddHandler(b.messageCreate)

	if err = dg.Open(); err != nil {
		return err
	}

	activeCat, err := b.getOrCreateCategory("💬 ACTIVE CHANNELS")
	if err != nil {
		return fmt.Errorf("active category: %w", err)
	}
	b.activeCatID = activeCat

	inactiveCat, err := b.getOrCreateCategory("💤 INACTIVE CHANNELS")
	if err != nil {
		return fmt.Errorf("inactive category: %w", err)
	}
	b.inactiveCatID = inactiveCat

	if err := b.loadChannels(); err != nil {
		return fmt.Errorf("load channels: %w", err)
	}

	log.Println("Discord bot connected and running.")
	return nil
}

func (b *DiscordBridge) Close() {
	if b.session != nil {
		b.session.Close()
	}
}

func (b *DiscordBridge) loadChannels() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	channels, err := b.session.GuildChannels(b.guildID)
	if err != nil {
		return err
	}

	for _, c := range channels {
		if c.Type != discordgo.ChannelTypeGuildText {
			continue
		}
		if strings.HasPrefix(c.Topic, "irc-target:") {
			target := strings.TrimSpace(strings.TrimPrefix(c.Topic, "irc-target:"))
			b.cache[target] = c.ID
			b.idToTarget[c.ID] = target
		}
	}
	return nil
}

func (b *DiscordBridge) HandleEvent(ev Event) {
	switch ev.Type {
	case "message.created":
		var p MessagePayload
		if err := json.Unmarshal(ev.Payload, &p); err == nil {
			b.handleIRCMessage(p)
		}
	case "dm.created":
		var p DMPayload
		if err := json.Unmarshal(ev.Payload, &p); err == nil {
			b.handleIRCDM(p)
		}
	case "sync.snapshot":
		var p SnapshotPayload
		if err := json.Unmarshal(ev.Payload, &p); err == nil {
			b.reconcileChannels(p.Buffers)
		}
	}
}

func (b *DiscordBridge) handleIRCMessage(p MessagePayload) {
	if p.BufferType != "channel" {
		return
	}
	if p.ClientID == "discord" {
		return
	}
	rawChanName := strings.TrimPrefix(p.BufferID, "chan:")
	chID, err := b.getOrCreateTextChannel(rawChanName, b.activeCatID)
	if err != nil {
		log.Printf("discord: get/create channel failed: %v", err)
		return
	}

	cleanedText := CleanHTMLForDiscord(p.Text)
	nick := p.Nick

	if b.isRelayBot(nick) {
		if nestedNick, nestedMsg, ok := parseNestedSender(cleanedText); ok {
			nick = nestedNick + " (via " + nick + ")"
			cleanedText = nestedMsg
		}
	}

	if err := b.sendWebhookMessage(chID, nick, cleanedText); err != nil {
		log.Printf("discord: webhook send failed, falling back to ChannelMessageSend: %v", err)
		content := fmt.Sprintf("**<%s>** %s", nick, cleanedText)
		_, _ = b.session.ChannelMessageSend(chID, content)
	}
}

func (b *DiscordBridge) handleIRCDM(p DMPayload) {
	if p.ClientID == "discord" {
		return
	}
	target := "dm:" + p.Peer
	chID, err := b.getOrCreateTextChannel(target, b.activeCatID)
	if err != nil {
		log.Printf("discord: get/create dm channel failed: %v", err)
		return
	}

	cleanedText := CleanHTMLForDiscord(p.Text)
	nick := p.Peer

	if b.isRelayBot(nick) {
		if nestedNick, nestedMsg, ok := parseNestedSender(cleanedText); ok {
			nick = nestedNick + " (via " + nick + ")"
			cleanedText = nestedMsg
		}
	}

	if err := b.sendWebhookMessage(chID, nick, cleanedText); err != nil {
		log.Printf("discord: webhook send failed, falling back to ChannelMessageSend: %v", err)
		content := fmt.Sprintf("**<%s>** %s", nick, cleanedText)
		_, _ = b.session.ChannelMessageSend(chID, content)
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

func (b *DiscordBridge) getOrCreateTextChannel(target, categoryID string) (string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if id, exists := b.cache[target]; exists {
		return id, nil
	}

	name := sanitizeTargetName(target)
	topic := "irc-target: " + target

	newChan, err := b.session.GuildChannelCreateComplex(b.guildID, discordgo.GuildChannelCreateData{
		Name:     name,
		Type:     discordgo.ChannelTypeGuildText,
		ParentID: categoryID,
		Topic:    topic,
	})
	if err != nil {
		return "", err
	}

	b.cache[target] = newChan.ID
	b.idToTarget[newChan.ID] = target
	return newChan.ID, nil
}

func (b *DiscordBridge) getOrCreateWebhook(channelID string) (*discordgo.Webhook, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if wh, exists := b.webhooks[channelID]; exists {
		return wh, nil
	}

	webhooks, err := b.session.ChannelWebhooks(channelID)
	if err == nil {
		for _, wh := range webhooks {
			if wh.Name == "Away-Bridge" && wh.Token != "" {
				b.webhooks[channelID] = wh
				return wh, nil
			}
		}
	}

	wh, err := b.session.WebhookCreate(channelID, "Away-Bridge", "")
	if err != nil {
		return nil, err
	}

	b.webhooks[channelID] = wh
	return wh, nil
}

func (b *DiscordBridge) sendWebhookMessage(channelID, username, text string) error {
	wh, err := b.getOrCreateWebhook(channelID)
	if err != nil {
		return err
	}

	avatarURL := "https://robohash.org/" + url.PathEscape(username) + ".png"

	_, err = b.session.WebhookExecute(wh.ID, wh.Token, false, &discordgo.WebhookParams{
		Content:   text,
		Username:  username,
		AvatarURL: avatarURL,
	})

	if err != nil {
		if restErr, ok := err.(*discordgo.RESTError); ok && restErr.Message != nil && restErr.Message.Code == discordgo.ErrCodeUnknownWebhook {
			log.Printf("discord: cached webhook not found (404), recreating: %v", err)
			b.mu.Lock()
			delete(b.webhooks, channelID)
			b.mu.Unlock()

			wh, err = b.getOrCreateWebhook(channelID)
			if err != nil {
				return err
			}

			_, err = b.session.WebhookExecute(wh.ID, wh.Token, false, &discordgo.WebhookParams{
				Content:   text,
				Username:  username,
				AvatarURL: avatarURL,
			})
		}
	}

	return err
}

func (b *DiscordBridge) reconcileChannels(activeBuffers []BufferInfo) {
	b.mu.Lock()
	activeMap := make(map[string]bool)
	for _, buf := range activeBuffers {
		target := strings.TrimPrefix(buf.ID, "chan:")
		activeMap[target] = true
	}
	b.mu.Unlock()

	channels, err := b.session.GuildChannels(b.guildID)
	if err != nil {
		log.Printf("discord: reconcile failed to fetch channels: %v", err)
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	for _, c := range channels {
		if c.Type != discordgo.ChannelTypeGuildText {
			continue
		}
		target, ok := b.idToTarget[c.ID]
		if !ok {
			continue
		}

		if activeMap[target] {
			b.activateChannel(c, target)
		} else {
			b.deactivateChannel(c, target)
		}
	}
}

func (b *DiscordBridge) activateChannel(c *discordgo.Channel, target string) {
	cleanName := sanitizeTargetName(target)

	if strings.HasPrefix(c.Name, "⚪-") || c.ParentID != b.activeCatID {
		log.Printf("discord: activating channel %s -> %s", c.Name, cleanName)
		_, _ = b.session.ChannelEditComplex(c.ID, &discordgo.ChannelEdit{
			Name:     cleanName,
			ParentID: b.activeCatID,
		})
	}
}

func (b *DiscordBridge) deactivateChannel(c *discordgo.Channel, target string) {
	isDM := strings.HasPrefix(target, "dm:")
	if isDM {
		if c.ParentID != b.inactiveCatID {
			log.Printf("discord: deactivating dm channel %s", c.Name)
			_, _ = b.session.ChannelEditComplex(c.ID, &discordgo.ChannelEdit{
				ParentID: b.inactiveCatID,
			})
		}
		return
	}

	baseName := strings.TrimPrefix(sanitizeTargetName(target), "🟢-")
	inactiveName := "⚪-" + baseName
	if c.Name != inactiveName || c.ParentID != b.inactiveCatID {
		log.Printf("discord: deactivating channel %s -> %s", c.Name, inactiveName)
		_, _ = b.session.ChannelEditComplex(c.ID, &discordgo.ChannelEdit{
			Name:     inactiveName,
			ParentID: b.inactiveCatID,
		})
	}
}

func (b *DiscordBridge) messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID || m.GuildID != b.guildID {
		return
	}

	b.mu.Lock()
	target, exists := b.idToTarget[m.ChannelID]
	b.mu.Unlock()
	if !exists {
		return
	}

	ircTarget := target
	if strings.HasPrefix(target, "dm:") {
		ircTarget = strings.TrimPrefix(target, "dm:")
	}

	line, err := json.Marshal(map[string]any{
		"action":    "send_message",
		"target":    ircTarget,
		"text":      m.Content,
		"client_id": "discord",
	})
	if err != nil {
		return
	}

	if err := b.writeFifo(line); err != nil {
		log.Printf("discord: command FIFO write failed: %v", err)
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

func sanitizeTargetName(target string) string {
	if strings.HasPrefix(target, "dm:") {
		peer := strings.TrimPrefix(target, "dm:")
		return "👤-" + cleanDiscordName(peer)
	}
	return "🟢-" + cleanDiscordName(target)
}

func cleanDiscordName(name string) string {
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
	return cleaned.String()
}

var spanRegex = regexp.MustCompile(`</?span[^>]*>`)

func CleanHTMLForDiscord(s string) string {
	s = strings.ReplaceAll(s, "<b>", "**")
	s = strings.ReplaceAll(s, "</b>", "**")
	s = strings.ReplaceAll(s, "<i>", "*")
	s = strings.ReplaceAll(s, "</i>", "*")
	s = strings.ReplaceAll(s, "<u>", "__")
	s = strings.ReplaceAll(s, "</u>", "__")

	s = spanRegex.ReplaceAllString(s, "")

	s = html.UnescapeString(s)
	return s
}

func (b *DiscordBridge) isRelayBot(nick string) bool {
	for _, bot := range b.relayBots {
		if bot == nick {
			return true
		}
	}
	return false
}

var nestedPatterns = []*regexp.Regexp{
	regexp.MustCompile(`^<([^>]+)>\s*(.*)$`),
	regexp.MustCompile(`^\[([^\]]+)\]\s*(.*)$`),
	regexp.MustCompile(`^\(([^)]+)\)\s*(.*)$`),
	regexp.MustCompile(`^([a-zA-Z0-9_\-\[\]\\^` + "`" + `{|}\x7f]+):\s*(.*)$`),
}

func parseNestedSender(text string) (string, string, bool) {
	text = strings.TrimSpace(text)
	for _, re := range nestedPatterns {
		matches := re.FindStringSubmatch(text)
		if len(matches) == 3 {
			nick := strings.TrimSpace(matches[1])
			msg := strings.TrimSpace(matches[2])
			if nick != "" && msg != "" {
				return nick, msg, true
			}
		}
	}
	return "", "", false
}
