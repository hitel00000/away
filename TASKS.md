# TASKS.md

Away post-MVP backlog (Phase 0.2+)

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

# MVP Status

Completed:

- [x] Milestone A — Walking Skeleton
- [x] Milestone B — Minimal Client
- [x] Milestone C — irssi Bridge
- [x] Milestone D — Resilience
- [x] Milestone E — MVP Hardening

MVP is considered achieved.

---

# Phase 0.2 — Usability Pass

Goal:

Move from “works” to “pleasant daily companion”.

---

## F-001 Buffer / Unread Correctness
Status: [ ]

Implement:
- verify unread counts stay correct
- active buffer switching correctness
- basic buffer list polish

Acceptance:
- channel and DM switching behaves reliably
- unread counters stay consistent

Priority:
High

Depends on:
Milestone E

---

## F-002 Mention Inbox MVP
Status: [ ]

Implement:
- synthetic mentions buffer
- collect highlight events into inbox
- basic read/clear handling

Minimal only.

Do not add ranking, notifications, or triage logic.

Acceptance:
- mentions appear in dedicated inbox
- mention buffer usable from browser

Priority:
High

Depends on:
F-001

---

## F-003 Send Acknowledgement
Status: [ ]

Implement:
- pending -> sent UI state
- basic optimistic feedback

Acceptance:
- user can distinguish sent vs pending

Priority:
Medium

Depends on:
F-001

---

## F-004 Presence Noise Collapse
Status: [ ]

Implement:
- collapse join/part noise
- optionally hide low-value presence spam

Acceptance:
- busy channels remain readable

Priority:
Medium

Depends on:
F-001

---

# Phase 0.3 — Lightweight Persistence

Goal:

Survive restarts with minimal added complexity.

---

## G-001 Append-Only Event Journal Spike
Status: [ ]

Very small spike only.

Implement:
- append-only local event log
- restore recent messages on restart

Do NOT introduce sqlite yet.

Acceptance:
- relay restart preserves recent history

Priority:
Medium

Depends on:
F-002

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

# Deferred (Not Now)

Explicit non-tasks:

- WebAuthn pairing
- push notifications
- sqlite database
- full mobile polish pass
- multi-device sync
- AI summarization
- hosted multi-user support

Do not start these.

---

# Agent Task Execution Rules

For each task:

1. Implement only this task.
2. Minimal patch.
3. Add tests.
4. Do not refactor unrelated code.
5. No speculative abstractions.
6. If scope grows, stop.

---

# Current Recommended Sequence

1. F-001
2. F-002
3. F-003

Reassess after these.