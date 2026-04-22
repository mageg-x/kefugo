package controllers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"kefu-server/models"
	"kefu-server/store"
	"kefu-server/utils"
	"kefu-server/utils/logger"
	"kefu-server/utils/response"
)

type AppAPIKeyController struct{}

func maskSecret(secret string) string {
	if secret == "" {
		return ""
	}
	if len(secret) <= 8 {
		return "********"
	}
	return secret[:4] + "********" + secret[len(secret)-4:]
}

func newKeyID() string {
	return "key_" + utils.GenerateTimestamp() + "_" + utils.GenerateRandomString(6)
}

func newSecret() string {
	return "ks_" + utils.GenerateTimestamp() + "_" + utils.GenerateRandomString(24)
}

// List 返回指定应用的 API 密钥列表（管理员）。
func (ac *AppAPIKeyController) List(c *gin.Context) {
	userName, role := getAuthUser(c)
	if role != "admin" && role != "agent" {
		logger.Errorf("api key list forbidden user=%s role=%s", userName, role)
		response.ResponseError(c, http.StatusForbidden, response.ErrCodePermissionAdminRequired)
		return
	}
	appID := strings.TrimSpace(c.Query("app_id"))
	if appID == "" {
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeAppConfigInvalidParams)
		return
	}

	var rows []models.AppAPIKey
	if err := store.DB.Where("app_id = ?", appID).Order("created_at DESC").Find(&rows).Error; err != nil {
		logger.Errorf("api key list failed app_id=%s err=%v", appID, err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeDBQueryFailed)
		return
	}

	type rowItem struct {
		ID           uint   `json:"id"`
		AppID        string `json:"app_id"`
		Name         string `json:"name"`
		KeyID        string `json:"key_id"`
		SecretMasked string `json:"secret_masked"`
		Enabled      bool   `json:"enabled"`
		CreatedAt    string `json:"created_at"`
		UpdatedAt    string `json:"updated_at"`
	}
	items := make([]rowItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, rowItem{
			ID:           row.ID,
			AppID:        row.AppID,
			Name:         row.Name,
			KeyID:        row.KeyID,
			SecretMasked: maskSecret(row.Secret),
			Enabled:      row.Enabled,
			CreatedAt:    row.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:    row.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	response.ResponseSuccess(c, gin.H{"data": items})
}

// Create 为应用创建新密钥（管理员）。
func (ac *AppAPIKeyController) Create(c *gin.Context) {
	userName, role := getAuthUser(c)
	if role != "admin" && role != "agent" {
		logger.Errorf("api key create forbidden user=%s role=%s", userName, role)
		response.ResponseError(c, http.StatusForbidden, response.ErrCodePermissionAdminRequired)
		return
	}
	var req struct {
		AppID string `json:"app_id" binding:"required"`
		Name  string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeInvalidParams)
		return
	}

	item := models.AppAPIKey{
		AppID:   strings.TrimSpace(req.AppID),
		Name:    strings.TrimSpace(req.Name),
		KeyID:   newKeyID(),
		Secret:  newSecret(),
		Enabled: true,
	}
	if item.AppID == "" || item.Name == "" {
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeInvalidParams)
		return
	}
	if err := store.DB.Create(&item).Error; err != nil {
		logger.Errorf("api key create failed app_id=%s err=%v", item.AppID, err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeDBWriteFailed)
		return
	}
	RecordAudit(c, "app_api_key.create", "app_api_key", strconv.FormatUint(uint64(item.ID), 10), "success", item.Name)
	response.ResponseSuccess(c, gin.H{
		"id":      item.ID,
		"app_id":  item.AppID,
		"name":    item.Name,
		"key_id":  item.KeyID,
		"secret":  item.Secret,
		"enabled": item.Enabled,
	})
}

// Rotate 轮换指定密钥 secret（管理员）。
func (ac *AppAPIKeyController) Rotate(c *gin.Context) {
	userName, role := getAuthUser(c)
	if role != "admin" && role != "agent" {
		logger.Errorf("api key rotate forbidden user=%s role=%s", userName, role)
		response.ResponseError(c, http.StatusForbidden, response.ErrCodePermissionAdminRequired)
		return
	}
	var req struct {
		ID uint `json:"id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeInvalidParams)
		return
	}
	var item models.AppAPIKey
	if err := store.DB.Where("id = ?", req.ID).First(&item).Error; err != nil {
		response.ResponseError(c, http.StatusNotFound, response.ErrCodeNotFound)
		return
	}
	item.Secret = newSecret()
	if err := store.DB.Save(&item).Error; err != nil {
		logger.Errorf("api key rotate failed id=%d err=%v", req.ID, err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeDBWriteFailed)
		return
	}
	RecordAudit(c, "app_api_key.rotate", "app_api_key", strconv.FormatUint(uint64(item.ID), 10), "success", item.Name)
	response.ResponseSuccess(c, gin.H{
		"id":      item.ID,
		"secret":  item.Secret,
		"key_id":  item.KeyID,
		"enabled": item.Enabled,
	})
}

// SetEnabled 启用/禁用指定密钥（管理员）。
func (ac *AppAPIKeyController) SetEnabled(c *gin.Context) {
	userName, role := getAuthUser(c)
	if role != "admin" && role != "agent" {
		logger.Errorf("api key set enabled forbidden user=%s role=%s", userName, role)
		response.ResponseError(c, http.StatusForbidden, response.ErrCodePermissionAdminRequired)
		return
	}
	var req struct {
		ID      uint `json:"id" binding:"required"`
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeInvalidParams)
		return
	}
	var item models.AppAPIKey
	if err := store.DB.Where("id = ?", req.ID).First(&item).Error; err != nil {
		response.ResponseError(c, http.StatusNotFound, response.ErrCodeNotFound)
		return
	}
	item.Enabled = req.Enabled
	if err := store.DB.Save(&item).Error; err != nil {
		logger.Errorf("api key set enabled failed id=%d err=%v", req.ID, err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeDBWriteFailed)
		return
	}
	RecordAudit(c, "app_api_key.set_enabled", "app_api_key", strconv.FormatUint(uint64(item.ID), 10), "success", item.Name)
	response.ResponseSuccess(c, gin.H{"id": item.ID, "enabled": item.Enabled})
}

// Delete 硬删除指定密钥（管理员）。
func (ac *AppAPIKeyController) Delete(c *gin.Context) {
	userName, role := getAuthUser(c)
	if role != "admin" && role != "agent" {
		logger.Errorf("api key delete forbidden user=%s role=%s", userName, role)
		response.ResponseError(c, http.StatusForbidden, response.ErrCodePermissionAdminRequired)
		return
	}
	idStr := strings.TrimSpace(c.Query("id"))
	id64, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || id64 == 0 {
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeInvalidParams)
		return
	}
	result := store.DB.Unscoped().Delete(&models.AppAPIKey{}, uint(id64))
	if result.Error != nil {
		logger.Errorf("api key delete failed id=%s err=%v", idStr, result.Error)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeDBWriteFailed)
		return
	}
	if result.RowsAffected == 0 {
		response.ResponseError(c, http.StatusNotFound, response.ErrCodeNotFound)
		return
	}
	RecordAudit(c, "app_api_key.delete", "app_api_key", idStr, "success", "")
	response.ResponseSuccess(c, gin.H{"message": "delete successful"})
}
