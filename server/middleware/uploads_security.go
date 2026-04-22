package middleware

import (
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// UploadsSecurity 为 /uploads 静态资源增加安全响应头，避免同源脚本执行。
func UploadsSecurity() gin.HandlerFunc {
	return func(c *gin.Context) {
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(c.Request.URL.Path)), "/uploads/") {
			name := filepath.Base(strings.TrimSpace(c.Request.URL.Path))
			name = strings.ReplaceAll(name, "\"", "")
			c.Header("X-Content-Type-Options", "nosniff")
			c.Header("Content-Security-Policy", "default-src 'none'; img-src 'self' data: blob:; media-src 'self' data: blob:; style-src 'none'; script-src 'none'; frame-ancestors 'none'")
			c.Header("X-Frame-Options", "DENY")
			c.Header("Referrer-Policy", "no-referrer")
			c.Header("Content-Disposition", "attachment; filename=\""+name+"\"")
		}
		c.Next()
	}
}
