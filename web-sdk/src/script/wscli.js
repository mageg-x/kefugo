// wscli.js
import {
  BUSINESS_MESSAGE_TYPES,
  BUSINESS_CONTENT_TYPES,
  buildOutgoingBusinessPayload,
  normalizeIncomingBusinessMessage,
} from "./message-protocol.js";

// compat legacy caller naming
export const MSG_TYPES = {
  MESSAGE_REQ: BUSINESS_MESSAGE_TYPES.VISITOR,
  MESSAGE_TYPING: BUSINESS_MESSAGE_TYPES.TYPING,
  MESSAGE_ACK: BUSINESS_MESSAGE_TYPES.ACK,
  MESSAGE_CLOSE: BUSINESS_MESSAGE_TYPES.CLOSE,
  MESSAGE_RSP: BUSINESS_MESSAGE_TYPES.AGENT,
  MESSAGE_SYSTEM: BUSINESS_MESSAGE_TYPES.SYSTEM,
};

// compat legacy caller naming
export const CONTENT_TYPES = {
  TEXT: BUSINESS_CONTENT_TYPES.TEXT,
  IMAGE: BUSINESS_CONTENT_TYPES.IMAGE,
  AUDIO: BUSINESS_CONTENT_TYPES.AUDIO,
  FILE: BUSINESS_CONTENT_TYPES.FILE,
};

/**
 * session status
 */
export const SESSION_STATUS = {
  WAITING: "waiting",
  CHATTING: "chatting",
  CLOSED: "closed",
};

/**
 * agent websocket client
 */
export class WSClient {
  constructor(appid, visitorId, options = {}) {
    this.appid = appid;
    this.visitorId = visitorId;
    this.wsUrl = options.wsUrl || "ws://localhost:5300/ws/chat";
    this.onMessage = options.onMessage || (() => {});
    this.onStatusChange = options.onStatusChange || (() => {});
    this.onError = options.onError || console.error;
    this.onConnected = options.onConnected || (() => {});

    this.ws = null;
    this.isConnected = false;
    this.reconnectAttempts = 0;
    this.maxReconnectAttempts = Number(options.maxReconnectAttempts || 15);
    this.reconnectBaseDelay = Number(options.reconnectBaseDelay || 1000);
    this.reconnectMaxDelay = Number(options.reconnectMaxDelay || 30000);
    this.disableReconnect = false;
  }

  connect() {
    if (this.ws?.readyState === WebSocket.OPEN) return;

    const url = `${this.wsUrl}?app_id=${encodeURIComponent(this.appid)}&visitor_id=${encodeURIComponent(this.visitorId)}`;
    this.ws = new WebSocket(url);

    this.ws.onopen = () => {
      this.isConnected = true;
      this.reconnectAttempts = 0;
      this.onConnected();
      this.onStatusChange("connected");
    };

    this.ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data);
        this._handleIncoming(msg);
      } catch (e) {
        this.onError("invalid WS message:", event.data);
      }
    };

    this.ws.onclose = (event) => {
      this.isConnected = false;
      // 4001: server marked connection as replaced by same-session newer connection.
      // continuing reconnect here may cause reconnect ping-pong, stop auto reconnect.
      if (Number(event?.code) === 4001) {
        this.disableReconnect = true;
        this.onStatusChange("replaced");
        return;
      }
      this.onStatusChange("disconnected");
      this._reconnect();
    };

    this.ws.onerror = (err) => {
      this.onError("WS error:", err);
      this.ws.close(); // trigger onclose
    };
  }

  _handleIncoming(msg) {
    const normalized = normalizeIncomingBusinessMessage(msg);
    if (!normalized) {
      this.onError("invalid business message:", msg);
      return;
    }
    this.onMessage(normalized);
  }

  // --- send api ---

  sendText(content, extra = {}) {
    if (!this.isConnected) return;
    const clientId = this._createClientId("visitor");
    this._send(
      MSG_TYPES.MESSAGE_REQ,
      {
        ...buildOutgoingBusinessPayload(CONTENT_TYPES.TEXT, { content, ...extra }),
        client_id: clientId,
      }
    );
    return clientId;
  }

  sendImage(url, name = "image.jpg", extra = {}) {
    if (!this.isConnected) return;
    const clientId = this._createClientId("visitor");
    this._send(
      MSG_TYPES.MESSAGE_REQ,
      {
        ...buildOutgoingBusinessPayload(CONTENT_TYPES.IMAGE, { url, name, ...extra }),
        client_id: clientId,
      }
    );
    return clientId;
  }

  sendAudio(url, duration, extra = {}) {
    if (!this.isConnected) return;
    const clientId = this._createClientId("visitor");
    this._send(
      MSG_TYPES.MESSAGE_REQ,
      {
        ...buildOutgoingBusinessPayload(CONTENT_TYPES.AUDIO, { url, duration, ...extra }),
        client_id: clientId,
      }
    );
    return clientId;
  }

  sendFile(url, name, size, extra = {}) {
    if (!this.isConnected) return;
    const clientId = this._createClientId("visitor");
    this._send(
      MSG_TYPES.MESSAGE_REQ,
      {
        ...buildOutgoingBusinessPayload(CONTENT_TYPES.FILE, { url, name, size, ...extra }),
        client_id: clientId,
      }
    );
    return clientId;
  }

  sendTyping() {
    if (!this.isConnected) return;
    this._send(MSG_TYPES.MESSAGE_TYPING);
  }

  sendAck(msgId, status = "received") {
    if (!this.isConnected) return;
    const normalizedMsgId = String(msgId || "").trim();
    if (!normalizedMsgId) return;
    this._send(MSG_TYPES.MESSAGE_ACK, {
      msg_id: normalizedMsgId,
      status: String(status || "received"),
      timestamp: Math.floor(Date.now() / 1000),
    });
  }

  closeSession() {
    if (!this.isConnected) return;
    this._send(MSG_TYPES.MESSAGE_CLOSE);
  }

  // --- internal methods ---
  _send(type, payload = {}) {
    this.ws.send(JSON.stringify({ type, payload }));
  }

  _createClientId(prefix = "cid") {
    return `${prefix}_${Date.now().toString(36)}_${Math.random().toString(36).slice(2, 10)}`;
  }

  _reconnect() {
    if (this.disableReconnect) {
      return;
    }
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      this.onStatusChange("reconnect-failed");
      return;
    }

    const expDelay = this.reconnectBaseDelay * Math.pow(2, this.reconnectAttempts);
    const capped = Math.min(expDelay, this.reconnectMaxDelay);
    const jitterRatio = 0.2;
    const jitter = capped * (Math.random() * jitterRatio);
    const delay = Math.floor(capped + jitter);
    setTimeout(() => {
      this.reconnectAttempts++;
      this.connect();
    }, delay);
  }

  disconnect() {
    this.disableReconnect = true;
    this.reconnectAttempts = this.maxReconnectAttempts;
    this.ws?.close();
  }
}
