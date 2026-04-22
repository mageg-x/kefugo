// message-protocol.js
// Manage protocol-to-UI mapping in one place to avoid mixing business and display types.

export const BUSINESS_MESSAGE_TYPES = {
  VISITOR: "message.visitor",
  AGENT: "message.agent",
  SYSTEM: "message.system",
  TYPING: "message.typing",
  ACK: "message.ack",
  CLOSE: "message.close",
};

export const BUSINESS_CONTENT_TYPES = {
  TEXT: "text",
  IMAGE: "image",
  AUDIO: "audio",
  FILE: "file",
};

export const CHAT_UI_MESSAGE_TYPES = {
  TEXT: "text",
  IMAGE: "image",
  AUDIO: "audio",
  FILE: "file",
  SYSTEM: "system",
};

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

function normalizeReplyTo(rawReply) {
  if (!rawReply || typeof rawReply !== "object" || Array.isArray(rawReply)) {
    return null;
  }
  const msgId = safeString(rawReply.msg_id || rawReply.msgId);
  const contentType = safeString(rawReply.content_type || rawReply.contentType || BUSINESS_CONTENT_TYPES.TEXT);
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

function inferContentType(rawPayload = {}) {
  const rawType = safeString(rawPayload.content_type || rawPayload.contentType).toLowerCase();
  const allowed = new Set(Object.values(BUSINESS_CONTENT_TYPES));
  if (allowed.has(rawType)) {
    return rawType;
  }
  const url = safeString(rawPayload.url || rawPayload.content).toLowerCase();
  const name = safeString(rawPayload.name).toLowerCase();
  const target = `${url} ${name}`;
  if (/\.(png|jpe?g|gif|webp|bmp|svg)(\?|#|$)/.test(target)) return BUSINESS_CONTENT_TYPES.IMAGE;
  if (/\.(mp3|wav|ogg|m4a|aac|webm)(\?|#|$)/.test(target)) return BUSINESS_CONTENT_TYPES.AUDIO;
  if (rawPayload.duration != null && Number(rawPayload.duration) > 0) return BUSINESS_CONTENT_TYPES.AUDIO;
  if (rawPayload.size != null || safeString(rawPayload.name)) return BUSINESS_CONTENT_TYPES.FILE;
  return BUSINESS_CONTENT_TYPES.TEXT;
}

function inferFileTypeByName(fileName = "") {
  const lower = safeString(fileName).toLowerCase();
  if (lower.endsWith(".pdf")) return "application/pdf";
  if (lower.endsWith(".txt")) return "text/plain";
  if (lower.endsWith(".doc")) return "application/msword";
  if (lower.endsWith(".docx")) {
    return "application/vnd.openxmlformats-officedocument.wordprocessingml.document";
  }
  if (lower.endsWith(".xls")) return "application/vnd.ms-excel";
  if (lower.endsWith(".xlsx")) {
    return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet";
  }
  if (lower.endsWith(".zip")) return "application/zip";
  if (lower.endsWith(".rar")) return "application/vnd.rar";
  return "application/octet-stream";
}

function toIsoTime(value) {
  if (!value) {
    return new Date().toISOString();
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return new Date().toISOString();
  }
  return date.toISOString();
}

export function buildOutgoingBusinessPayload(contentType, data = {}) {
  const payload = {
    content_type: contentType,
  };
  const replyTo = normalizeReplyTo(data.replyTo || data.reply_to);

  switch (contentType) {
    case BUSINESS_CONTENT_TYPES.IMAGE:
      payload.url = safeString(data.url);
      payload.name = safeString(data.name || "image.jpg");
      payload.content = payload.url;
      break;
    case BUSINESS_CONTENT_TYPES.AUDIO:
      payload.url = safeString(data.url);
      payload.duration = safeNumber(data.duration, 0);
      payload.content = payload.url;
      break;
    case BUSINESS_CONTENT_TYPES.FILE:
      payload.url = safeString(data.url);
      payload.name = safeString(data.name || "file");
      payload.size = safeNumber(data.size, 0);
      payload.content = payload.url;
      break;
    case BUSINESS_CONTENT_TYPES.TEXT:
    default:
      payload.content = safeString(data.content);
      break;
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

export function parseBusinessPayload(rawPayload) {
  if (typeof rawPayload === "string") {
    try {
      const parsed = JSON.parse(rawPayload);
      return parseBusinessPayload(parsed);
    } catch {
      return {
        contentType: BUSINESS_CONTENT_TYPES.TEXT,
        content: safeString(rawPayload),
        url: "",
        name: "",
        duration: 0,
        size: 0,
        raw: rawPayload,
      };
    }
  }

  if (!rawPayload || typeof rawPayload !== "object" || Array.isArray(rawPayload)) {
    return {
      contentType: BUSINESS_CONTENT_TYPES.TEXT,
      content: "",
      url: "",
      name: "",
      duration: 0,
      size: 0,
      raw: rawPayload,
    };
  }

  const contentType = inferContentType(rawPayload);

  return {
    contentType,
    content: safeString(rawPayload.content),
    url: safeString(rawPayload.url),
    name: safeString(rawPayload.name),
    from: safeString(rawPayload.from),
    agentName: safeString(rawPayload.agent_name || rawPayload.agentName),
    fromName: safeString(rawPayload.from_name || rawPayload.fromName),
    senderName: safeString(rawPayload.sender_name || rawPayload.senderName),
    replyTo: normalizeReplyTo(rawPayload.reply_to || rawPayload.replyTo),
    duration: safeNumber(rawPayload.duration, 0),
    size: safeNumber(rawPayload.size, 0),
    raw: rawPayload,
  };
}

export function normalizeIncomingBusinessMessage(rawMessage) {
  if (!rawMessage || typeof rawMessage !== "object") {
    return null;
  }

  const businessType = safeString(rawMessage.type);
  const payload = parseBusinessPayload(rawMessage.payload);

  return {
    businessType,
    payload,
    sid: safeString(rawMessage.sid),
    clientId: safeString(rawMessage.payload?.client_id || rawMessage.payload?.clientId || ""),
    code: safeString(rawMessage.payload?.code || ""),
    msgId: safeString(rawMessage.msg_id || rawMessage.payload?.msg_id || ""),
    timestamp: rawMessage.timestamp || Date.now(),
    raw: rawMessage,
  };
}

function buildUiPayloadMeta(payload) {
  switch (payload.contentType) {
    case BUSINESS_CONTENT_TYPES.IMAGE: {
      const imageUrl = safeString(payload.url || payload.content);
      if (!imageUrl) {
        return {
          type: CHAT_UI_MESSAGE_TYPES.TEXT,
          content: safeString(payload.content),
          contentType: BUSINESS_CONTENT_TYPES.TEXT,
        };
      }
      return {
        type: CHAT_UI_MESSAGE_TYPES.IMAGE,
        content: imageUrl,
        contentType: BUSINESS_CONTENT_TYPES.IMAGE,
      };
    }
    case BUSINESS_CONTENT_TYPES.AUDIO: {
      const audioUrl = safeString(payload.url || payload.content);
      if (!audioUrl) {
        return {
          type: CHAT_UI_MESSAGE_TYPES.TEXT,
          content: "[Voice message]",
          contentType: BUSINESS_CONTENT_TYPES.TEXT,
        };
      }
      return {
        type: CHAT_UI_MESSAGE_TYPES.AUDIO,
        content: "[Voice message]",
        contentType: BUSINESS_CONTENT_TYPES.AUDIO,
        audioUrl,
        duration: safeNumber(payload.duration, 0),
      };
    }
    case BUSINESS_CONTENT_TYPES.FILE: {
      const fileName = safeString(payload.name || "file");
      return {
        type: CHAT_UI_MESSAGE_TYPES.FILE,
        content: fileName,
        contentType: BUSINESS_CONTENT_TYPES.FILE,
        fileName,
        fileUrl: safeString(payload.url || payload.content),
        fileSize: safeNumber(payload.size, 0),
        fileType: inferFileTypeByName(fileName),
      };
    }
    case BUSINESS_CONTENT_TYPES.TEXT:
    default:
      return {
        type: CHAT_UI_MESSAGE_TYPES.TEXT,
        content: safeString(payload.content),
        contentType: BUSINESS_CONTENT_TYPES.TEXT,
      };
  }
}

export function buildChatUiMessageFromBusiness(message, options = {}) {
  if (!message) {
    return null;
  }

  const payload = message.payload || parseBusinessPayload("");
  const id = safeString(options.id || "");
  const serviceName = safeString(options.serviceName || "Agent");
  const serviceAvatar = safeString(options.serviceAvatar || "");
  const selfName = safeString(options.selfName || "Me");
  const selfAvatar = safeString(options.selfAvatar || "");
  const time = toIsoTime(message.timestamp || Date.now());

  if (message.businessType === BUSINESS_MESSAGE_TYPES.SYSTEM) {
    return {
      id,
      type: CHAT_UI_MESSAGE_TYPES.SYSTEM,
      isSelf: false,
      name: "System",
      content: safeString(payload.content || "System message"),
      contentType: BUSINESS_CONTENT_TYPES.TEXT,
      time,
      business: message,
    };
  }

  if (
    message.businessType !== BUSINESS_MESSAGE_TYPES.AGENT &&
    message.businessType !== BUSINESS_MESSAGE_TYPES.VISITOR
  ) {
    return null;
  }

  const uiMeta = buildUiPayloadMeta(payload);
  const isSelf = message.businessType === BUSINESS_MESSAGE_TYPES.VISITOR;
  const payloadAgentName = safeString(payload.agentName || payload.fromName || payload.senderName);
  const targetName = payloadAgentName || serviceName;

  return {
    id,
    ...uiMeta,
    isSelf,
    name: isSelf ? selfName : targetName,
    avatar: isSelf ? selfAvatar : serviceAvatar,
    replyTo: payload.replyTo || null,
    time,
    status: "sent",
    business: message,
  };
}

export function buildChatUiMessageFromOutgoingPayload(payload, options = {}) {
  const normalizedPayload = parseBusinessPayload(payload);
  const uiMeta = buildUiPayloadMeta(normalizedPayload);

  return {
    id: safeString(options.id || ""),
    ...uiMeta,
    isSelf: true,
    name: safeString(options.selfName || "Me"),
    avatar: safeString(options.selfAvatar || ""),
    replyTo: normalizedPayload.replyTo || null,
    time: toIsoTime(options.time || Date.now()),
    status: "sent",
  };
}

// compat old caller naming
export const mapIncomingBusinessToUiMessage = buildChatUiMessageFromBusiness;
export const mapOutgoingPayloadToUiMessage = buildChatUiMessageFromOutgoingPayload;

export function getImageUrlsFromUiMessage(message) {
  if (!message) {
    return [];
  }

  if (message.type === CHAT_UI_MESSAGE_TYPES.IMAGE && safeString(message.content)) {
    return [safeString(message.content)];
  }
  return [];
}

export function isVoiceUiMessage(message) {
  if (!message) {
    return false;
  }

  if (message.type === CHAT_UI_MESSAGE_TYPES.AUDIO) {
    return true;
  }

  if (message.contentType === BUSINESS_CONTENT_TYPES.AUDIO) {
    return true;
  }

  return false;
}
