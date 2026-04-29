import { collapsePresenceEvents, createChatState } from "./state.js";

const proto = location.protocol === "https:" ? "wss" : "ws";
const ws = new WebSocket(`${proto}://${location.host}/ws`);
const state = createChatState();

const messagesDiv = document.getElementById("messages");
const statusDiv = document.getElementById("status");
const inputField = document.getElementById("input-field");
const targetField = document.getElementById("target-field");
const sendButton = document.getElementById("send-button");
const bufferList = document.getElementById("buffer-list");
const activeTitle = document.getElementById("active-title");

const PENDING_TIMEOUT_MS = 10000;
const pendingMessageTimeouts = new Map();

function deliveryMetaFor(msg) {
  const state = msg && msg.delivery_state;
  if (state === "pending") return { cls: "pending", label: "Pending" };
  if (state === "failed") return { cls: "failed", label: "Unconfirmed" };
  if (state === "sent") return { cls: "sent", label: "Sent" };
  return null;
}

function addMessage(msg) {
  const el = document.createElement("div");
  const delivery = deliveryMetaFor(msg);
  el.className = "message" + (delivery ? ` ${delivery.cls}` : "");
  const nick = msg.nick || "me";
  const text = msg.text || "";
  const status = delivery
    ? `<span class="delivery ${delivery.cls}">${delivery.label}</span>`
    : "";
  el.innerHTML = `<span class="nick">${nick}</span>: ${text}${status}`;
  if (delivery && delivery.cls === "failed") {
    el.title = "Send unconfirmed (timeout)";
  }
  messagesDiv.appendChild(el);
  messagesDiv.scrollTop = messagesDiv.scrollHeight;
  return el;
}

function renderMessages() {
  const active = state.getActiveBuffer();
  activeTitle.textContent = active ? active.label : "(none)";
  messagesDiv.innerHTML = "";
  const rows = collapsePresenceEvents(active ? active.messages : []);
  for (const msg of rows) {
    addMessage(msg);
  }
}

function renderBuffers() {
  const rows = state.listBuffers();
  bufferList.innerHTML = "";
  const active = state.getActiveBuffer();
  const activeID = active ? active.id : null;

  for (const buf of rows) {
    const item = document.createElement("button");
    item.type = "button";
    item.className = "buffer-item" + (buf.id === activeID ? " active" : "");
    item.dataset.bufferId = buf.id;

    const label = document.createElement("span");
    label.textContent = buf.label;
    item.appendChild(label);

    if (buf.unread > 0) {
      const badge = document.createElement("span");
      badge.className = "unread-badge";
      badge.textContent = String(buf.unread);
      item.appendChild(badge);
    }

    item.addEventListener("click", () => {
      state.setActiveBuffer(buf.id);
      targetField.value = buf.label;
      render();
    });

    bufferList.appendChild(item);
  }
}

function render() {
  renderBuffers();
  renderMessages();
}

ws.onopen = () => {
  statusDiv.textContent = "Connected";
  statusDiv.className = "status connected";
  inputField.disabled = false;
  sendButton.disabled = false;

  state.activateTarget(targetField.value || "#test");
  render();
};

ws.onmessage = (event) => {
  try {
    const ev = JSON.parse(event.data);
    if (ev.type === "message.created" || ev.type === "dm.created") {
      const msg = { ...(ev.payload || {}), event_id: ev.id || "" };
      const clientId = msg.client_id;
      if (clientId && pendingMessageTimeouts.has(clientId)) {
        clearTimeout(pendingMessageTimeouts.get(clientId));
        pendingMessageTimeouts.delete(clientId);
      }

      state.receiveMessage(msg);
      render();
      return;
    }

    if (ev.type === "highlight.created") {
      const payload = ev.payload || {};
      state.receiveHighlight({
        source_buffer_id: payload.source_buffer_id || payload.buffer_id || "",
        target: payload.target || "",
        nick: payload.nick || "",
        text: payload.text || "",
        timestamp: payload.timestamp || payload.ts || "",
        message: payload.message || null,
      });
      render();
    }
  } catch (e) {
    console.error("parse error", e);
  }
};

ws.onclose = () => {
  statusDiv.textContent = "Disconnected";
  statusDiv.className = "status disconnected";
  inputField.disabled = true;
  sendButton.disabled = true;
};

ws.onerror = (error) => {
  console.error("websocket error", error);
};

window.sendMessage = function sendMessage(event) {
  event.preventDefault();
  const text = inputField.value.trim();
  const target = targetField.value.trim();
  if (!text || !target) return;

  const clientId =
    "cl_" + Date.now() + "_" + Math.random().toString(36).slice(2, 11);

  state.activateTarget(target);
  state.markPendingSelf(clientId, target, { text, nick: "me" });

  const timeoutId = setTimeout(() => {
    if (pendingMessageTimeouts.has(clientId)) {
      pendingMessageTimeouts.delete(clientId);
      state.failPendingSelf(clientId);
      state.clearPendingSelf(clientId);
      render();
    }
  }, PENDING_TIMEOUT_MS);

  pendingMessageTimeouts.set(clientId, timeoutId);

  const command = {
    type: "send_message",
    payload: {
      text,
      client_id: clientId,
      target,
    },
  };

  ws.send(JSON.stringify(command));
  inputField.value = "";
  render();
};
