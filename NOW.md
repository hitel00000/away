# NOW.md

## 🎯 Current Goal

Pivot the client interface from custom PWA to a private Discord Server (Discord Bot Pivot).

---

## 🔥 Top Priority (Do in order)

1. I-001 Discord Bot Initialization

   - add `discordgo` dependency to Go relayd
   - implement Discord client connection using token from env
   - verify connection with basic logging/ping

---

2. I-002 Ingestion to Discord Bridge

   - relay `message.created` events from irssi to matching Discord channels
   - dynamically create channels (`🟢-채널명`, `👤-닉네임`) under `💬 ACTIVE CHANNELS` category

---

3. I-003 Discord to IRC Egress Bridge

   - listen to user messages in text channels
   - send message contents back to IRC unix socket for `/msg`

---

4. I-004 Presence & Join/Part Sync

   - update channel names (e.g. `🟢-` to `⚪-`) and categories (`💬 ACTIVE` to `💤 INACTIVE`) on Join/Part events

---

## 🧪 Mandatory Test Loop

1. Run irssi and relayd
2. Send message from IRC, check if bot creates channel and sends message to Discord
3. Send message from Discord, check if message is relayed back to IRC
4. Join/Part channel on IRC, check if emoji prefix and category updates on Discord

---

## ❌ Do NOT Work On

- Custom PWA client (`web/` updates)
- Custom session auth or WebSocket snapshot logic
- UI polish or search in web client