# NOW.md

## 🎯 Current Goal

Make the system **state-safe under reconnect and initial load**.

Not adding features.
Fixing correctness.

---

## 🔥 Top Priority (Do in order)

1. H-005 Snapshot Ownership Fix

   - remove snapshot from plugin
   - move snapshot creation to relay
   - ensure snapshot is NOT part of journal

---

2. H-006 Reconnect & Resume

   - implement resume_from
   - snapshot fallback when replay fails
   - guarantee no duplication

---

3. H-007 Event Stream Correctness

   - dedupe by event_id
   - enforce ordering
   - fix snapshot boundary issues

---

## ⚠️ Stop Conditions

Stop immediately and fix if:

- snapshot appears in journal
- reconnect duplicates messages
- buffers disappear or duplicate
- unread count becomes inconsistent
- message order flips

---

## 🧪 Mandatory Test Loop

Repeat constantly:

1. open client
2. receive messages
3. disconnect network
4. reconnect
5. verify:
   - buffers intact
   - unread correct
   - no duplication
   - ordering preserved

---

## ❌ Do NOT Work On

- UI polish
- search
- push notifications
- auth beyond minimal session
- new features

---

## 🧠 Reminder

Snapshot is NOT an event.

It is a state reset boundary.

If treated like an event, everything breaks.