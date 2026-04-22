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

type FAQController struct{}

// List 返回 FAQ 列表。
func (fc *FAQController) List(c *gin.Context) {
	appID := strings.TrimSpace(c.Query("app_id"))
	category := strings.TrimSpace(c.Query("category"))
	if appID == "" {
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeAppConfigInvalidParams)
		return
	}

	query := store.DB.Model(&models.FAQItem{}).Where("app_id = ?", appID)
	if category != "" {
		query = query.Where("category = ?", category)
	}
	var rows []models.FAQItem
	if err := query.Order("updated_at DESC").Find(&rows).Error; err != nil {
		logger.Errorf("faq list failed app_id=%s err=%v", appID, err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeDBQueryFailed)
		return
	}
	response.ResponseSuccess(c, gin.H{"data": rows})
}

// Create 创建 FAQ（客服与管理员）。
func (fc *FAQController) Create(c *gin.Context) {
	var req struct {
		AppID    string `json:"app_id" binding:"required"`
		Question string `json:"question" binding:"required"`
		Answer   string `json:"answer" binding:"required"`
		Category string `json:"category"`
		Enabled  *bool  `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Errorf("faq create params invalid err=%v", err)
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeInvalidParams)
		return
	}

	item := models.FAQItem{
		AppID:    strings.TrimSpace(req.AppID),
		Question: strings.TrimSpace(req.Question),
		Answer:   strings.TrimSpace(req.Answer),
		Category: strings.TrimSpace(req.Category),
		Enabled:  true,
	}
	if req.Enabled != nil {
		item.Enabled = *req.Enabled
	}
	if item.AppID == "" || item.Question == "" || item.Answer == "" {
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeInvalidParams)
		return
	}

	if err := store.DB.Create(&item).Error; err != nil {
		logger.Errorf("faq create failed app_id=%s err=%v", item.AppID, err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeDBWriteFailed)
		return
	}
	RecordAudit(c, "faq.create", "faq", strconv.FormatUint(uint64(item.ID), 10), "success", item.Question)
	response.ResponseSuccess(c, gin.H{"item": item})
}

// Update 更新 FAQ（客服与管理员）。
func (fc *FAQController) Update(c *gin.Context) {
	var req struct {
		ID       uint   `json:"id" binding:"required"`
		Question string `json:"question" binding:"required"`
		Answer   string `json:"answer" binding:"required"`
		Category string `json:"category"`
		Enabled  bool   `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Errorf("faq update params invalid err=%v", err)
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeInvalidParams)
		return
	}

	var item models.FAQItem
	if err := store.DB.Where("id = ?", req.ID).First(&item).Error; err != nil {
		response.ResponseError(c, http.StatusNotFound, response.ErrCodeNotFound)
		return
	}
	item.Question = strings.TrimSpace(req.Question)
	item.Answer = strings.TrimSpace(req.Answer)
	item.Category = strings.TrimSpace(req.Category)
	item.Enabled = req.Enabled
	if item.Question == "" || item.Answer == "" {
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeInvalidParams)
		return
	}

	if err := store.DB.Save(&item).Error; err != nil {
		logger.Errorf("faq update failed id=%d err=%v", req.ID, err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeDBWriteFailed)
		return
	}
	RecordAudit(c, "faq.update", "faq", strconv.FormatUint(uint64(item.ID), 10), "success", item.Question)
	response.ResponseSuccess(c, gin.H{"item": item})
}

// Delete 硬删除 FAQ（客服与管理员）。
func (fc *FAQController) Delete(c *gin.Context) {
	idStr := strings.TrimSpace(c.Query("id"))
	id64, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || id64 == 0 {
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeInvalidParams)
		return
	}

	result := store.DB.Unscoped().Delete(&models.FAQItem{}, uint(id64))
	if result.Error != nil {
		logger.Errorf("faq delete failed id=%s err=%v", idStr, result.Error)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeDBWriteFailed)
		return
	}
	if result.RowsAffected == 0 {
		response.ResponseError(c, http.StatusNotFound, response.ErrCodeNotFound)
		return
	}
	RecordAudit(c, "faq.delete", "faq", idStr, "success", "")
	response.ResponseSuccess(c, gin.H{"message": "delete successful"})
}
