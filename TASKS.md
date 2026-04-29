# TASKS.md

Away MVP implementation backlog.

Status legend:

- [ ] todo
- [~] in progress
- [x] done
- [blocked]

Rule:

Work top-to-bottom unless explicitly reprioritized.

Do not expand scope.

---

# Milestone A — Walking Skeleton

Goal:

phone receives IRC events.

---

## A-001 Repository Bootstrap
Status: [x]

Deliverables:

- monorepo structure
- Makefile works
- relay placeholder boots

Acceptance:

- `make run-relay` succeeds

---

## A-002 Protocol Event Envelope
Status: [x]

Implement:

- Event struct
- `message.created`
- fixture NDJSON samples

Files:

- schemas/
- fixtures/

Acceptance:

- golden fixture test passes

Depends on:
none

---

## A-003 In-Memory Ring Buffer
Status: [x]

Implement:

- append event
- replay recent events
- capacity 500

Do NOT add persistence.

Acceptance:

- unit tests pass

Depends on:
A-002

---

## A-004 Websocket Broadcast Hub
Status: [x]

Implement:

- client connect
- broadcast
- disconnect cleanup

Acceptance:

two clients receive same event

Depends on:
A-003

---

## A-005 Fixture Replay Feed
Status: [x]

Implement:

make dev-feed

Replay sample events into relay.

Acceptance:

browser receives fixture events.

Depends on:
A-004

---

# Milestone B — Minimal Client

Goal:

read messages from browser.

---

## B-001 Message Feed UI
Status: [x]

Implement:

- websocket connect
- render incoming events

No styling work.

Acceptance:

messages visible in browser.

Depends on:
A-005

---

## B-002 Send Message Input
Status: [x]

Implement:

- text box
- send_message command

No slash command support.

Acceptance:

command reaches relay.

Depends on:
B-001

---

# Milestone C — irssi Bridge

Goal:

end-to-end real message flow.

---

## C-001 Public/Private Signal Hooks
Status: [x]

Implement:

- message public
- message private

Emit NDJSON.

Acceptance:

irssi event reaches relay.

Depends on:
A-002

---

## C-002 Relay → Plugin Send Path
Status: [x]

Implement:

send_message command bridge

Acceptance:

browser reply reaches irssi.

Depends on:
C-001
B-002

---

# Milestone D — Resilience

Goal:

survive reconnects.

---

## D-001 Plugin Socket Reconnect
Status: [x]

Implement:

- reconnect logic
- small outbound queue

Acceptance:

relay restart does not require irssi restart

Depends on:
C-001

---

## D-002 Browser Replay on Reconnect
Status: [x]

Implement:

recent replay on reconnect.

Acceptance:

refresh restores recent events.

Depends on:
A-003
B-001

---

## D-003a Outbound Correlation ID Plumbing
Status: [x]

Implement:
- client_id pass-through
- own echo includes client_id

Acceptance:
identical client_id observed end-to-end

Do not:
- dedupe logic
- text matching
- UI work

---

## D-003b Pending Send Reconciliation
Status: [ ]

Depends on:
D-003a

Implement:
- optimistic pending bubble
- reconcile on matching client_id
- prevent duplicate render

Acceptance:
sent message appears once

---

## D-003c Pending Timeout Cleanup
Status: [ ]

Depends on:
D-003b

Implement:
- failed pending expiry

Acceptance:
stuck pending clears safely

---

# Current Priority Queue

1.
A-002 Protocol Event Envelope

2.
A-003 Ring Buffer

3.
A-004 Websocket Hub

4.
A-005 Fixture Replay

5.
B-001 Message Feed UI

Do these first.

---

# Agent Task Execution Rules

For each task:

1. Implement only this task.
2. Minimal patch.
3. Add tests.
4. Do not refactor unrelated code.
5. Do not add new abstractions unless necessary.
6. For deduplication tasks, do not use text-only matching unless explicitly required.
   Prefer explicit identifiers and reconciliation.

---

# Explicit Non-Tasks (Do Not Start)

Not part of MVP:

- authentication
- push notifications
- sqlite persistence
- search
- Telegram bridge
- AI summarization
- advanced inbox logic

If you start these,
you are off scope.

---

# Nice-to-Have After MVP

Only after all above complete:

- sqlite storage
- mention inbox
- push
- device trust auth

Ignore until MVP done.