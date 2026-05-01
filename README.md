# Away

> A mobile-first companion for your irssi workflow.

Away is not an IRC client.

It is a **personal remote interface** to your existing irssi session.

---

# ✨ What This Is

- A PWA that lets you read and reply to IRC from your phone
- A relay that bridges irssi to a modern event-driven interface
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

PWA (mobile)
↓ WebSocket
Relay (Go)
↓ Unix socket
irssi plugin (Perl)
↓
irssi

```

---

# 🧠 Core Model

Away is built on two distinct concepts:

## 1. Events

Everything that *happens*:

- message.created
- highlight.created
- presence.*

Events are:

- append-only
- ordered
- replayable

---

## 2. Snapshot

A snapshot is:

> the current materialized state of the system

It includes:

- buffers
- unread counts
- active context

---

## ⚠️ Important Rule

**Snapshot is NOT an event.**

- It is NOT stored in the event journal
- It is NOT replayed
- It is sent only during connection initialization

---

# 🔄 Connection Flow

## Initial connect

```

client connects
→ relay sends snapshot
→ relay streams live events

```

---

## Reconnect

```

client sends resume_from
→ relay replays events (if possible)
→ otherwise sends snapshot
→ then resumes live stream

```

---

# 🧩 Responsibilities

## irssi plugin

- emits events only
- does NOT generate snapshots
- does NOT manage state

---

## relay

- owns system state
- stores event journal
- builds snapshots
- handles reconnect and replay

---

## client

- initializes state from snapshot
- updates state via events
- treats snapshot as full replace

---

# 🗂 Event Journal

- append-only
- contains only events
- used for replay on reconnect

Snapshot is never included.

---

# 🧪 Current Status

Phase: 0.4 — Real Usage Loop

Focus:

- correctness under reconnect
- stable state synchronization
- minimal friction in daily use

Not focused on:

- UI polish
- advanced features
- multi-device sync (yet)

---

# 🚧 Known Constraints

- single-user only
- assumes trusted environment
- minimal auth (for now)
- state consistency prioritized over features

---

# 🛣 Roadmap (Short-Term)

- stabilize snapshot + event boundary
- reliable reconnect behavior
- event deduplication
- minimal session identity

---

# 🛠 Development

## Components

```

irssi-plugin/
relayd/
pwa/

```

---

## Run (high-level)

1. run irssi with plugin
2. start relay
3. open PWA

---

# 🧠 Design Philosophy

> Build the smallest system that makes irssi feel persistent on mobile.

Key principles:

- irssi is the source of truth
- relay owns state materialization
- events drive everything
- snapshot is a boundary, not a stream

---

# ⚠️ Common Pitfalls

If you see these, something is wrong:

- snapshot appears in event journal
- duplicate messages after reconnect
- missing buffers after reload
- unread count resets incorrectly

---

# 📌 Summary

- events describe change
- snapshot defines state
- relay owns state
- plugin emits signals

Keep those boundaries clear, and the system stays simple.