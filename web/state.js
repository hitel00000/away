export function createChatState() {
  const MENTIONS_BUFFER_ID = "system:mentions";
  const buffers = new Map();
  const bufferOrder = [];
  const pendingSelfClientIDs = new Map();
  const pendingSelfMessageByClientID = new Map();
  const consumedSelfAckClientIDs = new Set();
  const seenHighlightKeys = new Set();
  let activeBufferID = null;

  function ensureBuffer(bufferID, type, label) {
    let buf = buffers.get(bufferID);
    if (!buf) {
      buf = {
        id: bufferID,
        type,
        label,
        unread: 0,
        messages: [],
      };
      buffers.set(bufferID, buf);
      bufferOrder.push(bufferID);
    }
    return buf;
  }

  function normalizeTarget(target) {
    const t = String(target || "").trim();
    if (t.startsWith("#")) {
      return { id: `ch:${t}`, type: "channel", label: t };
    }
    if (t.length > 0) {
      return { id: `dm:${t}`, type: "dm", label: t };
    }
    return { id: "ch:#test", type: "channel", label: "#test" };
  }

  function deriveBufferFromMessage(msg) {
    const bufferID = String((msg && msg.buffer_id) || "").trim();
    if (bufferID.startsWith("ch:")) {
      return { id: bufferID, type: "channel", label: bufferID.slice(3) };
    }
    if (bufferID.startsWith("dm:")) {
      return { id: bufferID, type: "dm", label: bufferID.slice(3) };
    }

    if (msg && msg.target) {
      return normalizeTarget(msg.target);
    }

    const nick = String((msg && msg.nick) || "").trim();
    if (nick.startsWith("#")) {
      return normalizeTarget(nick);
    }
    if (nick) {
      return normalizeTarget(nick);
    }
    return normalizeTarget("#test");
  }

  function deriveSourceBufferFromHighlight(highlight) {
    const sourceBufferID = String(
      (highlight && (highlight.source_buffer_id || highlight.buffer_id)) || ""
    ).trim();
    if (sourceBufferID.startsWith("ch:")) {
      return { id: sourceBufferID, type: "channel", label: sourceBufferID.slice(3) };
    }
    if (sourceBufferID.startsWith("dm:")) {
      return { id: sourceBufferID, type: "dm", label: sourceBufferID.slice(3) };
    }

    if (highlight && highlight.target) {
      return normalizeTarget(highlight.target);
    }
    if (highlight && highlight.message) {
      return deriveBufferFromMessage(highlight.message);
    }
    return normalizeTarget("#test");
  }

  function markPendingSelf(clientID, target, draftMessage = null) {
    if (!clientID) return;
    const buf = normalizeTarget(target);
    pendingSelfClientIDs.set(clientID, buf.id);

    if (!draftMessage) return;
    const ensured = ensureBuffer(buf.id, buf.type, buf.label);
    const message = {
      ...draftMessage,
      target,
      client_id: clientID,
      delivery_state: "pending",
    };
    ensured.messages.push(message);
    pendingSelfMessageByClientID.set(clientID, message);
  }

  function clearPendingSelf(clientID) {
    if (!clientID) return;
    pendingSelfClientIDs.delete(clientID);
    pendingSelfMessageByClientID.delete(clientID);
  }

  function failPendingSelf(clientID) {
    if (!clientID) return false;
    const pending = pendingSelfMessageByClientID.get(clientID);
    if (!pending) {
      pendingSelfClientIDs.delete(clientID);
      return false;
    }
    pending.delivery_state = "failed";
    return true;
  }

  function setActiveBuffer(bufferID) {
    const buf = buffers.get(bufferID);
    if (!buf) return false;
    activeBufferID = bufferID;
    buf.unread = 0;
    return true;
  }

  function activateTarget(target) {
    const bufDef = normalizeTarget(target);
    ensureBuffer(bufDef.id, bufDef.type, bufDef.label);
    setActiveBuffer(bufDef.id);
    return bufDef.id;
  }

  function receiveMessage(msg) {
    const clientID = msg && msg.client_id;
    const fromPending = clientID && pendingSelfClientIDs.has(clientID);
    if (fromPending && consumedSelfAckClientIDs.has(clientID)) {
      return {
        bufferID: pendingSelfClientIDs.get(clientID) || "",
        isSelf: true,
        unread: 0,
        active: false,
      };
    }
    const pendingBufferID = fromPending ? pendingSelfClientIDs.get(clientID) : null;
    const derived = deriveBufferFromMessage(msg);
    const bufferID = pendingBufferID || derived.id;
    const type = pendingBufferID
      ? (pendingBufferID.startsWith("dm:") ? "dm" : "channel")
      : derived.type;
    const label = pendingBufferID
      ? pendingBufferID.slice(3)
      : derived.label;

    const buf = ensureBuffer(bufferID, type, label);
    if (fromPending) {
      const pending = pendingSelfMessageByClientID.get(clientID);
      if (pending) {
        Object.assign(pending, msg);
        pending.delivery_state = "sent";
      } else {
        buf.messages.push({ ...msg, delivery_state: "sent" });
      }
    } else {
      buf.messages.push(msg);
    }

    // Self-correlation must use opaque client ids, not nick/text heuristics.
    const isSelf = fromPending;

    if (activeBufferID !== bufferID && !isSelf) {
      buf.unread += 1;
    }

    if (fromPending) {
      pendingSelfClientIDs.delete(clientID);
      pendingSelfMessageByClientID.delete(clientID);
      consumedSelfAckClientIDs.add(clientID);
    }

    return {
      bufferID,
      isSelf,
      unread: buf.unread,
      active: activeBufferID === bufferID,
    };
  }

  function receiveHighlight(highlight) {
    const source = deriveSourceBufferFromHighlight(highlight);
    const key = String((highlight && highlight.event_id) || "").trim() || [
      source.id,
      (highlight && highlight.nick) || "",
      (highlight && highlight.text) || "",
      (highlight && (highlight.ts || highlight.timestamp)) || "",
    ].join("|");
    if (seenHighlightKeys.has(key)) {
      const mentions = ensureBuffer(MENTIONS_BUFFER_ID, "system", "Mentions");
      return {
        bufferID: MENTIONS_BUFFER_ID,
        unread: mentions.unread,
        active: activeBufferID === MENTIONS_BUFFER_ID,
      };
    }
    seenHighlightKeys.add(key);

    const mentions = ensureBuffer(MENTIONS_BUFFER_ID, "system", "Mentions");
    mentions.messages.push({
      source,
      nick: (highlight && highlight.nick) || "",
      text: (highlight && highlight.text) || "",
      ts:
        (highlight && (highlight.ts || highlight.timestamp)) ||
        "",
    });

    if (activeBufferID !== MENTIONS_BUFFER_ID) {
      mentions.unread += 1;
    }

    return {
      bufferID: MENTIONS_BUFFER_ID,
      unread: mentions.unread,
      active: activeBufferID === MENTIONS_BUFFER_ID,
    };
  }

  function listBuffers() {
    return bufferOrder.map((id) => buffers.get(id));
  }

  function getActiveBuffer() {
    return activeBufferID ? buffers.get(activeBufferID) : null;
  }

  function getActiveTarget() {
    const active = getActiveBuffer();
    return active ? active.label : "";
  }

  ensureBuffer(MENTIONS_BUFFER_ID, "system", "Mentions");

  return {
    ensureBuffer,
    normalizeTarget,
    activateTarget,
    setActiveBuffer,
    receiveMessage,
    receiveHighlight,
    listBuffers,
    getActiveBuffer,
    getActiveTarget,
    markPendingSelf,
    clearPendingSelf,
    failPendingSelf,
  };
}

const PRESENCE_TYPES = new Set(["join", "part", "quit", "nick"]);
const PRESENCE_WINDOW_MS = 15000;
const PRESENCE_COLLAPSE_MIN = 3;
const PRESENCE_NAME_LIST_MAX = 3;

function parseTimestampMs(value) {
  if (!value) return null;
  if (typeof value === "number" && Number.isFinite(value)) return value;
  const n = Date.parse(String(value));
  return Number.isFinite(n) ? n : null;
}

function presenceTypeForMessage(msg) {
  const type = String((msg && (msg.event_type || msg.type || msg.kind)) || "")
    .trim()
    .toLowerCase();
  if (PRESENCE_TYPES.has(type)) return type;
  return "";
}

function summarizePresenceGroup(type, rows) {
  const names = [];
  for (const row of rows) {
    const nick = String((row && row.nick) || "").trim();
    if (nick) names.push(nick);
  }

  if (type === "nick") {
    if (names.length === 1) return `${names[0]} changed nick`;
    return `${rows.length} users changed nick`;
  }

  const verb = type === "join"
    ? "joined"
    : (type === "part" ? "left" : "quit");
  if (names.length <= PRESENCE_NAME_LIST_MAX) {
    return `${names.join(", ")} ${verb}`;
  }
  return `${rows.length} users ${verb}`;
}

export function collapsePresenceEvents(rows) {
  const result = [];
  let pending = null;

  function flushPending() {
    if (!pending) return;
    if (pending.rows.length >= PRESENCE_COLLAPSE_MIN) {
      const first = pending.rows[0];
      const last = pending.rows[pending.rows.length - 1];
      result.push({
        event_type: pending.type,
        nick: "",
        text: summarizePresenceGroup(pending.type, pending.rows),
        ts: (last && last.ts) || (first && first.ts) || "",
        timestamp: (last && last.timestamp) || (first && first.timestamp) || "",
        _collapsed_presence: true,
        _presence_count: pending.rows.length,
      });
    } else {
      result.push(...pending.rows);
    }
    pending = null;
  }

  for (const row of rows || []) {
    const presenceType = presenceTypeForMessage(row);
    if (!presenceType) {
      flushPending();
      result.push(row);
      continue;
    }

    const rowTs = parseTimestampMs(row && (row.ts || row.timestamp));
    if (!pending) {
      pending = { type: presenceType, rows: [row], firstTs: rowTs };
      continue;
    }

    const sameType = pending.type === presenceType;
    const withinWindow = (
      pending.firstTs === null ||
      rowTs === null ||
      rowTs - pending.firstTs <= PRESENCE_WINDOW_MS
    );

    if (sameType && withinWindow) {
      pending.rows.push(row);
      continue;
    }

    flushPending();
    pending = { type: presenceType, rows: [row], firstTs: rowTs };
  }
  flushPending();
  return result;
}
