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

Goal:

Survive restarts with minimal complexity.

---

## G-001 Append-Only Event Journal Spike
Status: [x]

Very small spike only.

Implement:
- append-only local event log
- restore recent messages on restart
- bounded recent history only

Do NOT introduce:
- sqlite
- indexing layer
- durable sync model

Acceptance:
- relay restart preserves recent context
- implementation remains simple
- complexity stays proportional

Priority:
High

Depends on:
Phase 0.2

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
- message threading
- notification routing

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

1. Use current build in real conversations
2. G-001
3. Reassess

Stop and reconsider after G-001.