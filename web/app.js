import { createChatState } from "./state.js";

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
const pendingMessages = new Map();

function addMessage(msg, isPending = false) {
  const el = document.createElement("div");
  el.className = isPending ? "message pending" : "message";
  const nick = msg.nick || "me";
  el.innerHTML = `<span class="nick">${nick}</span>: ${msg.text}`;
  messagesDiv.appendChild(el);
  messagesDiv.scrollTop = messagesDiv.scrollHeight;
  return el;
}

function renderMessages() {
  const active = state.getActiveBuffer();
  activeTitle.textContent = active ? active.label : "(none)";
  messagesDiv.innerHTML = "";
  const rows = active ? active.messages : [];
  for (const msg of rows) {
    addMessage(msg, false);
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
      const msg = ev.payload;
      const clientId = msg.client_id;

      if (clientId && pendingMessages.has(clientId)) {
        const { el, timeoutId } = pendingMessages.get(clientId);
        clearTimeout(timeoutId);
        el.className = "message";
        pendingMessages.delete(clientId);
        el.querySelector(".nick").textContent = msg.nick || "me";
      }

      state.receiveMessage(msg);
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
  state.markPendingSelf(clientId, target);

  const activeBefore = state.getActiveBuffer();
  const el = addMessage({ text, nick: "me" }, true);

  const timeoutId = setTimeout(() => {
    if (pendingMessages.has(clientId)) {
      const { el: timeoutEl } = pendingMessages.get(clientId);
      timeoutEl.className = "message unconfirmed";
      timeoutEl.title = "Send unconfirmed (timeout)";
      pendingMessages.delete(clientId);
      state.clearPendingSelf(clientId);
    }
  }, PENDING_TIMEOUT_MS);

  pendingMessages.set(clientId, { el, timeoutId, bufferID: activeBefore ? activeBefore.id : null });

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
