import test from "node:test";
import assert from "node:assert/strict";
import { collapsePresenceEvents, createChatState } from "./state.js";

function recv(st, target, nick = "alice", text = "hi", client_id = "") {
  st.receiveMessage({ target, nick, text, client_id });
}

function highlight(st, payload) {
  st.receiveHighlight(payload);
}

test("incoming message in inactive buffer increments unread", () => {
  const st = createChatState();
  st.activateTarget("#a");
  recv(st, "#b", "alice");

  const buffers = st.listBuffers();
  const b = buffers.find((x) => x.id === "ch:#b");
  assert.equal(b.unread, 1);
});

test("incoming message in active buffer does not increment unread", () => {
  const st = createChatState();
  st.activateTarget("#a");
  recv(st, "#a", "alice");

  const active = st.getActiveBuffer();
  assert.equal(active.id, "ch:#a");
  assert.equal(active.unread, 0);
});

test("switching into unread buffer marks read", () => {
  const st = createChatState();
  st.activateTarget("#a");
  recv(st, "#b", "alice");
  st.activateTarget("#b");

  const active = st.getActiveBuffer();
  assert.equal(active.id, "ch:#b");
  assert.equal(active.unread, 0);
});

test("mark-read on activation affects only activated buffer", () => {
  const st = createChatState();
  st.activateTarget("#a");
  recv(st, "#b", "alice");
  recv(st, "bob", "bob");

  const beforeB = st.listBuffers().find((x) => x.id === "ch:#b");
  const beforeDM = st.listBuffers().find((x) => x.id === "dm:bob");
  assert.equal(beforeB.unread, 1);
  assert.equal(beforeDM.unread, 1);

  st.activateTarget("#b");
  const afterB = st.listBuffers().find((x) => x.id === "ch:#b");
  const afterDM = st.listBuffers().find((x) => x.id === "dm:bob");
  assert.equal(afterB.unread, 0);
  assert.equal(afterDM.unread, 1);
});

test("channel and DM switching keeps state isolated", () => {
  const st = createChatState();
  st.activateTarget("#a");
  recv(st, "bob", "bob");
  recv(st, "#a", "alice");

  const dm = st.listBuffers().find((x) => x.id === "dm:bob");
  const ch = st.listBuffers().find((x) => x.id === "ch:#a");
  assert.equal(dm.unread, 1);
  assert.equal(ch.unread, 0);

  st.activateTarget("bob");
  assert.equal(st.getActiveBuffer().id, "dm:bob");
  assert.equal(st.getActiveBuffer().unread, 0);
});

test("active buffer target label tracks current selection", () => {
  const st = createChatState();
  st.activateTarget("#a");
  assert.equal(st.getActiveTarget(), "#a");
  st.activateTarget("bob");
  assert.equal(st.getActiveTarget(), "bob");
});

test("self-sent event never increments unread in same buffer", () => {
  const st = createChatState();
  st.activateTarget("#a");
  st.markPendingSelf("cid-1", "#a");
  recv(st, "#a", "someone-else", "mine", "cid-1");

  const active = st.getActiveBuffer();
  assert.equal(active.id, "ch:#a");
  assert.equal(active.unread, 0);
});

test("self-sent event in inactive buffer does not create unread", () => {
  const st = createChatState();
  st.activateTarget("#a");
  st.markPendingSelf("cid-2", "#b");
  recv(st, "#b", "alice", "mine", "cid-2");

  const b = st.listBuffers().find((x) => x.id === "ch:#b");
  assert.equal(b.unread, 0);
});

test("non-self message with same nick does increment unread when inactive", () => {
  const st = createChatState();
  st.activateTarget("#a");
  recv(st, "#b", "me", "not-correlated", "");

  const b = st.listBuffers().find((x) => x.id === "ch:#b");
  assert.equal(b.unread, 1);
});

test("pending self ACK must render only once", () => {
  const st = createChatState();
  st.activateTarget("#a");
  st.markPendingSelf("cid-ack-1", "#a");
  recv(st, "#a", "alice", "mine", "cid-ack-1");
  recv(st, "#a", "alice", "mine", "cid-ack-1");

  const active = st.getActiveBuffer();
  assert.equal(active.messages.length, 1);
  assert.equal(active.unread, 0);
});

test("highlight events populate mentions buffer", () => {
  const st = createChatState();
  st.activateTarget("#a");

  highlight(st, {
    source_buffer_id: "ch:#b",
    nick: "alice",
    text: "ping @me",
    timestamp: "2026-04-30T00:00:00Z",
  });

  const mentions = st.listBuffers().find((x) => x.id === "system:mentions");
  assert.equal(mentions.messages.length, 1);
  assert.equal(mentions.messages[0].source.id, "ch:#b");
  assert.equal(mentions.messages[0].nick, "alice");
  assert.equal(mentions.messages[0].text, "ping @me");
});

test("mentions unread increments on highlight when inactive", () => {
  const st = createChatState();
  st.activateTarget("#a");
  highlight(st, { source_buffer_id: "ch:#b", nick: "alice", text: "one" });
  highlight(st, { source_buffer_id: "ch:#b", nick: "alice", text: "two" });

  const mentions = st.listBuffers().find((x) => x.id === "system:mentions");
  assert.equal(mentions.unread, 2);
});

test("opening mentions clears only mentions unread", () => {
  const st = createChatState();
  st.activateTarget("#a");
  recv(st, "#b", "alice", "regular");
  highlight(st, { source_buffer_id: "ch:#b", nick: "alice", text: "highlight" });

  const beforeSource = st.listBuffers().find((x) => x.id === "ch:#b");
  const beforeMentions = st.listBuffers().find((x) => x.id === "system:mentions");
  assert.equal(beforeSource.unread, 1);
  assert.equal(beforeMentions.unread, 1);

  st.setActiveBuffer("system:mentions");

  const afterSource = st.listBuffers().find((x) => x.id === "ch:#b");
  const afterMentions = st.listBuffers().find((x) => x.id === "system:mentions");
  assert.equal(afterMentions.unread, 0);
  assert.equal(afterSource.unread, 1);
});

test("duplicate normal messages do not create mentions without highlight event", () => {
  const st = createChatState();
  st.activateTarget("#a");
  recv(st, "#b", "alice", "ping @me");
  recv(st, "#b", "alice", "ping @me");

  const mentions = st.listBuffers().find((x) => x.id === "system:mentions");
  assert.equal(mentions.messages.length, 0);
});

test("duplicate highlight event_id is deduped in mentions", () => {
  const st = createChatState();
  st.activateTarget("#a");

  highlight(st, {
    event_id: "hl-1",
    source_buffer_id: "ch:#b",
    nick: "alice",
    text: "ping @me",
    timestamp: "2026-04-30T00:00:00Z",
  });
  highlight(st, {
    event_id: "hl-1",
    source_buffer_id: "ch:#b",
    nick: "alice",
    text: "ping @me",
    timestamp: "2026-04-30T00:00:00Z",
  });

  const mentions = st.listBuffers().find((x) => x.id === "system:mentions");
  assert.equal(mentions.messages.length, 1);
  assert.equal(mentions.unread, 1);
});

test("pending becomes sent on correlated ACK", () => {
  const st = createChatState();
  st.activateTarget("#a");
  st.markPendingSelf("cid-f003-1", "#a", { nick: "me", text: "hello" });

  let active = st.getActiveBuffer();
  assert.equal(active.messages.length, 1);
  assert.equal(active.messages[0].delivery_state, "pending");

  recv(st, "#a", "me", "hello", "cid-f003-1");
  active = st.getActiveBuffer();
  assert.equal(active.messages.length, 1);
  assert.equal(active.messages[0].delivery_state, "sent");
});

test("pending becomes failed on timeout marker", () => {
  const st = createChatState();
  st.activateTarget("#a");
  st.markPendingSelf("cid-f003-2", "#a", { nick: "me", text: "hello" });

  assert.equal(st.failPendingSelf("cid-f003-2"), true);
  st.clearPendingSelf("cid-f003-2");

  const active = st.getActiveBuffer();
  assert.equal(active.messages.length, 1);
  assert.equal(active.messages[0].delivery_state, "failed");
});

test("duplicate ACK does not duplicate or corrupt state", () => {
  const st = createChatState();
  st.activateTarget("#a");
  st.markPendingSelf("cid-f003-3", "#a", { nick: "me", text: "hello" });
  recv(st, "#a", "me", "hello", "cid-f003-3");
  recv(st, "#a", "me", "hello", "cid-f003-3");

  const active = st.getActiveBuffer();
  assert.equal(active.messages.length, 1);
  assert.equal(active.messages[0].delivery_state, "sent");
});

test("unrelated ACK does not mutate wrong pending", () => {
  const st = createChatState();
  st.activateTarget("#a");
  st.markPendingSelf("cid-f003-4-a", "#a", { nick: "me", text: "a" });
  st.markPendingSelf("cid-f003-4-b", "#a", { nick: "me", text: "b" });

  recv(st, "#a", "me", "a", "cid-f003-4-a");

  const active = st.getActiveBuffer();
  assert.equal(active.messages.length, 2);
  assert.equal(active.messages[0].delivery_state, "sent");
  assert.equal(active.messages[1].delivery_state, "pending");
});

test("multiple pending ACK out-of-order resolves correctly", () => {
  const st = createChatState();
  st.activateTarget("#a");
  st.markPendingSelf("cid-f003-5-a", "#a", { nick: "me", text: "first" });
  st.markPendingSelf("cid-f003-5-b", "#a", { nick: "me", text: "second" });

  recv(st, "#a", "me", "second", "cid-f003-5-b");
  recv(st, "#a", "me", "first", "cid-f003-5-a");

  const active = st.getActiveBuffer();
  assert.equal(active.messages.length, 2);
  assert.equal(active.messages[0].client_id, "cid-f003-5-a");
  assert.equal(active.messages[0].delivery_state, "sent");
  assert.equal(active.messages[1].client_id, "cid-f003-5-b");
  assert.equal(active.messages[1].delivery_state, "sent");
});

test("timeout then late ACK settles to sent explicitly", () => {
  const st = createChatState();
  st.activateTarget("#a");
  st.markPendingSelf("cid-f003-6", "#a", { nick: "me", text: "late" });

  st.failPendingSelf("cid-f003-6");
  recv(st, "#a", "me", "late", "cid-f003-6");

  const active = st.getActiveBuffer();
  assert.equal(active.messages.length, 1);
  assert.equal(active.messages[0].delivery_state, "sent");
});

test("join burst collapses into one summary row", () => {
  const rows = [
    { event_type: "join", nick: "alice", text: "", ts: "2026-04-30T00:00:00Z" },
    { event_type: "join", nick: "bob", text: "", ts: "2026-04-30T00:00:02Z" },
    { event_type: "join", nick: "carol", text: "", ts: "2026-04-30T00:00:03Z" },
  ];
  const out = collapsePresenceEvents(rows);
  assert.equal(out.length, 1);
  assert.equal(out[0]._collapsed_presence, true);
  assert.equal(out[0].text, "alice, bob, carol joined");
});

test("conversational messages are not collapsed", () => {
  const rows = [
    { nick: "alice", text: "hello", ts: "2026-04-30T00:00:00Z" },
    { nick: "bob", text: "world", ts: "2026-04-30T00:00:01Z" },
    { nick: "carol", text: "!" },
  ];
  const out = collapsePresenceEvents(rows);
  assert.equal(out.length, 3);
  assert.equal(out[0].text, "hello");
  assert.equal(out[1].text, "world");
  assert.equal(out[2].text, "!");
});

test("message boundary splits join bursts into separate groups", () => {
  const rows = [
    { event_type: "join", nick: "alice", ts: "2026-04-30T00:00:00Z" },
    { event_type: "join", nick: "bob", ts: "2026-04-30T00:00:01Z" },
    { nick: "eve", text: "real chat", ts: "2026-04-30T00:00:02Z" },
    { event_type: "join", nick: "carol", ts: "2026-04-30T00:00:03Z" },
    { event_type: "join", nick: "dave", ts: "2026-04-30T00:00:04Z" },
    { event_type: "join", nick: "frank", ts: "2026-04-30T00:00:05Z" },
  ];
  const out = collapsePresenceEvents(rows);
  assert.equal(out.length, 3);
  assert.equal(out[0].event_type, "join");
  assert.equal(out[1].text, "real chat");
  assert.equal(out[2]._collapsed_presence, true);
  assert.equal(out[2].text, "carol, dave, frank joined");
});

test("collapse window boundary does not eat unrelated presence events", () => {
  const rows = [
    { event_type: "join", nick: "alice", ts: "2026-04-30T00:00:00Z" },
    { event_type: "join", nick: "bob", ts: "2026-04-30T00:00:10Z" },
    { event_type: "join", nick: "carol", ts: "2026-04-30T00:00:20Z" },
    { event_type: "join", nick: "dave", ts: "2026-04-30T00:00:21Z" },
    { event_type: "join", nick: "erin", ts: "2026-04-30T00:00:22Z" },
  ];
  const out = collapsePresenceEvents(rows);
  assert.equal(out.length, 3);
  assert.equal(out[0].event_type, "join");
  assert.equal(out[1].event_type, "join");
  assert.equal(out[2]._collapsed_presence, true);
  assert.equal(out[2].text, "carol, dave, erin joined");
});

test("part and nick presence events collapse by type", () => {
  const partRows = [
    { event_type: "part", nick: "alice" },
    { event_type: "part", nick: "bob" },
    { event_type: "part", nick: "carol" },
  ];
  const nickRows = [
    { event_type: "nick", nick: "alice" },
    { event_type: "nick", nick: "bob" },
    { event_type: "nick", nick: "carol" },
  ];
  const partOut = collapsePresenceEvents(partRows);
  const nickOut = collapsePresenceEvents(nickRows);
  assert.equal(partOut.length, 1);
  assert.equal(partOut[0].text, "alice, bob, carol left");
  assert.equal(nickOut.length, 1);
  assert.equal(nickOut[0].text, "3 users changed nick");
});
