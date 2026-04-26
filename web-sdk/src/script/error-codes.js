import { localeRef } from "./i18n";

const SDK_ERROR_MESSAGES = {
  "zh-CN": {
    10001: "参数错误",
    10005: "服务器错误，请稍后重试",
    10008: "请求超时，请稍后重试",
    13001: "请求过于频繁，请稍后再试",

    20004: "应用不存在",
    20014: "当前域名未在白名单，请联系管理员",
    20017: "应用不存在或已禁用",

    22003: "会话不存在，请刷新后重试",
    22006: "会话已关闭",

    24003: "缺少 app_id",
    24005: "当前域名不允许上传",
    24006: "请选择上传文件",
    24014: "文件过大，请压缩后重试",
    24015: "文件类型不允许",

    1003: "当前域名未在白名单，请联系管理员",
    2004: "请求过于频繁，请稍后再试",
  },
  "en-US": {
    10001: "Invalid parameters",
    10005: "Server error. Please try again later",
    10008: "Request timeout. Please try again later",
    13001: "Too many requests. Please retry later",

    20004: "App not found",
    20014: "Current domain is not allowlisted. Contact admin.",
    20017: "App not found or disabled",

    22003: "Session not found, refresh and retry",
    22006: "Session is closed",

    24003: "Missing app_id",
    24005: "Upload is not allowed for current domain",
    24006: "Please select a file to upload",
    24014: "File is too large, compress and retry",
    24015: "File type is not allowed",

    1003: "Current domain is not allowlisted. Contact admin.",
    2004: "Too many requests. Please retry later",
  },
};

const FALLBACK_MESSAGES = {
  "zh-CN": {
    requestFailed: "请求失败",
    tooManyRequests: "请求过于频繁，请稍后再试",
    serverRetry: "服务开小差了，请稍后重试",
    timeout: "请求超时，请稍后重试",
    network: "网络连接失败，请检查网络",
  },
  "en-US": {
    requestFailed: "Request failed",
    tooManyRequests: "Too many requests, please retry later",
    serverRetry: "Service unavailable, retry later",
    timeout: "Request timeout, retry later",
    network: "Network error, check connection",
  },
};

function toCode(value) {
  const n = Number(value);
  return Number.isFinite(n) ? n : 0;
}

function getLocaleMessages() {
  return SDK_ERROR_MESSAGES[localeRef.value] || SDK_ERROR_MESSAGES["zh-CN"];
}

function getFallbackMessages() {
  return FALLBACK_MESSAGES[localeRef.value] || FALLBACK_MESSAGES["zh-CN"];
}

export function isDomainForbiddenCode(code, httpStatus) {
  const c = toCode(code);
  return c === 20014 || c === 24005 || c === 1003 || Number(httpStatus) === 403;
}

export function toSdkError(error, fallbackMessage = "") {
  const payload = error?.response?.data || {};
  const code = toCode(payload.code);
  const httpStatus = Number(error?.response?.status || 0);
  const fallbackMessages = getFallbackMessages();

  const localizedByCode = getLocaleMessages()[code] || "";
  const payloadMsg = typeof payload.msg === "string" ? payload.msg.trim() : "";

  let message = localizedByCode;
  if (!message && payloadMsg && code !== 0) {
    message = payloadMsg;
  }
  if (!message) {
    if (httpStatus === 429) {
      message = fallbackMessages.tooManyRequests;
    } else if (httpStatus >= 500) {
      message = fallbackMessages.serverRetry;
    } else if (String(error?.message || "").includes("timeout")) {
      message = fallbackMessages.timeout;
    } else if (String(error?.message || "").includes("Network Error")) {
      message = fallbackMessages.network;
    }
  }
  if (!message) {
    message = fallbackMessage || fallbackMessages.requestFailed;
  }

  const wrapped = new Error(message);
  wrapped.name = "ApiError";
  wrapped.code = code;
  wrapped.httpStatus = httpStatus;
  wrapped.raw = payload;
  wrapped.cause = error;
  return wrapped;
}
