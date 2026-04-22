export const ERROR_CODE_MESSAGES = {
  0: "成功",

  11001: "请先登录",
  11002: "登录凭证格式错误",
  11003: "登录状态无效，请重新登录",
  11004: "登录已过期，请重新登录",
  11007: "登录上下文缺失，请重新登录",
  11008: "用户名或密码错误",
  11009: "登录参数错误",

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
  31508: "模型配置无效",
  31509: "模型提供方无效",

  // legacy compatibility
  1002: "请先登录",
  1003: "权限不足",
  2001: "用户名或密码错误",
  2002: "登录已过期，请重新登录",
  2003: "登录状态无效，请重新登录",
  2004: "请求过于频繁，请稍后再试",
};

const AUTH_RESET_CODES = new Set([11001, 11002, 11003, 11004, 11006, 11007, 1002, 2002, 2003]);

function normalizeCode(code) {
  const n = Number(code);
  return Number.isFinite(n) ? n : 0;
}

export function shouldResetAuth(code) {
  return AUTH_RESET_CODES.has(normalizeCode(code));
}

export function getErrorMessageByCode(code, fallback = "请求失败") {
  const normalized = normalizeCode(code);
  return ERROR_CODE_MESSAGES[normalized] || fallback;
}

export function toApiError(error, defaultMessage = "请求失败") {
  const payload = error?.response?.data || {};
  const code = normalizeCode(payload.code);
  const httpStatus = Number(error?.response?.status || 0);

  const payloadMsg = typeof payload.msg === "string" ? payload.msg.trim() : "";
  const networkMsg = typeof error?.message === "string" ? error.message : "";
  const normalizedPayloadMsg = payloadMsg.toLowerCase();
  const payloadLooksLikeSuccess = normalizedPayloadMsg === "success" || payloadMsg === "成功";

  let message = "";
  if (code === 31507 && payloadMsg) {
    const base = getErrorMessageByCode(code, "模型运行时库未就绪");
    message = `${base}：${payloadMsg}`;
  } else if (code === 31506 && payloadMsg) {
    const base = getErrorMessageByCode(code, "模型推理失败");
    message = `${base}：${payloadMsg}`;
  } else if (payloadMsg && !payloadLooksLikeSuccess && code !== 0) {
    message = payloadMsg;
  } else if (code !== 0) {
    message = getErrorMessageByCode(code, "");
  }
  if (!message) {
    if (httpStatus === 401) {
      message = "请先登录";
    } else if (httpStatus === 403) {
      message = "权限不足";
    } else if (httpStatus === 429) {
      message = "请求过于频繁，请稍后再试";
    } else if (httpStatus >= 500) {
      message = "服务开小差了，请稍后重试";
    } else if (networkMsg.includes("timeout")) {
      message = "请求超时，请稍后重试";
    } else if (networkMsg.includes("Network Error")) {
      message = "网络连接失败，请检查网络";
    }
  }
  if (!message) {
    message = defaultMessage;
  }

  const wrapped = new Error(message);
  wrapped.name = "ApiError";
  wrapped.code = code;
  wrapped.httpStatus = httpStatus;
  wrapped.raw = payload;
  wrapped.cause = error;
  return wrapped;
}
