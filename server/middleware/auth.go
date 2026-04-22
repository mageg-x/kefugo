package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"kefu-server/utils"
	"kefu-server/utils/logger"
	"kefu-server/utils/response"
)

// AuthMiddleware 负责解析并校验 JWT，并把用户上下文写入 gin.Context。
// 校验失败时直接终止请求链路并返回统一错误码。
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		tokenStr := ""
		if authHeader != "" {
			// 仅接受 Bearer token 格式。
			if strings.HasPrefix(authHeader, "Bearer ") {
				tokenStr = authHeader[7:]
			} else {
				logger.Errorf("auth token format invalid")
				response.ResponseError(c, http.StatusUnauthorized, response.ErrCodeAuthTokenFormatInvalid)
				c.Abort()
				return
			}
		}
		if tokenStr == "" {
			tokenStr = strings.TrimSpace(c.Query("token"))
		}
		if tokenStr == "" {
			logger.Errorf("auth token missing")
			response.ResponseError(c, http.StatusUnauthorized, response.ErrCodeAuthTokenRequired)
			c.Abort()
			return
		}

		claims, err := utils.ParseToken(tokenStr)
		if err != nil {
			logger.Errorf("auth token parse failed: %v", err)
			if utils.IsTokenExpired(err) {
				response.ResponseError(c, http.StatusUnauthorized, response.ErrCodeAuthTokenExpired)
			} else {
				response.ResponseError(c, http.StatusUnauthorized, response.ErrCodeAuthTokenInvalid)
			}
			c.Abort()
			return
		}

		// 把鉴权成功后的用户信息写入上下文，供后续 handler 复用。
		c.Set("userID", claims.UserID)
		c.Set("userName", claims.UserName)
		c.Set("role", claims.Role)

		c.Next()
	}
}

// RequireRoles 进行 RBAC 角色校验，要求当前用户角色命中白名单。
func RequireRoles(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		normalized := strings.TrimSpace(strings.ToLower(role))
		if normalized != "" {
			allowed[normalized] = struct{}{}
		}
	}

	return func(c *gin.Context) {
		roleValue, exists := c.Get("role")
		if !exists {
			logger.Errorf("auth role missing in context")
			response.ResponseError(c, http.StatusForbidden, response.ErrCodePermissionRoleDenied)
			c.Abort()
			return
		}

		roleText, ok := roleValue.(string)
		if !ok {
			logger.Errorf("auth role type invalid")
			response.ResponseError(c, http.StatusForbidden, response.ErrCodePermissionRoleDenied)
			c.Abort()
			return
		}

		if _, ok := allowed[strings.ToLower(strings.TrimSpace(roleText))]; !ok {
			logger.Errorf("auth role denied role=%s", strings.ToLower(strings.TrimSpace(roleText)))
			response.ResponseError(c, http.StatusForbidden, response.ErrCodePermissionRoleDenied)
			c.Abort()
			return
		}

		c.Next()
	}
}
