package controllers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"kefu-server/models"
	"kefu-server/store"
	"kefu-server/utils/logger"
	"kefu-server/utils/response"
)

// UploadByURL 将已上传文件（upload接口返回的 url）导入知识库。
// 当前实现：读取文本类文件内容；二进制文件可记录为元数据占位，后续可接入异步解析器。
func (kc *KnowledgeController) UploadByURL(c *gin.Context) {
	if !IsAdmin(c) {
		response.ResponseError(c, http.StatusForbidden, response.ErrCodePermissionAdminRequired)
		return
	}
	var req struct {
		AppID string `json:"app_id" binding:"required"`
		Name  string `json:"name" binding:"required"`
		URL   string `json:"url" binding:"required"`
		Tags  string `json:"tags"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Errorf("knowledge upload params invalid err=%v", err)
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeInvalidParams)
		return
	}

	appID := strings.TrimSpace(req.AppID)
	name := strings.TrimSpace(req.Name)
	fileURL := strings.TrimSpace(req.URL)
	tags := strings.TrimSpace(req.Tags)
	if appID == "" || name == "" || fileURL == "" {
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeInvalidParams)
		return
	}

	content, sourceType, sourceName, err := extractKnowledgeContentFromURL(fileURL, name)
	if err != nil {
		logger.Errorf("knowledge upload parse failed app_id=%s url=%s err=%v", appID, fileURL, err)
		response.ResponseErrorWithMsg(c, http.StatusBadRequest, response.ErrCodeUploadValidateFailed, err.Error())
		return
	}
	if strings.TrimSpace(content) == "" {
		response.ResponseErrorWithMsg(c, http.StatusBadRequest, response.ErrCodeUploadValidateFailed, "empty content")
		return
	}

	item := models.KnowledgeArticle{
		AppID:      appID,
		Title:      name,
		Content:    content,
		Tags:       tags,
		Enabled:    true,
		SourceType: sourceType,
		SourceName: sourceName,
	}
	if err := store.DB.Create(&item).Error; err != nil {
		logger.Errorf("knowledge upload create failed app_id=%s err=%v", appID, err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeDBWriteFailed)
		return
	}
	response.ResponseSuccess(c, gin.H{"item": item})
}
