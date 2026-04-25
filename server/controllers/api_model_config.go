package controllers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"kefu-server/models"
	"kefu-server/service"
	"kefu-server/service/ai/embedding"
	"kefu-server/service/ai/rerank"
	"kefu-server/store"
	"kefu-server/utils"
	"kefu-server/utils/logger"
	"kefu-server/utils/response"
)

type APIModelConfigController struct{}

type apiModelConfigReq struct {
	ModelType   string  `json:"model_type"`
	Name        string  `json:"name"`
	Provider    string  `json:"provider"`
	APIKey      string  `json:"api_key"`
	BaseURL     string  `json:"base_url"`
	ModelName   string  `json:"model_name"`
	Dims        int     `json:"dims"`
	TopK        int     `json:"top_k"`
	TopN        int     `json:"top_n"`
	TimeoutSec  int     `json:"timeout_sec"`
	Temperature float64 `json:"temperature"`
	TopP        float64 `json:"top_p"`
	MaxTokens   int     `json:"max_tokens"`
	IsDefault   *bool   `json:"is_default"`
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
	decryptedKey := utils.DecryptAPIKey(item.APIKey)
	return gin.H{
		"id":           item.ID,
		"created_at":   item.CreatedAt,
		"updated_at":   item.UpdatedAt,
		"model_type":   item.ModelType,
		"name":         item.Name,
		"provider":     item.Provider,
		"api_key":      strings.TrimSpace(decryptedKey),
		"api_key_mask": maskAPIKey(decryptedKey),
		"base_url":     item.BaseURL,
		"model_name":   item.ModelName,
		"dims":         item.Dims,
		"top_k":        item.TopK,
		"top_n":        item.TopN,
		"timeout_sec":  item.TimeoutSec,
		"temperature":  item.Temperature,
		"top_p":        item.TopP,
		"max_tokens":   item.MaxTokens,
		"is_default":   item.IsDefault,
		"status":       item.Status,
	}
}

func serviceClampAPIModelConfig(item models.APIModelConfig) models.APIModelConfig {
	item.ModelType = strings.ToLower(strings.TrimSpace(item.ModelType))
	if item.ModelType == "" {
		item.ModelType = string(models.AIModelTypeChat)
	}
	item.Name = strings.TrimSpace(item.Name)
	item.Provider = strings.ToLower(strings.TrimSpace(item.Provider))
	item.BaseURL = strings.TrimSpace(item.BaseURL)
	item.ModelName = strings.TrimSpace(item.ModelName)
	item.APIKey = utils.EncryptAPIKey(strings.TrimSpace(item.APIKey))
	if item.Dims <= 0 {
		item.Dims = 384
	}
	if item.TopK <= 0 {
		item.TopK = 20
	}
	if item.TopN <= 0 {
		item.TopN = 5
	}
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
	if !models.ValidAIModelType(item.ModelType) {
		return false
	}
	rawKey := strings.TrimSpace(utils.DecryptAPIKey(item.APIKey))
	switch item.Provider {
	case "openai", "qwen", "deepseek", "zhipu", "cohere", "jina":
		if rawKey == "" {
			return false
		}
	case "ollama":
	default:
		return false
	}
	if strings.TrimSpace(item.ModelName) == "" {
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
	query := store.DB.Order("updated_at DESC")
	if mt := strings.TrimSpace(c.Query("model_type")); mt != "" {
		query = query.Where("model_type = ?", mt)
	}
	var rows []models.APIModelConfig
	if err := query.Find(&rows).Error; err != nil {
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
	out["api_key"] = strings.TrimSpace(utils.DecryptAPIKey(item.APIKey))
	response.ResponseSuccess(c, gin.H{"item": out})
}

func (ac *APIModelConfigController) Create(c *gin.Context) {
	var req apiModelConfigReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeKnowledgeModelSaveInvalid)
		return
	}
	item := serviceClampAPIModelConfig(models.APIModelConfig{
		ModelType:   req.ModelType,
		Name:        req.Name,
		Provider:    req.Provider,
		APIKey:      req.APIKey,
		BaseURL:     req.BaseURL,
		ModelName:   req.ModelName,
		Dims:        req.Dims,
		TopK:        req.TopK,
		TopN:        req.TopN,
		TimeoutSec:  req.TimeoutSec,
		Temperature: req.Temperature,
		TopP:        req.TopP,
		MaxTokens:   req.MaxTokens,
		Status:      req.Status,
	})
	if req.IsDefault != nil {
		item.IsDefault = *req.IsDefault
	}
	if item.ModelType == string(models.AIModelTypeEmbedding) && item.Dims <= 0 {
		if detectedDims, err := autoDetectEmbeddingDims(c.Request.Context(), item); err == nil && detectedDims > 0 {
			item.Dims = detectedDims
		}
	}
	if !validateAPIModelConfigInput(item) {
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeKnowledgeModelSaveInvalid)
		return
	}
	if err := store.DB.Transaction(func(tx *gorm.DB) error {
		if item.Status == 1 {
			if err := tx.Model(&models.APIModelConfig{}).
				Where("model_type = ? AND status = ?", item.ModelType, 1).
				Update("status", 0).Error; err != nil {
				return err
			}
		}
		if item.IsDefault {
			if err := tx.Model(&models.APIModelConfig{}).
				Where("model_type = ? AND is_default = ?", item.ModelType, true).
				Update("is_default", false).Error; err != nil {
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

	if req.ModelType != "" {
		item.ModelType = req.ModelType
	}
	item.Name = req.Name
	item.Provider = req.Provider
	item.BaseURL = req.BaseURL
	item.ModelName = req.ModelName
	item.Dims = req.Dims
	item.TopK = req.TopK
	item.TopN = req.TopN
	item.TimeoutSec = req.TimeoutSec
	item.Temperature = req.Temperature
	item.TopP = req.TopP
	item.MaxTokens = req.MaxTokens
	item.Status = req.Status
	if req.IsDefault != nil {
		item.IsDefault = *req.IsDefault
	}
	if strings.TrimSpace(req.APIKey) != "" {
		item.APIKey = utils.EncryptAPIKey(strings.TrimSpace(req.APIKey))
	}
	item = serviceClampAPIModelConfig(item)
	if item.ModelType == string(models.AIModelTypeEmbedding) && item.Dims <= 0 {
		if detectedDims, err := autoDetectEmbeddingDims(c.Request.Context(), item); err == nil && detectedDims > 0 {
			item.Dims = detectedDims
		}
	}
	if !validateAPIModelConfigInput(item) {
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeKnowledgeModelSaveInvalid)
		return
	}
	if err := store.DB.Transaction(func(tx *gorm.DB) error {
		if item.Status == 1 {
			if err := tx.Model(&models.APIModelConfig{}).
				Where("id <> ? AND model_type = ? AND status = ?", item.ID, item.ModelType, 1).
				Update("status", 0).Error; err != nil {
				return err
			}
		}
		if item.IsDefault {
			if err := tx.Model(&models.APIModelConfig{}).
				Where("id <> ? AND model_type = ? AND is_default = ?", item.ID, item.ModelType, true).
				Update("is_default", false).Error; err != nil {
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
	item.APIKey = utils.DecryptAPIKey(item.APIKey)
	timeout := item.TimeoutSec
	if timeout <= 0 {
		timeout = 60
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), time.Duration(timeout+10)*time.Second)
	defer cancel()

	var text string
	var inferMS int64

	switch models.AIModelType(item.ModelType) {
	case models.AIModelTypeEmbedding:
		text, inferMS, err = testEmbeddingConnection(ctx, item)
	case models.AIModelTypeRerank:
		text, inferMS, err = testRerankConnection(ctx, item)
	default:
		text, inferMS, err = service.TestAPIModelConnection(ctx, item)
	}

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
		"model_type":  item.ModelType,
		"infer_ms":    inferMS,
		"preview":     text,
		"status_text": "ready",
	})
}

func (ac *APIModelConfigController) SetDefault(c *gin.Context) {
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
	if err := store.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.APIModelConfig{}).
			Where("model_type = ? AND is_default = ?", item.ModelType, true).
			Update("is_default", false).Error; err != nil {
			return err
		}
		return tx.Model(&item).Update("is_default", true).Error
	}); err != nil {
		logger.Errorf("api model config set default failed id=%d err=%v", item.ID, err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeKnowledgeModelSaveFailed)
		return
	}
	item.IsDefault = true
	RecordAudit(c, "knowledge_model_api.set_default", "api_model_config", strconv.FormatUint(uint64(item.ID), 10), "success", item.ModelName)
	response.ResponseSuccess(c, gin.H{"item": toAPIModelConfigOutput(item)})
}

func testEmbeddingConnection(ctx context.Context, cfg models.APIModelConfig) (string, int64, error) {
	provider, err := embedding.NewProvider(&cfg)
	if err != nil {
		return "", 0, err
	}
	startAt := time.Now()
	vec, err := provider.GetEmbedding(ctx, "test")
	if err != nil {
		return "", time.Since(startAt).Milliseconds(), err
	}
	preview := fmt.Sprintf("dims=%d", len(vec))
	return preview, time.Since(startAt).Milliseconds(), nil
}

func autoDetectEmbeddingDims(ctx context.Context, cfg models.APIModelConfig) (int, error) {
	timeout := cfg.TimeoutSec
	if timeout <= 0 {
		timeout = 30
	}
	detectCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()
	provider, err := embedding.NewProvider(&cfg)
	if err != nil {
		return 0, err
	}
	vec, err := provider.GetEmbedding(detectCtx, "test")
	if err != nil {
		return 0, err
	}
	return len(vec), nil
}

func (ac *APIModelConfigController) TriggerRebuild(c *gin.Context) {
	id, err := strconv.ParseUint(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || id == 0 {
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeKnowledgeModelSaveInvalid)
		return
	}
	task, err := service.TriggerRebuild(uint(id))
	if err != nil {
		response.ResponseErrorWithMsg(c, http.StatusBadRequest, response.ErrCodeKnowledgeModelSaveInvalid, err.Error())
		return
	}
	response.ResponseSuccess(c, gin.H{"task": task})
}

func (ac *APIModelConfigController) GetRebuildStatus(c *gin.Context) {
	id, err := strconv.ParseUint(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || id == 0 {
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeKnowledgeModelSaveInvalid)
		return
	}
	task, err := service.GetLatestRebuildTask(uint(id))
	if err != nil {
		response.ResponseSuccess(c, gin.H{"task": nil})
		return
	}
	response.ResponseSuccess(c, gin.H{"task": task})
}

func testRerankConnection(ctx context.Context, cfg models.APIModelConfig) (string, int64, error) {
	provider, err := rerank.NewProvider(&cfg)
	if err != nil {
		return "", 0, err
	}
	startAt := time.Now()
	candidates := []rerank.Candidate{{ID: "1", Content: "test document content"}}
	reranked, err := provider.Rerank("test query", candidates, 1)
	if err != nil {
		return "", time.Since(startAt).Milliseconds(), err
	}
	preview := "ok"
	if len(reranked) > 0 {
		preview = fmt.Sprintf("score=%.4f", reranked[0].Score)
	}
	return preview, time.Since(startAt).Milliseconds(), nil
}
