# TASKS.md

Away post-MVP backlog (Phase 0.3+)

Status legend

- [ ] todo
- [~] in progress
- [x] done
- [blocked]

Rule:

Work top-to-bottom unless explicitly reprioritized.
Favor product value over infrastructure.
Do not expand scope.

---

# Completed

## MVP
- [x] Milestone A — Walking Skeleton
- [x] Milestone B — Minimal Client
- [x] Milestone C — irssi Bridge
- [x] Milestone D — Resilience
- [x] Milestone E — MVP Hardening

## Phase 0.2 — Usability Pass
- [x] F-001 Buffer / Unread Correctness
- [x] F-002 Mention Inbox MVP
- [x] F-003 Send Acknowledgement
- [x] F-004 Presence Noise Collapse

Phase 0.2 complete.

---

# Phase 0.3 — Lightweight Persistence

## G-001 Append-Only Event Journal Spike
Status: [x]

---

## G-002 Search Spike (Optional)
Status: [ ]

Very small experiment only.

Acceptance:
- decide whether local search belongs in scope

Priority:
Low

Depends on:
G-001

---

# Phase 0.4 — Real Usage Loop

Goal:

Validate that the system is actually usable in daily IRC workflow.

---

## H-001 Use In Real Conversations
Status: [x]

## H-002 Identify Top 3 Frictions
Status: [x]

## H-003 Targeted Fixes Only
Status: [x]

## H-004 Stability & Lifecycle Fixes
Status: [x]

---

## H-005 Snapshot Ownership Fix
Status: [ ]

Goal:

Move snapshot responsibility from irssi plugin to relay.

Reason:

Current implementation treats snapshot as an event,
causing ordering issues, replay inconsistency,
and state corruption.

---

Tasks:

- [ ] remove `sync.snapshot` emission from irssi plugin

- [ ] ensure plugin only emits:
      - message.created
      - highlight.created
      - presence.*

- [ ] implement initial buffer dump from plugin
      (event-based, one-time seed only)

- [ ] implement `buildSnapshot()` in relay
      using in-memory or persisted state

- [ ] send snapshot only during websocket init

- [ ] ensure snapshot is NOT stored in event journal

- [ ] ensure snapshot is NOT replayed

- [ ] client: treat snapshot as full state replace
      (not reducer-based merge)

---

Acceptance:

- snapshot never appears in journal
- snapshot always arrives before first live event
- reconnect produces consistent buffer list
- no duplicate / missing buffers after reload

---

## H-006 Reconnect & Resume Correctness
Status: [ ]

Goal:

Ensure reconnect behavior is deterministic and safe.

---

Tasks:

- [ ] implement `resume_from` handling in relay

- [ ] replay events strictly AFTER given event_id

- [ ] fallback to snapshot if replay not possible

- [ ] guarantee no duplicate events on reconnect

- [ ] ensure ordering: snapshot → replay → live

---

Acceptance:

- reconnect never causes message duplication
- reconnect never loses messages
- reconnect preserves scroll position semantics

---

## H-007 Event Stream Correctness
Status: [ ]

Goal:

Ensure event stream behaves predictably under all conditions.

---

Tasks:

- [ ] dedupe events by event_id

- [ ] enforce ordering (timestamp vs arrival consistency)

- [ ] verify no out-of-order application in client

- [ ] ensure snapshot boundary is respected

---

Acceptance:

- no duplicate messages visible
- no ordering glitches during heavy traffic
- consistent state after long sessions

---

## H-008 Minimal Auth (Session Layer)
Status: [ ]

Goal:

Introduce minimal session identity without full auth complexity.

---

Tasks:

- [ ] introduce temporary device_id

- [ ] bind websocket session to device_id

- [ ] basic handshake validation

- [ ] prepare structure for future trusted devices

---

Acceptance:

- multiple clients do not conflict
- session identity is stable across reconnect

---

# Deferred (Not Now)

- WebAuthn pairing
- push notifications
- sqlite full-text search
- multi-device sync conflict resolution
- advanced read tracking
- message threading
- hosted multi-user support
- AI summarization

Do not start these until after Phase 0.4 stabilizes.

---

# Agent Task Execution Rules

For each task:

1. Implement only this task.
2. Minimal patch.
3. Add tests where possible.
4. Do not refactor unrelated code.
5. No speculative abstractions.
6. If scope grows, stop.
