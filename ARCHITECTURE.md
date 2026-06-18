# ARCHITECTURE

Away system architecture and state model (Discord-based).

---

# 1. Overview

Away is a relay-based companion system for irssi that bridges IRC communication to Discord.

It separates:

- Event Production: irssi (via Perl plugin)
- Bridging & Event Translation: Go Relay (`relayd`) running as a Discord Bot
- Interaction Surface: Official Discord Client (Mobile/Desktop/Web)

---

# 2. System Diagram

```
Discord App (Client)
↓ HTTPS / WebSocket (Gateway)
Discord API
↓
Relay (relayd / Go)
↓ Unix socket
irssi plugin (away_bridge.pl / Perl)
↓
irssi (IRC client / Source of Truth)
```

---

# 3. Core Responsibilities

## irssi
- Maintains actual IRC sessions and canonical connections.

## irssi plugin (away_bridge.pl)
- Emits structured events (JSON) over a Unix socket.
- Receives send commands from `relayd` via Unix FIFO.
- Emits: `message.created`, `dm.created`, `sync.snapshot`.

## Relay (relayd)
- Runs as a Discord Bot in a dedicated, private Discord Server (Guild).
- Ingests events from irssi plugin:
  - Dynamically creates/manages Discord channels in a private Guild.
  - Maps active channels under `💬 ACTIVE CHANNELS` category.
  - Relays IRC public/private messages to corresponding Discord channels.
- Egresses messages from Discord:
  - Listens to user messages in text channels within the target Guild.
  - Forwards user messages back to the irssi command FIFO.

## Client (Discord App)
- Renders the channels and messages.
- Handles user connection state, notifications, push alerts, and typing indicators natively.

---

# 4. Channel Mapping & Lifecycle

Away maps IRC buffers to Discord channels:

- **IRC Channels**: Mapped to `#🟢-channelname` when active, or `#⚪-channelname` when inactive/parted.
- **IRC DMs / Queries**: Mapped to `#👤-nickname` under the active category.
- **Categories**:
  - `💬 ACTIVE CHANNELS`: Channels currently joined.
  - `💤 INACTIVE CHANNELS`: Channels currently parted or inactive queries.

State changes (Join/Part) trigger automatic channel renaming and category migration by the bot.

---

# 5. Non-Goals

- Multi-tenant architecture (designed strictly for one user in their private Discord server).
- Replacing irssi configuration or authentication.
- Implementing a full Discord integration beyond bridging text chat and status.