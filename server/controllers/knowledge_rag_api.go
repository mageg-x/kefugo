package controllers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"kefu-server/utils/response"
)

// RAGTest 提供知识库检索测试接口（管理后台调试使用）。
func (kc *KnowledgeController) RAGTest(c *gin.Context) {
	var req struct {
		AppID string `json:"app_id" binding:"required"`
		Query string `json:"query" binding:"required"`
		TopK  int    `json:"top_k"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeInvalidParams)
		return
	}
	appID := strings.TrimSpace(req.AppID)
	query := strings.TrimSpace(req.Query)
	if appID == "" || query == "" {
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeInvalidParams)
		return
	}
	results := queryKnowledgeChunks(appID, query, req.TopK)
	response.ResponseSuccess(c, gin.H{
		"query":   query,
		"results": results,
		"count":   len(results),
	})
}
