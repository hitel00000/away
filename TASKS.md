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
Status: [deferred] (due to Discord Pivot)

---

## H-006 Reconnect & Resume Correctness
Status: [deferred] (due to Discord Pivot)

---

## H-007 Event Stream Correctness
Status: [deferred] (due to Discord Pivot)

---

## H-008 Minimal Auth (Session Layer)
Status: [deferred] (due to Discord Pivot)

---

# Phase 0.5 — Discord Pivot

Goal:

Pivot the client interface from custom PWA to a private Discord Server.

---

## I-001 Discord Bot Initialization
Status: [x]

Tasks:
- [x] add `discordgo` dependency to Go relayd
- [x] implement Discord client connection using token from env
- [x] verify connection with basic logging/ping

---

## I-002 Ingestion to Discord Bridge
Status: [x]

Tasks:
- [x] relay `message.created` events from irssi to matching Discord channels
- [x] dynamically create channels (`🟢-채널명`, `👤-닉네임`) under `💬 ACTIVE CHANNELS` category

---

## I-003 Discord to IRC Egress Bridge
Status: [x]

Tasks:
- [x] listen to user messages in text channels
- [x] send message contents back to IRC unix socket for `/msg`

---

## I-004 Presence & Join/Part Sync
Status: [x]

Tasks:
- [x] update channel names (e.g. `🟢-` to `⚪-`) and categories (`💬 ACTIVE` to `💤 INACTIVE`) on Join/Part events

---

## I-005 Spoof Sender Names via Webhooks
Status: [ ]

Tasks:
- [ ] Create and cache Discord Webhooks per target text channel to avoid repeated creation API calls
- [ ] Implement Go relay function to send messages via Webhook execution endpoint
- [ ] Pass the IRC nickname to Webhook `username` parameter
- [ ] Generate deterministic avatar URLs based on IRC nickname (e.g. ui-avatars.com or RoboHash) and pass to Webhook `avatar_url`

---

## I-006 Normalize Terminal Escape / HTML in irssi Bridge
Status: [ ]

Tasks:
- [ ] Strip or translate HTML tags (e.g. `<b>`, `<i>`) to Discord Markdown (e.g. `**`, `*`)
- [ ] Convert HTML entities (e.g. `&lt;`, `&gt;`, `&amp;`) back to raw plain text characters
- [ ] Decide parsing boundaries: clean formatting in irssi bridge Perl plugin or normalize in Go relayd ingestion layer

---

## I-007 Handle Relay Bots & Nested Senders
Status: [ ]

Tasks:
- [ ] Implement config option for known relay bot nicks (e.g. `||`)
- [ ] Parse nested sender pattern from message text (e.g., `<jw> message` or `[jw] message` inside the text body)
- [ ] Override the sender's nickname and message text with extracted values when relay bot match is detected
- [ ] Forward the extracted sender name to the Discord Webhook bridge (display as `jw (via ||)` or similar to indicate bridged/relayed status)

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
