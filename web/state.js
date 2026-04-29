export function createChatState() {
  const MENTIONS_BUFFER_ID = "system:mentions";
  const buffers = new Map();
  const bufferOrder = [];
  const pendingSelfClientIDs = new Map();
  const consumedSelfAckClientIDs = new Set();
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

  function markPendingSelf(clientID, target) {
    if (!clientID) return;
    const buf = normalizeTarget(target);
    pendingSelfClientIDs.set(clientID, buf.id);
  }

  function clearPendingSelf(clientID) {
    if (!clientID) return;
    pendingSelfClientIDs.delete(clientID);
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
    buf.messages.push(msg);

    // Self-correlation must use opaque client ids, not nick/text heuristics.
    const isSelf = fromPending;

    if (activeBufferID !== bufferID && !isSelf) {
      buf.unread += 1;
    }

    if (fromPending) {
      pendingSelfClientIDs.delete(clientID);
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
  };
}
