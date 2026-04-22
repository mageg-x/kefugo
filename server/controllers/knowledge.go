package controllers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"kefu-server/models"
	"kefu-server/store"
	"kefu-server/utils/logger"
	"kefu-server/utils/response"
)

type KnowledgeController struct{}

// List 返回知识库文章列表。
func (kc *KnowledgeController) List(c *gin.Context) {
	appID := strings.TrimSpace(c.Query("app_id"))
	keyword := strings.TrimSpace(c.Query("keyword"))
	enabled := strings.TrimSpace(c.Query("enabled"))

	if appID == "" {
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeAppConfigInvalidParams)
		return
	}

	query := store.DB.Model(&models.KnowledgeArticle{}).Where("app_id = ?", appID)
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("title LIKE ? OR content LIKE ? OR tags LIKE ? OR source_name LIKE ?", like, like, like, like)
	}
	if enabled != "" {
		if enabled == "1" || strings.EqualFold(enabled, "true") {
			query = query.Where("enabled = ?", true)
		} else {
			query = query.Where("enabled = ?", false)
		}
	}

	var rows []models.KnowledgeArticle
	if err := query.Order("updated_at DESC").Find(&rows).Error; err != nil {
		logger.Errorf("knowledge list failed app_id=%s err=%v", appID, err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeDBQueryFailed)
		return
	}
	response.ResponseSuccess(c, gin.H{"data": rows})
}

// Create 创建知识库文章（客服与管理员）。
func (kc *KnowledgeController) Create(c *gin.Context) {
	var req struct {
		AppID   string `json:"app_id" binding:"required"`
		Title   string `json:"title" binding:"required"`
		Content string `json:"content" binding:"required"`
		Tags    string `json:"tags"`
		Enabled *bool  `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Errorf("knowledge create params invalid err=%v", err)
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeInvalidParams)
		return
	}

	item := models.KnowledgeArticle{
		AppID:      strings.TrimSpace(req.AppID),
		Title:      strings.TrimSpace(req.Title),
		Content:    strings.TrimSpace(req.Content),
		Tags:       strings.TrimSpace(req.Tags),
		Enabled:    true,
		SourceType: "manual",
		SourceName: strings.TrimSpace(req.Title),
	}
	if req.Enabled != nil {
		item.Enabled = *req.Enabled
	}
	if item.AppID == "" || item.Title == "" || item.Content == "" {
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeInvalidParams)
		return
	}

	if err := store.DB.Create(&item).Error; err != nil {
		logger.Errorf("knowledge create failed app_id=%s err=%v", item.AppID, err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeDBWriteFailed)
		return
	}

	RecordAudit(c, "knowledge.create", "knowledge", strconv.FormatUint(uint64(item.ID), 10), "success", item.Title)
	response.ResponseSuccess(c, gin.H{"item": item})
}

// Update 更新知识库文章（客服与管理员）。
func (kc *KnowledgeController) Update(c *gin.Context) {
	var req struct {
		ID      uint   `json:"id" binding:"required"`
		Title   string `json:"title" binding:"required"`
		Content string `json:"content" binding:"required"`
		Tags    string `json:"tags"`
		Enabled bool   `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Errorf("knowledge update params invalid err=%v", err)
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeInvalidParams)
		return
	}

	var item models.KnowledgeArticle
	if err := store.DB.Where("id = ?", req.ID).First(&item).Error; err != nil {
		response.ResponseError(c, http.StatusNotFound, response.ErrCodeNotFound)
		return
	}

	item.Title = strings.TrimSpace(req.Title)
	item.Content = strings.TrimSpace(req.Content)
	item.Tags = strings.TrimSpace(req.Tags)
	item.Enabled = req.Enabled
	if strings.TrimSpace(item.SourceType) == "" {
		item.SourceType = "manual"
	}
	if strings.TrimSpace(item.SourceName) == "" {
		item.SourceName = item.Title
	}
	if item.Title == "" || item.Content == "" {
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeInvalidParams)
		return
	}
	if err := store.DB.Save(&item).Error; err != nil {
		logger.Errorf("knowledge update failed id=%d err=%v", req.ID, err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeDBWriteFailed)
		return
	}
	RecordAudit(c, "knowledge.update", "knowledge", strconv.FormatUint(uint64(item.ID), 10), "success", item.Title)
	response.ResponseSuccess(c, gin.H{"item": item})
}

// Delete 硬删除知识库文章（客服与管理员）。
func (kc *KnowledgeController) Delete(c *gin.Context) {
	idStr := strings.TrimSpace(c.Query("id"))
	id64, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || id64 == 0 {
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeInvalidParams)
		return
	}
	result := store.DB.Unscoped().Delete(&models.KnowledgeArticle{}, uint(id64))
	if result.Error != nil {
		logger.Errorf("knowledge delete failed id=%s err=%v", idStr, result.Error)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeDBWriteFailed)
		return
	}
	if result.RowsAffected == 0 {
		response.ResponseError(c, http.StatusNotFound, response.ErrCodeNotFound)
		return
	}
	RecordAudit(c, "knowledge.delete", "knowledge", idStr, "success", "")
	response.ResponseSuccess(c, gin.H{"message": "delete successful"})
}
