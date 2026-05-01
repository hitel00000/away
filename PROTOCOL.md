# PROTOCOL

Away wire protocol and data formats.

---

# 1. Transport

- WebSocket (/ws)
- JSON messages
- newline-delimited on plugin side (NDJSON)

---

# 2. Envelope Types

## Server → Client

```json id="proto-event"
{
  "kind": "event",
  "body": {}
}
````

---

## Client → Server

```json id="proto-command"
{
  "kind": "command",
  "body": {}
}
```

---

## System

```json id="proto-heartbeat"
{
  "kind": "heartbeat"
}
```

---

# 3. Events

All events follow:

```json id="proto-event-envelope"
{
  "type": "message.created",
  "version": 1,
  "id": "evt_123",
  "timestamp": "ISO8601",
  "payload": {}
}
```

---

## Properties

* id: globally unique
* timestamp: event time
* payload: type-specific

---

# 4. Core Event Types

## message.created

```json id="proto-msg"
{
  "type": "message.created",
  "payload": {
    "network": "libera",
    "buffer_id": "chan:#golang",
    "nick": "alice",
    "text": "hello"
  }
}
```

---

## dm.created

```json id="proto-dm"
{
  "type": "dm.created",
  "payload": {
    "peer": "bob",
    "text": "ping"
  }
}
```

---

## highlight.created

```json id="proto-highlight"
{
  "type": "highlight.created",
  "payload": {
    "buffer_id": "...",
    "text": "...",
    "priority": "normal"
  }
}
```

---

## presence.*

* presence.join
* presence.part
* presence.quit
* nick.changed

---

## buffer.updated

```json id="proto-buffer"
{
  "type": "buffer.updated",
  "payload": {
    "buffer_id": "...",
    "unread": 10,
    "mentions": 1
  }
}
```

---

# 5. Snapshot (Special Case)

Snapshot uses event envelope but is NOT an event.

```json id="proto-snapshot"
{
  "type": "sync.snapshot",
  "payload": {
    "buffers": [...],
    "active_buffer": "..."
  }
}
```

---

## ⚠️ Rules

* NOT stored in journal
* NOT replayed
* ONLY sent during init or fallback

---

# 6. Commands

## send_message

```json id="proto-send"
{
  "action": "send_message",
  "payload": {
    "target": "#foo",
    "text": "hello"
  }
}
```

---

## irc_command

```json id="proto-irc"
{
  "action": "irc_command",
  "payload": {
    "command": "/whois alice"
  }
}
```

---

## mark_read

```json id="proto-read"
{
  "action": "mark_read",
  "payload": {
    "buffer_id": "chan:#foo"
  }
}
```

---

## fetch_backlog

```json id="proto-backlog"
{
  "action": "fetch_backlog",
  "payload": {
    "buffer_id": "...",
    "before": "evt_123",
    "limit": 100
  }
}
```

---

# 7. Reconnect Protocol

Client may send:

```json id="proto-resume"
{
  "resume_from": "evt_9211"
}
```

---

## Server behavior

1. if possible:
   → replay events after id

2. if not:
   → send snapshot

3. then:
   → continue live stream

---

# 8. Ordering Rules

* events must be applied in order
* snapshot resets ordering context
* no event before snapshot on init

---

# 9. Idempotency

Client must:

* ignore duplicate event_id
* handle replay safely

---

# 10. Error Handling

Future:

```json id="proto-error"
{
  "type": "system.error",
  "payload": {
    "message": "..."
  }
}
```

---

# 11. Versioning

* version field per event
* no silent schema changes
* backward compatibility preferred

---

# 12. Invariants

* event_id uniqueness
* snapshot boundary respected
* no mixed snapshot/event replay
* deterministic reconnect

---

# 13. Philosophy

Protocol favors:

* simplicity over completeness
* explicit boundaries over magic
* correctness over convenience