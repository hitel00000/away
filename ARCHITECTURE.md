# ARCHITECTURE

Away system architecture and state model.

---

# 1. Overview

Away is a relay-based companion system for irssi.

It separates:

- event production (irssi)
- state materialization (relay)
- interaction surface (client)

---

# 2. System Diagram

```id="arch-diagram"

PWA Client
↓ WebSocket
Relay (state authority)
↓ Unix socket
irssi plugin (event emitter)
↓
irssi (source of truth)

```

---

# 3. Core Responsibilities

## irssi

- maintains real IRC session
- holds canonical live connection state

---

## irssi plugin

- emits structured events
- does NOT store state
- does NOT build snapshots

Allowed outputs:

- message.created
- dm.created
- highlight.created
- presence.*

---

## relay

The most important component.

Responsibilities:

- maintains materialized state
- stores event journal (append-only)
- builds snapshots
- handles reconnect and replay
- mediates commands

---

## client (PWA)

- renders state
- sends commands
- holds ephemeral UI state

---

# 4. State Model

Away uses a hybrid model:

- event sourcing (for changes)
- materialized state (for usability)

---

## 4.1 Events

Events represent changes:

- immutable
- ordered
- replayable

Stored in journal.

---

## 4.2 Snapshot

Snapshot represents current state.

It is:

- derived from relay state
- NOT part of event stream
- NOT stored in journal

---

## ⚠️ Critical Rule

Snapshot is NOT an event.

Mixing snapshot into event stream causes:

- ordering corruption
- replay inconsistency
- state duplication

---

# 5. State Flow

## 5.1 Initial Connection

```id="state-init"

client connects
→ relay builds snapshot
→ relay sends snapshot
→ relay starts event stream

```

---

## 5.2 Reconnect

```id="state-reconnect"

client reconnects with resume_from
→ relay attempts replay
→ if replay fails:
send snapshot
→ continue streaming

```

---

## 5.3 Event Processing

```id="state-events"

plugin emits event
→ relay appends to journal
→ relay updates state
→ relay pushes to clients

```

---

# 6. Buffer Model

Buffers are primary units of interaction.

Examples:

```id="buffer-ids"

chan:#golang
pm:alice
system:mentions

```

Relay maintains:

- buffer list
- unread counts
- mention counts
- last activity

---

# 7. Source of Truth

There are two layers:

## irssi

- source of live IRC state

## relay

- source of application state

These are NOT identical.

---

# 8. Initial State Seeding

Relay cannot reconstruct state from nothing.

Therefore plugin must provide:

- initial buffer list (event-based)

Example:

```

buffer.opened (one per buffer)

```

This is NOT a snapshot.

It is a seed.

---

# 9. Persistence

Event journal:

- append-only
- used for replay

Materialized state:

- in-memory (primary)
- optional persistence (SQLite)

---

# 10. Failure Model

System must tolerate:

- network disconnects
- relay restarts
- client reloads

Guarantees:

- no duplicated events
- no missing buffers
- consistent unread state

---

# 11. Design Constraints

- single-user system
- trusted environment
- minimal auth (for now)
- correctness over features

---

# 12. Non-Goals

- multi-tenant architecture
- full IRC client replacement
- distributed consensus
- perfect event sourcing fidelity

---

# 13. Key Invariants

These must always hold:

- snapshot is never in journal
- events are never mutated
- event_id is unique
- replay is deterministic
- snapshot fully replaces client state

---

# 14. Mental Model

Think of the system as:

- irssi → produces signals
- relay → builds reality
- client → views reality

If these roles blur, bugs appear.