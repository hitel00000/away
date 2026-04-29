# NOW.md

Current focus: Phase 0.2 complete / entering Phase 0.3

## Completed (Phase 0.2 Usability Pass)

- [x] F-001 Buffer / Unread Correctness
- [x] F-002 Mention Inbox MVP
- [x] F-003 Send Acknowledgement
- [x] F-004 Presence Noise Collapse

Phase 0.2 is considered complete.

System now has:

- multi-buffer navigation
- unread correctness
- mentions inbox
- optimistic send feedback
- presence noise reduction

The “daily companion” loop exists.

---

## Current mode

Stabilize and validate through real use.

Use the system.
Watch for pain points.
Prefer learning over feature growth.

Do not expand scope casually.

---

## Next candidate

Primary next spike:

- [ ] G-001 Append-Only Event Journal Spike

Goal:
Survive relay/browser restart with recent context.

Keep it tiny.

No sqlite.
No persistence framework.
Just a spike.

---

## After G-001 reassess

Possible outcomes:

1. Stop at lightweight local-first companion.
2. Explore G-002 search spike.
3. Reprioritize based on real usage pain.

No commitment yet.

---

## Rules

- favor simplicity over capability
- preserve single-user companion scope
- minimal patches
- avoid infrastructure drift
- reassess before new phase expansion