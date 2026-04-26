const BASE58_ALPHABET = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz";

export const BUSINESS_MESSAGE_TYPES = {
  VISITOR: "message.visitor",
  AGENT: "message.agent",
  SYSTEM: "message.system",
  TYPING: "message.typing",
  CLOSE: "message.close",
};

export const BUSINESS_CONTENT_TYPES = {
  TEXT: "text",
  IMAGE: "image",
  AUDIO: "audio",
  FILE: "file",
};

export const UI_MESSAGE_STATUS = {
  SENDING: "sending",
  SENT: "sent",
  FAILED: "failed",
};

function normalizeReplyTo(rawReply) {
  if (!rawReply || typeof rawReply !== "object" || Array.isArray(rawReply)) {
    return null;
  }
  const msgId = safeString(rawReply.msg_id || rawReply.msgId);
  const contentType = safeString(rawReply.content_type || rawReply.contentType || BUSINESS_CONTENT_TYPES.TEXT).toLowerCase();
  const preview = safeString(rawReply.preview || rawReply.content || "");
  const sender = safeString(rawReply.sender || rawReply.from_name || rawReply.fromName || "");
  const timestamp = safeNumber(rawReply.timestamp, 0);
  if (!msgId && !preview) {
    return null;
  }
  return {
    msgId,
    contentType,
    preview,
    sender,
    timestamp,
  };
}

function safeString(value) {
  if (typeof value === "string") {
    return value;
  }
  if (value == null) {
    return "";
  }
  return String(value);
}

function safeNumber(value, fallback = 0) {
  const num = Number(value);
  return Number.isFinite(num) ? num : fallback;
}

function inferContentType(payload = {}) {
  const raw = safeString(payload.content_type || payload.contentType).toLowerCase();
  const allowed = new Set(Object.values(BUSINESS_CONTENT_TYPES));
  if (allowed.has(raw)) return raw;
  const url = safeString(payload.url || payload.content).toLowerCase();
  const name = safeString(payload.name).toLowerCase();
  const target = `${url} ${name}`;
  if (/\.(png|jpe?g|gif|webp|bmp|svg)(\?|#|$)/.test(target)) return BUSINESS_CONTENT_TYPES.IMAGE;
  if (/\.(mp3|wav|ogg|m4a|aac|webm)(\?|#|$)/.test(target)) return BUSINESS_CONTENT_TYPES.AUDIO;
  if (payload.duration != null && Number(payload.duration) > 0) return BUSINESS_CONTENT_TYPES.AUDIO;
  if (payload.size != null || safeString(payload.name)) return BUSINESS_CONTENT_TYPES.FILE;
  return BUSINESS_CONTENT_TYPES.TEXT;
}

export function createLocalId(prefix = "local") {
  return `${prefix}_${Date.now()}_${Math.random().toString(16).slice(2)}`;
}

export function normalizeUnixTs(value) {
  const raw = safeNumber(value, 0);
  if (!raw) {
    return Math.floor(Date.now() / 1000);
  }
  return raw > 1e12 ? Math.floor(raw / 1000) : raw;
}

export function base58EncodeSessionID(sessionID) {
  const bytes = new TextEncoder().encode(safeString(sessionID));
  if (bytes.length === 0) {
    return "";
  }

  let zeros = 0;
  while (zeros < bytes.length && bytes[zeros] === 0) {
    zeros++;
  }

  const encoded = [];
  const input = Array.from(bytes);
  let startAt = zeros;
  while (startAt < input.length) {
    let mod = 0;
    for (let i = startAt; i < input.length; i++) {
      const num = input[i] + mod * 256;
      input[i] = Math.floor(num / 58);
      mod = num % 58;
    }
    encoded.push(BASE58_ALPHABET[mod]);
    while (startAt < input.length && input[startAt] === 0) {
      startAt++;
    }
  }

  for (let i = 0; i < zeros; i++) {
    encoded.push("1");
  }
  return encoded.reverse().join("");
}

export function base58DecodeSessionID(encodedText) {
  const input = safeString(encodedText).trim();
  if (!input) {
    return "";
  }

  let zeros = 0;
  while (zeros < input.length && input[zeros] === "1") {
    zeros++;
  }

  const bytes = [];
  for (let i = zeros; i < input.length; i++) {
    const ch = input[i];
    const value = BASE58_ALPHABET.indexOf(ch);
    if (value < 0) {
      return "";
    }
    let carry = value;
    for (let j = 0; j < bytes.length; j++) {
      const num = bytes[j] * 58 + carry;
      bytes[j] = num & 0xff;
      carry = num >> 8;
    }
    while (carry > 0) {
      bytes.push(carry & 0xff);
      carry >>= 8;
    }
  }

  for (let i = 0; i < zeros; i++) {
    bytes.push(0);
  }
  bytes.reverse();
  return new TextDecoder().decode(new Uint8Array(bytes));
}

export function buildOutgoingBusinessPayload(contentType, data = {}) {
  const payload = { content_type: contentType };
  const replyTo = normalizeReplyTo(data.replyTo || data.reply_to);
  if (contentType === BUSINESS_CONTENT_TYPES.TEXT) {
    payload.content = safeString(data.content);
  } else {
    payload.url = safeString(data.url);
    payload.content = payload.url;

    if (data.name != null) {
      payload.name = safeString(data.name);
    }
    if (data.size != null) {
      payload.size = safeNumber(data.size, 0);
    }
    if (data.duration != null) {
      payload.duration = safeNumber(data.duration, 0);
    }
  }

  if (replyTo) {
    payload.reply_to = {
      msg_id: replyTo.msgId,
      content_type: replyTo.contentType,
      preview: replyTo.preview,
      sender: replyTo.sender,
      timestamp: replyTo.timestamp,
    };
  }
  return payload;
}

export function normalizeIncomingBusinessMessage(packet) {
  if (!packet || typeof packet !== "object") {
    return null;
  }

  const payload = packet.payload && typeof packet.payload === "object" ? packet.payload : {};
  const businessType = safeString(packet.type);
  const contentType = inferContentType(payload);
  const timestamp = normalizeUnixTs(packet.timestamp || payload.timestamp);

  const decodedSID = base58DecodeSessionID(packet.sid);
  const plainSID = safeString(packet.sid);

  return {
    businessType,
    sid: decodedSID || plainSID,
    timestamp,
    payload: {
      contentType,
      content: safeString(payload.content),
      url: safeString(payload.url),
      name: safeString(payload.name),
      size: safeNumber(payload.size, 0),
      duration: safeNumber(payload.duration, 0),
      from: safeString(payload.from),
      agentName: safeString(payload.agent_name || payload.agentName),
      fromName: safeString(payload.from_name || payload.fromName),
      senderName: safeString(payload.sender_name || payload.senderName),
      clientId: safeString(payload.client_id || payload.clientId),
      code: safeString(payload.code),
      msgId: safeString(payload.msg_id || packet.msg_id),
      replyTo: normalizeReplyTo(payload.reply_to || payload.replyTo),
      timestamp,
    },
    raw: packet,
  };
}

function normalizeBusinessInput(messageLike = {}) {
  const payload = messageLike.payload || {};
  const contentTypeRaw = safeString(payload.contentType || payload.content_type);
  const allowedContentTypes = new Set(Object.values(BUSINESS_CONTENT_TYPES));
  const contentType = allowedContentTypes.has(contentTypeRaw) ? contentTypeRaw : BUSINESS_CONTENT_TYPES.TEXT;
  return {
    businessType: safeString(messageLike.businessType || messageLike.type),
    sid: safeString(messageLike.sid),
    timestamp: normalizeUnixTs(messageLike.timestamp || payload.timestamp),
    payload: {
      contentType,
      content: safeString(payload.content),
      url: safeString(payload.url),
      name: safeString(payload.name),
      size: safeNumber(payload.size, 0),
      duration: safeNumber(payload.duration, 0),
      from: safeString(payload.from),
      agentName: safeString(payload.agentName || payload.agent_name),
      fromName: safeString(payload.fromName || payload.from_name),
      senderName: safeString(payload.senderName || payload.sender_name),
      msgId: safeString(payload.msgId || payload.msg_id),
      replyTo: normalizeReplyTo(payload.replyTo || payload.reply_to),
    },
  };
}

export function buildInboxUiMessageFromBusiness(messageLike = {}, options = {}) {
  const normalized = normalizeBusinessInput(messageLike);
  if (
    normalized.businessType === BUSINESS_MESSAGE_TYPES.TYPING ||
    normalized.businessType === BUSINESS_MESSAGE_TYPES.CLOSE
  ) {
    return null;
  }

  const contentType = normalized.payload.contentType;
  const url = normalized.payload.url || normalized.payload.content;
  const currentUserName = safeString(options.currentUserName || "").trim().toLowerCase();
  const senderName = safeString(
    normalized.payload.senderName || normalized.payload.fromName || normalized.payload.agentName
  ).trim();
  const senderKey = senderName.toLowerCase();
  const senderType =
    normalized.businessType === BUSINESS_MESSAGE_TYPES.SYSTEM
      ? "system"
      : normalized.businessType === BUSINESS_MESSAGE_TYPES.VISITOR
        ? "visitor"
        : senderKey && currentUserName && senderKey === currentUserName
          ? "self"
          : "agent";

  return {
    local_id: safeString(options.localId || createLocalId("in")),
    msg_id: normalized.payload.msgId,
    business_type: normalized.businessType,
    content_type: contentType,
    content:
      contentType === BUSINESS_CONTENT_TYPES.TEXT
        ? normalized.payload.content
        : contentType === BUSINESS_CONTENT_TYPES.AUDIO
          ? "[语音消息]"
          : normalized.payload.content || url,
    url,
    name: normalized.payload.name,
    size: normalized.payload.size,
    duration: normalized.payload.duration,
    replyTo: normalized.payload.replyTo || null,
    timestamp: normalized.timestamp,
    from: normalized.payload.from,
    sender_name: senderName,
    sender_type: senderType,
    isSelf: senderType === "self",
    sid: normalized.sid,
    status: UI_MESSAGE_STATUS.SENT,
  };
}

export function buildInboxUiMessageFromOutgoing(contentType, data = {}, options = {}) {
  const payload = buildOutgoingBusinessPayload(contentType, data);
  return buildInboxUiMessageFromBusiness(
    {
      businessType: BUSINESS_MESSAGE_TYPES.AGENT,
      sid: safeString(options.sid),
      timestamp: options.timestamp || Math.floor(Date.now() / 1000),
      payload: {
        contentType,
        content: payload.content,
        url: payload.url,
        name: payload.name,
        size: payload.size,
        duration: payload.duration,
        replyTo: payload.reply_to,
        senderName: safeString(options.currentUserName || ""),
      },
    },
    {
      localId: options.localId || createLocalId("out"),
      currentUserName: safeString(options.currentUserName || ""),
    }
  );
}

export function toInboxDisplayText(message = {}) {
  if (message.content_type === BUSINESS_CONTENT_TYPES.IMAGE) {
    return "[图片]";
  }
  if (message.content_type === BUSINESS_CONTENT_TYPES.AUDIO) {
    return "[语音]";
  }
  if (message.content_type === BUSINESS_CONTENT_TYPES.FILE) {
    return message.name ? `[文件] ${message.name}` : "[文件]";
  }
  return safeString(message.content);
}
