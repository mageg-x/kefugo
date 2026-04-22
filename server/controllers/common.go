package controllers

import "github.com/gin-gonic/gin"

// getAuthUser 从 gin 上下文读取认证后的用户名与角色。
// 上游 AuthMiddleware 已保证字段存在，函数内部做安全断言并返回空值容错。
func getAuthUser(c *gin.Context) (userName, role string) {
	userNameValue, _ := c.Get("userName")
	roleValue, _ := c.Get("role")

	userNameStr, _ := userNameValue.(string)
	roleStr, _ := roleValue.(string)
	return userNameStr, roleStr
}

// getAuthUserID 从 gin 上下文读取认证后的用户 ID。
// 优先读取统一键 userID，同时兼容历史键 user_id，避免控制器读取不一致导致误判未登录。
func getAuthUserID(c *gin.Context) (uint, bool) {
	if value, exists := c.Get("userID"); exists {
		if uid, ok := value.(uint); ok {
			return uid, true
		}
	}
	if value, exists := c.Get("user_id"); exists {
		if uid, ok := value.(uint); ok {
			return uid, true
		}
	}
	return 0, false
}
