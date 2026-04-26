package controllers

import (
	"fmt"
	"html"
	"strings"

	"github.com/gin-gonic/gin"
)

func callbackPageLocale(c *gin.Context) string {
	if c == nil {
		return "zh-CN"
	}
	for _, raw := range []string{c.Query("lang"), c.Query("locale"), c.GetHeader("Accept-Language")} {
		value := strings.ToLower(strings.TrimSpace(raw))
		if value == "" {
			continue
		}
		if strings.HasPrefix(value, "en") {
			return "en-US"
		}
		if strings.HasPrefix(value, "zh") {
			return "zh-CN"
		}
	}
	return "zh-CN"
}

func callbackPageText(c *gin.Context, key string) string {
	texts := map[string]map[string]string{
		"zh-CN": {
			"page_title":           "绑定结果",
			"bind_success":         "绑定成功",
			"bind_success_named":   "%s 绑定成功",
			"bind_failed":          "绑定失败",
			"close_page":           "您可以关闭此页面。",
			"unsupported_channel":  "当前渠道暂不支持个人绑定。",
			"missing_params":       "缺少必要参数。",
			"bind_expired":         "二维码已过期，请重新扫码。",
			"channel_mismatch":     "绑定渠道不匹配。",
			"config_missing":       "通知配置缺失，请联系管理员。",
			"wecom_config_missing": "企业微信配置缺失，请联系管理员。",
			"fetch_user_failed":    "获取用户信息失败，请稍后重试。",
			"bind_save_failed":     "绑定保存失败，请稍后重试。",
		},
		"en-US": {
			"page_title":           "Binding Result",
			"bind_success":         "Binding successful",
			"bind_success_named":   "%s binding successful",
			"bind_failed":          "Binding failed",
			"close_page":           "You can close this page now.",
			"unsupported_channel":  "This channel does not support personal binding.",
			"missing_params":       "Missing required parameters.",
			"bind_expired":         "The QR code has expired. Please scan again.",
			"channel_mismatch":     "Binding channel mismatch.",
			"config_missing":       "Notification config is missing. Contact admin.",
			"wecom_config_missing": "WeCom config is missing. Contact admin.",
			"fetch_user_failed":    "Failed to fetch user info. Please retry later.",
			"bind_save_failed":     "Failed to save binding. Please retry later.",
		},
	}
	locale := callbackPageLocale(c)
	if msg, ok := texts[locale][key]; ok && msg != "" {
		return msg
	}
	return texts["zh-CN"][key]
}

func renderCallbackPage(c *gin.Context, httpStatus int, title string, detail string) {
	if c == nil {
		return
	}
	title = html.EscapeString(strings.TrimSpace(title))
	detail = html.EscapeString(strings.TrimSpace(detail))
	pageTitle := html.EscapeString(callbackPageText(c, "page_title"))

	body := fmt.Sprintf(`<!doctype html>
<html lang="%s">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>%s</title>
  <style>
    :root { color-scheme: light; }
    body {
      margin: 0;
      min-height: 100vh;
      display: grid;
      place-items: center;
      background: linear-gradient(180deg, #f7f8fa 0%%, #eef3f8 100%%);
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
      color: #1f2937;
    }
    .card {
      width: min(92vw, 420px);
      box-sizing: border-box;
      padding: 32px 28px;
      border-radius: 20px;
      background: #fff;
      box-shadow: 0 16px 48px rgba(15, 23, 42, 0.12);
      text-align: center;
    }
    h1 {
      margin: 0 0 12px;
      font-size: 24px;
      line-height: 1.3;
    }
    p {
      margin: 0;
      font-size: 15px;
      line-height: 1.7;
      color: #4b5563;
    }
  </style>
</head>
<body>
  <main class="card">
    <h1>%s</h1>
    <p>%s</p>
  </main>
</body>
</html>`, callbackPageLocale(c), pageTitle, title, detail)

	c.Data(httpStatus, "text/html; charset=utf-8", []byte(body))
}
