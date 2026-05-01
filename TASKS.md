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

Use the system in real IRC sessions.

Focus:

- passive reading
- occasional replies
- reconnect scenarios

Record:

- friction points
- missing primitives
- surprising behavior

Do NOT fix immediately unless critical.

---

## H-002 Identify Top 3 Frictions
Status: [x]

From real usage, identify:

- top 3 UX issues
- top 3 reliability issues

Write them down explicitly.

No implementation in this step.

---

## H-003 Targeted Fixes Only
Status: [x]

Implement only:

- show joined buffers from snapshot
- fix duplicate buffer entries on reload
- minimal mark_read (reset unread)
- minimal mobile layout (list toggle + full-width chat)
- basic UI cleanup (spacing, typography only)

Do NOT:

- redesign UI
- introduce new state models
- add advanced read tracking

---

## H-004 — Stability & Lifecycle Fixes

Goal:

Close gaps discovered during real usage.
Focus on state consistency and basic lifecycle.

No new features.

---

- [x] fix mark_read not surviving reload
      (reconcile unread with snapshot)

- [x] fix initial landing state
      (no default #test, show neutral empty state)

- [x] implement WebSocket reconnect
      (simple retry with backoff)

- [x] fix mobile scroll behavior
      - message list is only scroll container
      - auto-stick to bottom when appropriate
      - input stays fixed at bottom

- [x] verify snapshot-driven buffer rendering remains correct

---

Exit condition:

- unread does not reappear after reload
- first screen is not misleading
- reconnect works reliably
- mobile chat feels natural to use

---

# Deferred (Not Now)

- G-002 Search Spike
- WebAuthn pairing
- push notifications
- sqlite database
- full mobile polish pass
- multi-device sync
- AI summarization
- hosted multi-user support
- message threading
- notification routing

Do not start these until after Phase 0.4.

---

# Agent Task Execution Rules

For each task:

1. Implement only this task.
2. Minimal patch.
3. Add tests.
4. Do not refactor unrelated code.
5. No speculative abstractions.
6. If scope grows, stop.
