# Away

> A Discord-based companion for your irssi workflow.

Away is not an IRC client.

It is a **personal remote interface** to your existing irssi session, using Discord as the client interface.

---

# ✨ What This Is

- A Discord Bot relay that lets you read and reply to IRC from your phone/desktop using the official Discord app
- A relay that bridges irssi events to Discord text channels
- A system designed for **casual, intermittent interaction**

---

# ❌ What This Is NOT

- Not a full IRC client
- Not a bouncer replacement
- Not a hosted multi-user system
- Not trying to replace irssi

irssi remains the source of truth.

---

# 🧱 Architecture

```
Discord App (mobile/desktop)
↓ HTTPS / WebSocket
Discord Gateway / API
↓
Relay (Go)
↓ Unix socket
irssi plugin (Perl)
↓
irssi
```

---

# 🧩 Responsibilities

## irssi plugin
- Emits events only
- Does not manage state

## relay (relayd)
- Runs as a Discord Bot
- Bridges events from irssi to private Discord guild channels
- Bridges user messages from Discord back to irssi

---

# 🛠 Configuration

Run `relayd` with the following environment variables:
- `AWAY_DISCORD_TOKEN`: Discord Bot Token
- `AWAY_DISCORD_GUILD_ID`: Target private Discord Server (Guild) ID
- `AWAY_IRC_SOCKET`: UNIX socket path for irssi events (default: `/tmp/away/irc-companion.sock`)
- `AWAY_IRC_FIFO`: CMD FIFO path for sending messages back to irssi (default: `/tmp/away/irc-companion.cmd`)