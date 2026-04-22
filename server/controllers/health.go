package controllers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"kefu-server/store"
	"kefu-server/utils/logger"
)

// HealthController 提供服务健康检查接口。
type HealthController struct{}

// Health 返回当前服务与核心依赖（SQLite/Badger）的健康状态。
func (hc *HealthController) Health(c *gin.Context) {
	dbOK := false
	kvOK := false
	if store.DB != nil {
		sqlDB, err := store.DB.DB()
		if err == nil && sqlDB.Ping() == nil {
			dbOK = true
		} else if err != nil {
			logger.Errorf("health sqlite db handle failed: %v", err)
		} else {
			logger.Errorf("health sqlite ping failed")
		}
	}
	if store.KV != nil {
		kvOK = true
	} else {
		logger.Errorf("health kv unavailable")
	}

	status := "ok"
	code := http.StatusOK
	if !dbOK || !kvOK {
		status = "degraded"
		code = http.StatusServiceUnavailable
	}

	c.JSON(code, gin.H{
		"status": status,
		"time":   time.Now().Unix(),
		"db":     dbOK,
		"kv":     kvOK,
	})
}
