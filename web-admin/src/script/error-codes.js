import { localeRef } from "./i18n";

const ERROR_CODE_MESSAGES = {
  "zh-CN": {
    10001: "参数错误",
    10005: "服务器错误，请稍后重试",
    10008: "请求超时，请稍后重试",

    11000: "请先登录",
    11001: "请先登录",
    11002: "登录凭证格式错误",
    11003: "登录状态无效，请重新登录",
    11004: "登录已过期，请重新登录",
    11007: "登录上下文缺失，请重新登录",
    11008: "用户名或密码错误",
    11009: "登录参数错误",

    12000: "权限不足",
    12001: "角色权限不足",
    12002: "需要管理员权限",
    12003: "需要客服权限",

    13001: "请求过于频繁，请稍后再试",
    13005: "请输入验证码",
    13006: "验证码错误",
    13008: "当前IP不允许登录",
    13009: "请求超时，请稍后重试",

    20002: "AppID 已存在",
    20004: "应用不存在",
    20014: "当前域名未在白名单",
    20017: "应用不存在或已禁用",

    21010: "创建用户参数错误",
    21025: "用户不存在",

    22003: "会话不存在",
    22004: "无权访问该会话",
    22005: "会话已被其他客服接待",
    22006: "会话已关闭",
    22010: "转接目标客服不可用",

    24003: "缺少 app_id",
    24005: "上传来源域名不允许",
    24006: "请选择上传文件",
    24014: "文件大小超出限制",
    24015: "文件类型不允许",

    25004: "快捷回复不存在",

    26000: "审计日志加载失败",
    27000: "看板加载失败",

    31003: "知识库不存在",
    31105: "文档不存在",
    31204: "片段不存在",
    31301: "检索失败",
    31401: "问答失败",

    31500: "模型配置加载失败",
    31501: "模型配置参数错误",
    31502: "模型配置保存失败",
    31503: "模型不存在，请先配置",
    31504: "模型配置无效",
    31505: "切换模型失败",
    31506: "模型推理失败",
    31507: "模型运行时库未就绪",
    31508: "模型路径无效",
    31509: "模型提供方无效",

    32000: "当前通知渠道暂不支持个人绑定",
    32001: "通知渠道未配置或未启用，请联系管理员",
    32002: "缺少绑定状态参数",
    32003: "绑定二维码已过期，请重新获取",
    32004: "绑定渠道不匹配",
    32005: "加载绑定信息失败",
    32006: "解绑通知渠道失败",
    32007: "通知渠道绑定失败",

    32100: "企业微信配置参数错误",
    32101: "企业微信配置序列化失败",
    32102: "保存企业微信配置失败",
    32103: "更新企业微信配置失败",
    32104: "CorpID 和 Secret 不能为空",
    32105: "企业微信功能未配置，请联系管理员",
    32106: "缺少绑定状态参数",
    32107: "绑定二维码已过期，请重新获取",
    32108: "加载企业微信绑定信息失败",
    32109: "解绑企业微信失败",
    32110: "企业微信回调参数错误",
    32111: "企业微信回调处理失败",
    32112: "企业微信绑定失败",

    1002: "请先登录",
    1003: "权限不足",
    2001: "用户名或密码错误",
    2002: "登录已过期，请重新登录",
    2003: "登录状态无效，请重新登录",
    2004: "请求过于频繁，请稍后再试",
  },
  "en-US": {
    10001: "Invalid parameters",
    10005: "Server error. Please try again later",
    10008: "Request timeout. Please try again later",

    11000: "Please sign in first",
    11001: "Please sign in first",
    11002: "Invalid auth token format",
    11003: "Invalid login state. Please sign in again",
    11004: "Login expired. Please sign in again",
    11007: "Missing auth context. Please sign in again",
    11008: "Incorrect username or password",
    11009: "Invalid login parameters",

    12000: "Permission denied",
    12001: "Insufficient role permissions",
    12002: "Admin permission required",
    12003: "Agent permission required",

    13001: "Too many requests. Please retry later",
    13005: "Captcha is required",
    13006: "Invalid captcha",
    13008: "Current IP is not allowed to sign in",
    13009: "Request timeout. Please try again later",

    20002: "AppID already exists",
    20004: "App not found",
    20014: "Current domain is not allowlisted",
    20017: "App not found or disabled",

    21010: "Invalid user creation parameters",
    21025: "User not found",

    22003: "Session not found",
    22004: "You do not have access to this session",
    22005: "This session has already been accepted by another agent",
    22006: "Session is closed",
    22010: "Target agent is unavailable for transfer",

    24003: "Missing app_id",
    24005: "Upload is not allowed for current domain",
    24006: "Please select a file to upload",
    24014: "File is too large",
    24015: "File type is not allowed",

    25004: "Quick reply not found",

    26000: "Failed to load audit logs",
    27000: "Failed to load dashboard",

    31003: "Knowledge base not found",
    31105: "Document not found",
    31204: "Chunk not found",
    31301: "Retrieval failed",
    31401: "Q&A failed",

    31500: "Failed to load model config",
    31501: "Invalid model config parameters",
    31502: "Failed to save model config",
    31503: "Model not found. Configure one first",
    31504: "Invalid model config",
    31505: "Failed to switch model",
    31506: "Model inference failed",
    31507: "Model runtime is not ready",
    31508: "Invalid model path",
    31509: "Invalid model provider",

    32000: "This notification channel does not support personal binding",
    32001: "Notification channel is not configured or enabled",
    32002: "Missing bind state parameter",
    32003: "Binding QR code expired. Please request a new one",
    32004: "Binding channel mismatch",
    32005: "Failed to load binding info",
    32006: "Failed to unbind notification channel",
    32007: "Failed to bind notification channel",

    32100: "Invalid WeCom config parameters",
    32101: "Failed to serialize WeCom config",
    32102: "Failed to save WeCom config",
    32103: "Failed to update WeCom config",
    32104: "CorpID and Secret are required",
    32105: "WeCom is not configured. Contact admin",
    32106: "Missing bind state parameter",
    32107: "Binding QR code expired. Please request a new one",
    32108: "Failed to load WeCom binding info",
    32109: "Failed to unbind WeCom",
    32110: "Invalid WeCom callback parameters",
    32111: "Failed to handle WeCom callback",
    32112: "Failed to bind WeCom",

    1002: "Please sign in first",
    1003: "Permission denied",
    2001: "Incorrect username or password",
    2002: "Login expired. Please sign in again",
    2003: "Invalid login state. Please sign in again",
    2004: "Too many requests. Please retry later",
  },
};

const FALLBACK_MESSAGES = {
  "zh-CN": {
    requestFailed: "请求失败",
    unauthorized: "请先登录",
    forbidden: "权限不足",
    tooManyRequests: "请求过于频繁，请稍后再试",
    serverRetry: "服务开小差了，请稍后重试",
    timeout: "请求超时，请稍后重试",
    network: "网络连接失败，请检查网络",
  },
  "en-US": {
    requestFailed: "Request failed",
    unauthorized: "Please sign in first",
    forbidden: "Permission denied",
    tooManyRequests: "Too many requests. Please retry later",
    serverRetry: "Service unavailable. Please retry later",
    timeout: "Request timeout. Please try again later",
    network: "Network error. Please check your connection",
  },
};

const AUTH_RESET_CODES = new Set([11001, 11002, 11003, 11004, 11006, 11007, 1002, 2002, 2003]);

function normalizeCode(code) {
  const n = Number(code);
  return Number.isFinite(n) ? n : 0;
}

function getLocaleMessages() {
  return ERROR_CODE_MESSAGES[localeRef.value] || ERROR_CODE_MESSAGES["zh-CN"];
}

function getFallbackMessages() {
  return FALLBACK_MESSAGES[localeRef.value] || FALLBACK_MESSAGES["zh-CN"];
}

export function shouldResetAuth(code) {
  return AUTH_RESET_CODES.has(normalizeCode(code));
}

export function getErrorMessageByCode(code, fallback = "") {
  const normalized = normalizeCode(code);
  const localized = getLocaleMessages()[normalized];
  if (localized) {
    return localized;
  }
  return fallback || getFallbackMessages().requestFailed;
}

export function toApiError(error, defaultMessage = "") {
  const payload = error?.response?.data || {};
  const code = normalizeCode(payload.code);
  const httpStatus = Number(error?.response?.status || 0);
  const fallbackMessages = getFallbackMessages();

  const payloadMsg = typeof payload.msg === "string" ? payload.msg.trim() : "";
  const networkMsg = typeof error?.message === "string" ? error.message : "";
  const normalizedPayloadMsg = payloadMsg.toLowerCase();
  const payloadLooksLikeSuccess = normalizedPayloadMsg === "success" || payloadMsg === "成功";

  const localizedByCode = code !== 0 ? getErrorMessageByCode(code, "") : "";

  let message = "";
  if (code === 31507 && payloadMsg) {
    message = `${localizedByCode || getErrorMessageByCode(code)}：${payloadMsg}`;
  } else if (code === 31506 && payloadMsg) {
    message = `${localizedByCode || getErrorMessageByCode(code)}：${payloadMsg}`;
  } else if (localizedByCode) {
    message = localizedByCode;
  } else if (payloadMsg && !payloadLooksLikeSuccess && code !== 0) {
    message = payloadMsg;
  }
  if (!message) {
    if (httpStatus === 401) {
      message = fallbackMessages.unauthorized;
    } else if (httpStatus === 403) {
      message = fallbackMessages.forbidden;
    } else if (httpStatus === 429) {
      message = fallbackMessages.tooManyRequests;
    } else if (httpStatus >= 500) {
      message = fallbackMessages.serverRetry;
    } else if (networkMsg.includes("timeout")) {
      message = fallbackMessages.timeout;
    } else if (networkMsg.includes("Network Error")) {
      message = fallbackMessages.network;
    }
  }
  if (!message) {
    message = defaultMessage || fallbackMessages.requestFailed;
  }

  const wrapped = new Error(message);
  wrapped.name = "ApiError";
  wrapped.code = code;
  wrapped.httpStatus = httpStatus;
  wrapped.raw = payload;
  wrapped.cause = error;
  return wrapped;
}
