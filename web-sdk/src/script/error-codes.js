export const SDK_ERROR_MESSAGES = {
  13001: "Too many requests, please retry later",

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

  // legacy compatibility
  1003: "Current domain is not allowlisted. Contact admin.",
  2004: "Too many requests, please retry later",
};

function toCode(value) {
  const n = Number(value);
  return Number.isFinite(n) ? n : 0;
}

export function isDomainForbiddenCode(code, httpStatus) {
  const c = toCode(code);
  return c === 20014 || c === 24005 || c === 1003 || Number(httpStatus) === 403;
}

export function toSdkError(error, fallbackMessage = "Request failed") {
  const payload = error?.response?.data || {};
  const code = toCode(payload.code);
  const httpStatus = Number(error?.response?.status || 0);

  let message = SDK_ERROR_MESSAGES[code] || "";
  if (!message && typeof payload.msg === "string" && payload.msg.trim()) {
    message = payload.msg.trim();
  }
  if (!message) {
    if (httpStatus === 429) {
      message = "Too many requests, please retry later";
    } else if (httpStatus >= 500) {
      message = "Service unavailable, retry later";
    } else if (String(error?.message || "").includes("timeout")) {
      message = "Request timeout, retry later";
    } else if (String(error?.message || "").includes("Network Error")) {
      message = "Network error, check connection";
    }
  }
  if (!message) {
    message = fallbackMessage;
  }

  const wrapped = new Error(message);
  wrapped.name = "ApiError";
  wrapped.code = code;
  wrapped.httpStatus = httpStatus;
  wrapped.raw = payload;
  wrapped.cause = error;
  return wrapped;
}
