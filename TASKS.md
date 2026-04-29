# TASKS.md

Away MVP backlog (post D-003)

Status
- [ ] todo
- [~] in progress
- [x] done
- [blocked]

Rule:
Work top-to-bottom.
Do not expand scope.
Prefer robustness over feature growth.

---

# Completed

A-001 [x]
A-002 [x]
A-003 [x]
A-004 [x]
A-005 [x]

B-001 [x]
B-002 [x]

C-001 [x]
C-002 [x]

D-001 [x]
D-002 [x]
D-003a [x]
D-003b [x]
D-003c [x]

---

# Milestone E — MVP Hardening

Goal:
Validate assumptions and remove highest-risk defects.

## E-001 Verify own-message ordering assumption
Status: [x]

Problem:
D-003a assumes send order == irssi own echo order.

Implement:
- stress test rapid consecutive sends
- verify no client_id mismatch
- document assumption or fix if invalid

Acceptance:
50 rapid sends reconcile correctly.

Priority: High

---

## E-002 Multi-buffer routing correctness
Status: [ ]

Implement:
- verify sends land in intended buffer
- verify channel vs DM routing

Acceptance:
multi-buffer manual test passes.

Priority: High

---

## E-003 Event schema freeze (v1)
Status: [ ]

Implement:
- review current emitted schemas
- remove accidental drift
- document message.created and dm.created as stable

Acceptance:
schema fixtures committed.

Priority: High

---

## E-004 Reconnect smoke test
Status: [ ]

Implement:
- restart relay
- refresh browser
- verify D-001/D-002 behavior together

Acceptance:
reconnect flow works end-to-end.

Priority: Medium

---

## E-005 Backlog MVP spike
Status: [ ]

Very small spike only:
- fetch recent 20 messages on connect

Timebox:
1 session max.

If scope grows, stop.

Priority: Medium

---

# Deferred (Not Now)

- auth pairing
- push
- sqlite
- search
- mobile polish

---

## Patch Budget Rule

Default task budget:
~30 lines preferred
>100 lines requires justification