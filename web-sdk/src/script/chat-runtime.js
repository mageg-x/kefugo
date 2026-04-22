// chat-runtime.js
// chat runtime utilities: visitor id, URL params, welcome message, URL conversion.

export function createLocalMessageId(prefix = "msg") {
  return `${prefix}_${Date.now().toString(36)}_${Math.random().toString(36).slice(2, 10)}`;
}

const VISITOR_ID_SANITIZE_RE = /[^A-Za-z0-9_-]/g;

function sanitizeVisitorIdentity(raw) {
  const value = String(raw || "").trim();
  if (!value) {
    return "";
  }
  return value
    .replace(VISITOR_ID_SANITIZE_RE, "_")
    .replace(/_+/g, "_")
    .replace(/^_+|_+$/g, "");
}

function stripKnownVisitorPrefix(value) {
  if (value.startsWith("u_") || value.startsWith("v_")) {
    return value.slice(2);
  }
  return value;
}

function ensureVisitorPrefix(raw, prefix) {
  const sanitized = sanitizeVisitorIdentity(raw);
  if (!sanitized) {
    return "";
  }
  if (sanitized.startsWith(prefix)) {
    return sanitized;
  }
  const noPrefix = stripKnownVisitorPrefix(sanitized);
  if (!noPrefix) {
    return "";
  }
  return `${prefix}${noPrefix}`;
}

export function normalizeAppUserId(raw) {
  return ensureVisitorPrefix(raw, "u_");
}

export function normalizeSdkVisitorId(raw) {
  return ensureVisitorPrefix(raw, "v_");
}

export function getAppIdFromQuery(defaultAppId = "default") {
  try {
    const params = new URLSearchParams(window.location.search);
    return params.get("data-kefu-appid") || params.get("appId") || defaultAppId;
  } catch {
    return defaultAppId;
  }
}

function readFirstNonEmpty(params, keys = []) {
  for (const key of keys) {
    const value = params.get(key);
    if (typeof value === "string" && value.trim()) {
      return value.trim();
    }
  }
  return "";
}

export function getChatInitOptionsFromQuery(defaultAppId = "default") {
  try {
    const params = new URLSearchParams(window.location.search);
    return {
      appId:
        readFirstNonEmpty(params, ["data-kefu-appid", "appId", "appid"]) ||
        defaultAppId,
      userId: readFirstNonEmpty(params, [
        "data-kefu-user-id",
        "userId",
        "userid",
        "visitorId",
        "visitor_id",
        "uid",
      ]),
      apiBaseUrl: readFirstNonEmpty(params, [
        "data-kefu-api-base-url",
        "apiBaseUrl",
        "api_base_url",
        "api",
      ]),
      wsUrl: readFirstNonEmpty(params, [
        "data-kefu-ws-url",
        "wsUrl",
        "ws_url",
        "ws",
      ]),
    };
  } catch {
    return {
      appId: defaultAppId,
      userId: "",
      apiBaseUrl: "",
      wsUrl: "",
    };
  }
}

function generateMachineFingerprint() {
  const rand = Math.random().toString(16).slice(2, 10);
  const now = Date.now().toString(16);
  return `v_${now}_${rand}`;
}

export function getOrCreateVisitorId(storageKey = "zerospace_kefu_user_id") {
  let visitorId = "";

  if (typeof window !== "undefined" && window.localStorage) {
    visitorId = localStorage.getItem(storageKey) || "";
  }

  if (!visitorId && typeof document !== "undefined") {
    const cookieMatch = document.cookie.match(new RegExp(`${storageKey}=([^;]+)`));
    if (cookieMatch) {
      visitorId = decodeURIComponent(cookieMatch[1]);
    }
  }

  if (visitorId) {
    const normalized = normalizeSdkVisitorId(visitorId);
    if (normalized) {
      if (normalized !== visitorId) {
        if (typeof window !== "undefined" && window.localStorage) {
          localStorage.setItem(storageKey, normalized);
        }
        if (typeof document !== "undefined") {
          const expires = new Date();
          expires.setTime(expires.getTime() + 10 * 365 * 24 * 60 * 60 * 1000);
          document.cookie = `${storageKey}=${encodeURIComponent(normalized)};expires=${expires.toUTCString()};path=/`;
        }
      }
      return normalized;
    }
  }

  visitorId = normalizeSdkVisitorId(generateMachineFingerprint()) || generateMachineFingerprint();

  if (typeof window !== "undefined" && window.localStorage) {
    localStorage.setItem(storageKey, visitorId);
  }

  if (typeof document !== "undefined") {
    const expires = new Date();
    expires.setTime(expires.getTime() + 10 * 365 * 24 * 60 * 60 * 1000);
    document.cookie = `${storageKey}=${encodeURIComponent(visitorId)};expires=${expires.toUTCString()};path=/`;
  }

  return visitorId;
}

export function buildWsUrlFromApiBase(apiBaseUrl = "") {
  const fallbackBase =
    typeof window !== "undefined" && window.location?.origin
      ? window.location.origin
      : "http://localhost:5300";
  const finalBase = apiBaseUrl || fallbackBase;
  try {
    const base = new URL(finalBase);
    const protocol = base.protocol === "https:" ? "wss:" : "ws:";
    return `${protocol}//${base.host}/ws/chat`;
  } catch {
    return "ws://localhost:5300/ws/chat";
  }
}

export function buildWelcomeMessages(config = {}) {
  const appName = config.name || "Agent";
  const now = new Date().toISOString();
  const messages = [
    {
      id: createLocalMessageId("welcome"),
      type: "system",
      isSelf: false,
      name: "System",
      content: `${appName} is serving you`,
      time: now,
      contentType: "text",
    },
  ];

  if (config.welcome_msg) {
    messages.push({
      id: createLocalMessageId("welcome_text"),
      type: "text",
      isSelf: false,
      name: appName,
      avatar: config.logo || "",
      content: String(config.welcome_msg),
      time: now,
      contentType: "text",
      status: "sent",
    });
  }

  return messages;
}
