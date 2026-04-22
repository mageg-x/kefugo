package controllers

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"kefu-server/models"
	"kefu-server/service"
	"kefu-server/store"
	"kefu-server/utils/logger"
	"kefu-server/utils/response"
)

type APIModelConfigController struct{}

type apiModelConfigReq struct {
	Provider    string  `json:"provider"`
	APIKey      string  `json:"api_key"`
	BaseURL     string  `json:"base_url"`
	ModelName   string  `json:"model_name"`
	TimeoutSec  int     `json:"timeout_sec"`
	Temperature float64 `json:"temperature"`
	TopP        float64 `json:"top_p"`
	MaxTokens   int     `json:"max_tokens"`
	Status      int     `json:"status"`
}

func maskAPIKey(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if len(raw) <= 8 {
		return "****"
	}
	return raw[:4] + "****" + raw[len(raw)-4:]
}

func toAPIModelConfigOutput(item models.APIModelConfig) gin.H {
	item = serviceClampAPIModelConfig(item)
	return gin.H{
		"id":           item.ID,
		"created_at":   item.CreatedAt,
		"updated_at":   item.UpdatedAt,
		"provider":     item.Provider,
		"api_key":      strings.TrimSpace(item.APIKey),
		"api_key_mask": maskAPIKey(item.APIKey),
		"base_url":     item.BaseURL,
		"model_name":   item.ModelName,
		"timeout_sec":  item.TimeoutSec,
		"temperature":  item.Temperature,
		"top_p":        item.TopP,
		"max_tokens":   item.MaxTokens,
		"status":       item.Status,
	}
}

func serviceClampAPIModelConfig(item models.APIModelConfig) models.APIModelConfig {
	item.Provider = strings.ToLower(strings.TrimSpace(item.Provider))
	item.BaseURL = strings.TrimSpace(item.BaseURL)
	item.ModelName = strings.TrimSpace(item.ModelName)
	item.APIKey = strings.TrimSpace(item.APIKey)
	if item.TimeoutSec <= 0 {
		item.TimeoutSec = 60
	}
	if item.TimeoutSec > 600 {
		item.TimeoutSec = 600
	}
	if item.Temperature < 0 {
		item.Temperature = 0
	}
	if item.Temperature > 2 {
		item.Temperature = 2
	}
	if item.TopP <= 0 || item.TopP > 1 {
		item.TopP = 0.9
	}
	if item.MaxTokens <= 0 {
		item.MaxTokens = 512
	}
	if item.MaxTokens > 8192 {
		item.MaxTokens = 8192
	}
	if item.Status != 1 {
		item.Status = 0
	}
	return item
}

func validateAPIModelConfigInput(item models.APIModelConfig) bool {
	switch strings.ToLower(strings.TrimSpace(item.Provider)) {
	case "openai", "qwen", "deepseek", "zhipu":
	default:
		return false
	}
	if strings.TrimSpace(item.ModelName) == "" {
		return false
	}
	if strings.TrimSpace(item.APIKey) == "" {
		return false
	}
	if strings.TrimSpace(item.BaseURL) != "" {
		parsedURL, err := url.Parse(strings.TrimSpace(item.BaseURL))
		if err != nil || !parsedURL.IsAbs() || strings.TrimSpace(parsedURL.Host) == "" {
			return false
		}
	}
	return true
}

func (ac *APIModelConfigController) List(c *gin.Context) {
	var rows []models.APIModelConfig
	if err := store.DB.Order("updated_at DESC").Find(&rows).Error; err != nil {
		logger.Errorf("api model config list failed err=%v", err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeKnowledgeModelListFailed)
		return
	}
	out := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		out = append(out, toAPIModelConfigOutput(row))
	}
	response.ResponseSuccess(c, gin.H{"data": out})
}

func (ac *APIModelConfigController) Get(c *gin.Context) {
	id, err := strconv.ParseUint(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || id == 0 {
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeKnowledgeModelSaveInvalid)
		return
	}
	var item models.APIModelConfig
	if err := store.DB.Where("id = ?", uint(id)).First(&item).Error; err != nil {
		response.ResponseError(c, http.StatusNotFound, response.ErrCodeKnowledgeModelProfileNotFound)
		return
	}
	out := toAPIModelConfigOutput(item)
	out["api_key"] = strings.TrimSpace(item.APIKey)
	response.ResponseSuccess(c, gin.H{"item": out})
}

func (ac *APIModelConfigController) Create(c *gin.Context) {
	var req apiModelConfigReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeKnowledgeModelSaveInvalid)
		return
	}
	item := serviceClampAPIModelConfig(models.APIModelConfig{
		Provider:    req.Provider,
		APIKey:      req.APIKey,
		BaseURL:     req.BaseURL,
		ModelName:   req.ModelName,
		TimeoutSec:  req.TimeoutSec,
		Temperature: req.Temperature,
		TopP:        req.TopP,
		MaxTokens:   req.MaxTokens,
		Status:      req.Status,
	})
	if !validateAPIModelConfigInput(item) {
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeKnowledgeModelSaveInvalid)
		return
	}
	if err := store.DB.Transaction(func(tx *gorm.DB) error {
		if item.Status == 1 {
			if err := tx.Model(&models.APIModelConfig{}).Where("status = ?", 1).Update("status", 0).Error; err != nil {
				return err
			}
		}
		return tx.Create(&item).Error
	}); err != nil {
		logger.Errorf("api model config create failed err=%v", err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeKnowledgeModelSaveFailed)
		return
	}
	RecordAudit(c, "knowledge_model_api.create", "api_model_config", strconv.FormatUint(uint64(item.ID), 10), "success", item.ModelName)
	response.ResponseSuccess(c, gin.H{"item": toAPIModelConfigOutput(item)})
}

func (ac *APIModelConfigController) Update(c *gin.Context) {
	id, err := strconv.ParseUint(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || id == 0 {
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeKnowledgeModelSaveInvalid)
		return
	}
	var req apiModelConfigReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeKnowledgeModelSaveInvalid)
		return
	}
	var item models.APIModelConfig
	if err := store.DB.Where("id = ?", uint(id)).First(&item).Error; err != nil {
		response.ResponseError(c, http.StatusNotFound, response.ErrCodeKnowledgeModelProfileNotFound)
		return
	}

	item.Provider = req.Provider
	item.BaseURL = req.BaseURL
	item.ModelName = req.ModelName
	item.TimeoutSec = req.TimeoutSec
	item.Temperature = req.Temperature
	item.TopP = req.TopP
	item.MaxTokens = req.MaxTokens
	item.Status = req.Status
	if strings.TrimSpace(req.APIKey) != "" {
		item.APIKey = req.APIKey
	}
	item = serviceClampAPIModelConfig(item)
	if !validateAPIModelConfigInput(item) {
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeKnowledgeModelSaveInvalid)
		return
	}
	if err := store.DB.Transaction(func(tx *gorm.DB) error {
		if item.Status == 1 {
			if err := tx.Model(&models.APIModelConfig{}).
				Where("id <> ? AND status = ?", item.ID, 1).
				Update("status", 0).Error; err != nil {
				return err
			}
		}
		return tx.Save(&item).Error
	}); err != nil {
		logger.Errorf("api model config update failed id=%d err=%v", item.ID, err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeKnowledgeModelSaveFailed)
		return
	}
	RecordAudit(c, "knowledge_model_api.update", "api_model_config", strconv.FormatUint(uint64(item.ID), 10), "success", item.ModelName)
	response.ResponseSuccess(c, gin.H{"item": toAPIModelConfigOutput(item)})
}

func (ac *APIModelConfigController) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || id == 0 {
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeKnowledgeModelSaveInvalid)
		return
	}
	var item models.APIModelConfig
	if err := store.DB.Where("id = ?", uint(id)).First(&item).Error; err != nil {
		response.ResponseError(c, http.StatusNotFound, response.ErrCodeKnowledgeModelProfileNotFound)
		return
	}
	if err := store.DB.Delete(&models.APIModelConfig{}, item.ID).Error; err != nil {
		logger.Errorf("api model config delete failed id=%d err=%v", item.ID, err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeKnowledgeModelSaveFailed)
		return
	}
	RecordAudit(c, "knowledge_model_api.delete", "api_model_config", strconv.FormatUint(uint64(item.ID), 10), "success", item.ModelName)
	response.ResponseSuccess(c, gin.H{"message": "delete successful"})
}

func (ac *APIModelConfigController) Test(c *gin.Context) {
	id, err := strconv.ParseUint(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || id == 0 {
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeKnowledgeModelSaveInvalid)
		return
	}
	var item models.APIModelConfig
	if err := store.DB.Where("id = ?", uint(id)).First(&item).Error; err != nil {
		response.ResponseError(c, http.StatusNotFound, response.ErrCodeKnowledgeModelProfileNotFound)
		return
	}
	timeout := item.TimeoutSec
	if timeout <= 0 {
		timeout = 60
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), time.Duration(timeout+10)*time.Second)
	defer cancel()

	text, inferMS, err := service.TestAPIModelConnection(ctx, item)
	if err != nil {
		logger.Errorf("api model config test failed id=%d provider=%s model=%s err=%v",
			item.ID, item.Provider, item.ModelName, err)
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			response.ResponseErrorWithMsg(c, http.StatusBadRequest, response.ErrCodeKnowledgeModelInferFailed, "model request timeout")
			return
		}
		response.ResponseErrorWithMsg(c, http.StatusBadRequest, response.ErrCodeKnowledgeModelInferFailed, err.Error())
		return
	}
	RecordAudit(c, "knowledge_model_api.test", "api_model_config", strconv.FormatUint(uint64(item.ID), 10), "success", item.ModelName)
	response.ResponseSuccess(c, gin.H{
		"id":          item.ID,
		"provider":    item.Provider,
		"model_name":  item.ModelName,
		"infer_ms":    inferMS,
		"preview":     text,
		"status_text": "ready",
	})
}
