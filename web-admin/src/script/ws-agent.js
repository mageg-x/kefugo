import {
  BUSINESS_CONTENT_TYPES,
  BUSINESS_MESSAGE_TYPES,
  base58EncodeSessionID,
  buildOutgoingBusinessPayload,
  normalizeIncomingBusinessMessage,
} from "./message-protocol";

function toWSBase(httpBase) {
  try {
    const url = new URL(httpBase);
    const protocol = url.protocol === "https:" ? "wss:" : "ws:";
    return `${protocol}//${url.host}`;
  } catch {
    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
    return `${protocol}//${window.location.host}`;
  }
}

function createClientId(prefix = "cid") {
  return `${prefix}_${Date.now().toString(36)}_${Math.random().toString(36).slice(2, 10)}`;
}

export class AgentWSClient {
  constructor(token, options = {}) {
    this.token = token || "";
    this.apiBaseUrl = options.apiBaseUrl || window.location.origin;
    this.onMessage = options.onMessage || (() => {});
    this.onStatus = options.onStatus || (() => {});
    this.onError = options.onError || (() => {});
    this.ws = null;
    this.manualClose = false;
    this.reconnectAttempts = 0;
    this.maxReconnectAttempts = Number(options.maxReconnectAttempts || 20);
    this.reconnectBaseDelay = Number(options.reconnectBaseDelay || 1000);
    this.reconnectMaxDelay = Number(options.reconnectMaxDelay || 30000);
    this.reconnectJitterRatio = Number(options.reconnectJitterRatio || 0.2);
    this.pendingQueue = [];
    this.receivedMsgIds = new Set();
    this.receivedMsgIdLimit = Number(options.receivedMsgIdLimit || 2000);
  }

  buildURL() {
    const base = toWSBase(this.apiBaseUrl);
    const url = new URL(`${base}/ws/agent`);
    if (this.token) {
      url.searchParams.set("token", this.token);
    }
    return url.toString();
  }

  connect() {
    if (!this.token) {
      return;
    }
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      return;
    }

    this.ws = new WebSocket(this.buildURL());
    this.ws.onopen = () => {
      this.reconnectAttempts = 0;
      this.onStatus("connected");
      this.flushQueue();
    };
    this.ws.onmessage = (event) => {
      try {
        const packet = JSON.parse(event.data);
        const msg = normalizeIncomingBusinessMessage(packet);
        if (msg) {
          const dedupId = String(msg?.msg_id || msg?.payload?.msg_id || "").trim();
          if (dedupId) {
            if (this.receivedMsgIds.has(dedupId)) {
              return;
            }
            this.receivedMsgIds.add(dedupId);
            if (this.receivedMsgIds.size > this.receivedMsgIdLimit) {
              const first = this.receivedMsgIds.values().next().value;
              if (first) {
                this.receivedMsgIds.delete(first);
              }
            }
          }
          this.onMessage(msg);
        }
      } catch (error) {
        this.onError(error);
      }
    };
    this.ws.onclose = () => {
      this.onStatus("disconnected");
      if (!this.manualClose) {
        this.reconnect();
      }
    };
    this.ws.onerror = (error) => {
      this.onError(error);
      this.ws?.close();
    };
  }

  reconnect() {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      this.onStatus("reconnect-failed");
      return;
    }
    const expDelay = this.reconnectBaseDelay * Math.pow(2, this.reconnectAttempts);
    const capped = Math.min(expDelay, this.reconnectMaxDelay);
    const jitter = capped * Math.max(0, Math.min(this.reconnectJitterRatio, 1)) * Math.random();
    const delay = Math.floor(capped + jitter);
    this.reconnectAttempts += 1;
    setTimeout(() => this.connect(), delay);
  }

  sendText(sid, content, extra = {}, onSent) {
    return this.sendPayload(sid, BUSINESS_CONTENT_TYPES.TEXT, { content, ...extra }, onSent);
  }

  sendImage(sid, url, name = "image.jpg", extra = {}, onSent) {
    return this.sendPayload(sid, BUSINESS_CONTENT_TYPES.IMAGE, { url, name, ...extra }, onSent);
  }

  sendAudio(sid, url, duration = 0, extra = {}, onSent) {
    return this.sendPayload(sid, BUSINESS_CONTENT_TYPES.AUDIO, { url, duration, ...extra }, onSent);
  }

  sendFile(sid, url, name = "file", size = 0, extra = {}, onSent) {
    return this.sendPayload(sid, BUSINESS_CONTENT_TYPES.FILE, { url, name, size, ...extra }, onSent);
  }

  sendPayload(sid, contentType, data, onSent) {
    const encodedSID = base58EncodeSessionID(sid);
    const clientId = data?.clientId || createClientId("agent");
    const packet = {
      type: BUSINESS_MESSAGE_TYPES.AGENT,
      sid: encodedSID,
      payload: {
        ...buildOutgoingBusinessPayload(contentType, data),
        client_id: clientId,
      },
    };
    const ok = this.sendPacket(packet, onSent);
    return { ok, clientId };
  }

  sendClose(sid, onSent) {
    const encodedSID = base58EncodeSessionID(sid);
    return this.sendPacket({
      type: BUSINESS_MESSAGE_TYPES.CLOSE,
      sid: encodedSID,
      payload: {},
    }, onSent);
  }

  sendTyping(sid, onSent) {
    const encodedSID = base58EncodeSessionID(sid);
    return this.sendPacket({
      type: BUSINESS_MESSAGE_TYPES.TYPING,
      sid: encodedSID,
      payload: {},
    }, onSent);
  }

  sendPacket(packet, onSent) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(packet));
      if (typeof onSent === "function") onSent();
      return true;
    }
    this.pendingQueue.push({ packet, onSent });
    if (!this.ws || this.ws.readyState === WebSocket.CLOSED) {
      this.connect();
    }
    return false;
  }

  flushQueue() {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN || this.pendingQueue.length === 0) {
      return;
    }
    const queue = [...this.pendingQueue];
    this.pendingQueue = [];
    for (const item of queue) {
      this.ws.send(JSON.stringify(item.packet));
      if (typeof item.onSent === "function") item.onSent();
    }
  }

  disconnect() {
    this.manualClose = true;
    this.ws?.close();
  }
}

export { BUSINESS_CONTENT_TYPES, BUSINESS_MESSAGE_TYPES };
