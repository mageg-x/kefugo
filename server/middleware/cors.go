package middleware

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"

	"kefu-server/models"
	"kefu-server/utils/logger"
)

// CORS 处理跨域请求。
// 策略：同源放行；带 appid/app_id 时按应用白名单校验来源域名。
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := strings.TrimSpace(c.GetHeader("Origin"))
		allowOrigin := ""
		if origin != "" {
			if isSameOrigin(origin, c.Request.Host) {
				allowOrigin = origin
			}
			if appID := strings.TrimSpace(c.Query("appid")); appID != "" {
				app := models.GetApp(appID)
				if app != nil && models.IsDomainAllowed(origin, "", app.AllowDomain) {
					allowOrigin = origin
				}
			}
			if appID := strings.TrimSpace(c.Query("app_id")); appID != "" {
				app := models.GetApp(appID)
				if app != nil && models.IsDomainAllowed(origin, "", app.AllowDomain) {
					allowOrigin = origin
				}
			}
			if appID := strings.TrimSpace(c.PostForm("app_id")); appID != "" {
				app := models.GetApp(appID)
				if app != nil && models.IsDomainAllowed(origin, "", app.AllowDomain) {
					allowOrigin = origin
				}
			}
			if appID := strings.TrimSpace(c.PostForm("appid")); appID != "" {
				app := models.GetApp(appID)
				if app != nil && models.IsDomainAllowed(origin, "", app.AllowDomain) {
					allowOrigin = origin
				}
			}
		}
		if allowOrigin != "" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", allowOrigin)
			c.Writer.Header().Set("Vary", "Origin")
		}
		// 允许的请求头（含 WebSocket upgrade 相关头）。
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, X-Encryption-Enabled, X-Device-ID, Upgrade, Connection, Sec-WebSocket-Key, Sec-WebSocket-Version, Sec-WebSocket-Protocol, Sec-WebSocket-Extensions")
		// 允许的请求方法。
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")
		// 允许前端读取的响应头。
		c.Writer.Header().Set("Access-Control-Expose-Headers", "Content-Length")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")

		// 预检请求直接处理。
		if c.Request.Method == "OPTIONS" {
			if allowOrigin == "" {
				logger.Errorf("cors preflight forbidden origin=%s host=%s", origin, c.Request.Host)
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func isSameOrigin(origin, requestHost string) bool {
	parsed, err := url.Parse(origin)
	if err != nil {
		logger.Errorf("cors origin parse failed origin=%s", origin)
		return false
	}
	if parsed.Host == "" || requestHost == "" {
		return false
	}
	return strings.EqualFold(parsed.Host, requestHost)
}
