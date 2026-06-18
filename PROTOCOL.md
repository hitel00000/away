# PROTOCOL

Away internal wire protocol and formats (irssi ↔ relayd).

---

# 1. Transport

All communications between the irssi Perl plugin and the Go relay backend (`relayd`) occur over local UNIX domain sockets and FIFOs using Newline-Delimited JSON (NDJSON).

- **Ingress (IRC to Relay)**: Stream socket at `/tmp/away/irc-companion.sock`
- **Egress (Relay to IRC)**: Named pipe (FIFO) at `/tmp/away/irc-companion.cmd`

---

# 2. Ingress Events (Perl Plugin ➔ Go Relay)

Every event sent from the Perl plugin over `/tmp/away/irc-companion.sock` has the following envelope:

```json
{
  "type": "event_type",
  "version": 1,
  "id": "evt_timestamp_random",
  "timestamp": "ISO8601_Timestamp",
  "payload": {}
}
```

### 2.1 `message.created` (Public Channel Messages)

Emitted when a new message is received or sent in a channel.

```json
{
  "type": "message.created",
  "version": 1,
  "id": "evt_1718712345678_9999",
  "timestamp": "2026-06-18T08:15:00Z",
  "payload": {
    "network": "libera",
    "buffer_id": "chan:#away",
    "buffer_type": "channel",
    "nick": "alice",
    "text": "hello world",
    "highlight": false,
    "tags": [],
    "client_id": ""
  }
}
```

### 2.2 `dm.created` (Private Query Messages)

Emitted when a private message (query) is received or sent.

```json
{
  "type": "dm.created",
  "version": 1,
  "id": "evt_1718712345679_8888",
  "timestamp": "2026-06-18T08:15:01Z",
  "payload": {
    "network": "libera",
    "peer": "bob",
    "text": "are you there?",
    "client_id": ""
  }
}
```

### 2.3 `sync.snapshot` (Active Channel List)

Emitted periodically or on join/part events to sync the list of active/joined buffers.

```json
{
  "type": "sync.snapshot",
  "version": 1,
  "id": "evt_1718712345680_7777",
  "timestamp": "2026-06-18T08:15:02Z",
  "payload": {
    "buffers": [
      {"id": "chan:#away", "type": "channel", "label": "#away"},
      {"id": "chan:#go", "type": "channel", "label": "#go"},
      {"id": "dm:bob", "type": "dm", "label": "bob"}
    ]
  }
}
```

---

# 3. Egress Commands (Go Relay ➔ Perl Plugin)

Commands are written as single-line JSON strings to the FIFO pipe at `/tmp/away/irc-companion.cmd`.

### 3.1 `send_message`

Instructs irssi to send a message to a specific channel or nick.

```json
{
  "action": "send_message",
  "target": "#away",
  "text": "response from discord"
}
```

### 3.2 `mark_read`

Instructs irssi to clear unread highlighting or status for a channel.

```json
{
  "action": "mark_read",
  "target": "#away"
}
```