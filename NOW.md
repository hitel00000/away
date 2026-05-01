# NOW.md

Current focus:

Fix the smallest set of issues required to make the app usable.

No new features beyond observed friction.

---

## What to do (in order)

### 1. Fix correctness issues first

- fix duplicate buffers on reload
- ensure buffer list reflects actual joined channels

If data feels inconsistent, fix that before anything else.

---

### 2. Make basic navigation usable

- show all joined buffers from snapshot
- allow switching without confusion

No sorting, no grouping, no polish.

Just make it usable.

---

### 3. Add minimal read handling

- implement basic mark_read
- allow clearing unread manually

Do not implement precise read positions.

---

### 4. Make it barely usable on mobile

Limit scope strictly:

- channel list toggle (show/hide)
- message view full width
- input fixed at bottom

Do not attempt full responsive design.

---

### 5. Clean up obvious UI friction

Only fix what blocks usage:

- spacing consistency
- readable font sizes
- reduce visual noise

Do NOT:

- redesign layout
- introduce design system
- refactor components

---

## What NOT to do

- do not start G-002 (search)
- do not introduce new persistence layers
- do not redesign state management
- do not optimize prematurely

If something feels like a “nice improvement”,
it is probably out of scope.

---

## Working style

- make small patches
- test in real usage immediately
- commit frequently
- stop when it feels “good enough”

---

## Exit condition

Stop when:

- you can comfortably read and reply from phone
- reconnect does not confuse you
- buffer navigation is not frustrating

Then:

→ update TASKS.md
→ identify next real need (not assumed need)