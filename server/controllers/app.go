package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/gin-gonic/gin"
	"kefu-server/models"
	"kefu-server/service"
	"kefu-server/store"
	"kefu-server/utils/logger"
	"kefu-server/utils/response"
)

type AppController struct{}

type AppRequest struct {
	Name        string `json:"name" binding:"required"`
	AppID       string `json:"app_id"`
	Logo        string `json:"logo"`
	AllowDomain string `json:"allow_domain"`
	WelcomeMsg  string `json:"welcome_msg"`
	Contact     string `json:"contact"`
	Status      int    `json:"status" binding:"required,oneof=0 1"`
}

func normalizeAppBindings(raw string) ([]string, bool) {
	text := strings.TrimSpace(raw)
	if text == "" {
		return nil, false
	}
	var apps []string
	if err := json.Unmarshal([]byte(text), &apps); err != nil {
		return nil, false
	}
	result := make([]string, 0, len(apps))
	seen := make(map[string]struct{}, len(apps))
	for _, item := range apps {
		appID := strings.TrimSpace(item)
		if appID == "" {
			continue
		}
		if _, ok := seen[appID]; ok {
			continue
		}
		seen[appID] = struct{}{}
		result = append(result, appID)
	}
	return result, true
}

func removeAppBinding(rawApps, targetAppID string) (string, bool) {
	normalizedApps, ok := normalizeAppBindings(rawApps)
	if !ok {
		return rawApps, false
	}
	target := strings.ToLower(strings.TrimSpace(targetAppID))
	changed := false
	filtered := make([]string, 0, len(normalizedApps))
	for _, appID := range normalizedApps {
		if strings.ToLower(strings.TrimSpace(appID)) == target {
			changed = true
			continue
		}
		filtered = append(filtered, appID)
	}
	if !changed {
		return rawApps, false
	}
	data, err := json.Marshal(filtered)
	if err != nil {
		return rawApps, false
	}
	return string(data), true
}

func (ac *AppController) hardDeleteAppRelatedRows(appID string) error {
	if store.DB == nil {
		return fmt.Errorf("database not initialized")
	}
	tx := store.DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	rollback := func(err error) error {
		_ = tx.Rollback().Error
		return err
	}

	// 清理用户应用绑定，避免残留无效 app_id。
	var users []models.User
	if err := tx.Where("apps LIKE ?", "%"+appID+"%").Find(&users).Error; err != nil {
		return rollback(err)
	}
	for _, user := range users {
		nextApps, changed := removeAppBinding(user.Apps, appID)
		if !changed {
			continue
		}
		if err := tx.Model(&models.User{}).Where("id = ?", user.ID).Update("apps", nextApps).Error; err != nil {
			return rollback(err)
		}
	}

	if err := tx.Unscoped().Where("app_id = ?", appID).Delete(&models.AppAPIKey{}).Error; err != nil {
		return rollback(err)
	}
	if err := tx.Unscoped().Where("app_id = ?", appID).Delete(&models.FAQItem{}).Error; err != nil {
		return rollback(err)
	}
	if err := tx.Unscoped().Where("app_id = ?", appID).Delete(&models.KnowledgeChunk{}).Error; err != nil {
		return rollback(err)
	}
	if err := tx.Unscoped().Where("collection IN (?)",
		tx.Model(&models.KnowledgeBase{}).Select("collection").Where("app_id = ?", appID)).
		Delete(&models.KnowledgeVectorEntry{}).Error; err != nil {
		return rollback(err)
	}
	if err := tx.Unscoped().Where("name IN (?)",
		tx.Model(&models.KnowledgeBase{}).Select("collection").Where("app_id = ?", appID)).
		Delete(&models.KnowledgeVectorCollection{}).Error; err != nil {
		return rollback(err)
	}
	if err := tx.Unscoped().Where("app_id = ?", appID).Delete(&models.KnowledgeDocument{}).Error; err != nil {
		return rollback(err)
	}
	if err := tx.Unscoped().Where("app_id = ?", appID).Delete(&models.KnowledgeQAFeedback{}).Error; err != nil {
		return rollback(err)
	}
	if err := tx.Unscoped().Where("app_id = ?", appID).Delete(&models.KnowledgeBase{}).Error; err != nil {
		return rollback(err)
	}
	if err := tx.Unscoped().Where("app_id = ?", appID).Delete(&models.SessionListIndex{}).Error; err != nil {
		return rollback(err)
	}

	return tx.Commit().Error
}

func collectBadgerDeleteKeysByApp(appID string) ([][]byte, error) {
	kv := store.GetStore()
	if kv == nil {
		return nil, nil
	}
	target := strings.ToLower(strings.TrimSpace(appID))
	keys := make([][]byte, 0, 1024)

	err := kv.View(func(txn *badger.Txn) error {
		// 删除会话 key（s:...）。
		sIt := txn.NewIterator(badger.DefaultIteratorOptions)
		defer sIt.Close()
		sPrefix := []byte("s:")
		for sIt.Seek(sPrefix); sIt.ValidForPrefix(sPrefix); sIt.Next() {
			item := sIt.Item()
			val, err := item.ValueCopy(nil)
			if err != nil {
				continue
			}
			var session models.Session
			if err := json.Unmarshal(val, &session); err != nil {
				continue
			}
			if strings.ToLower(strings.TrimSpace(session.AppID())) != target {
				continue
			}
			keys = append(keys, append([]byte(nil), item.Key()...))
		}

		// 删除消息 key（m:...）.
		mIt := txn.NewIterator(badger.DefaultIteratorOptions)
		defer mIt.Close()
		mPrefix := []byte("m:")
		for mIt.Seek(mPrefix); mIt.ValidForPrefix(mPrefix); mIt.Next() {
			key := string(mIt.Item().Key())
			_, appIDFromMsg, _, _ := models.ParseMessageID(key)
			if strings.ToLower(strings.TrimSpace(appIDFromMsg)) != target {
				continue
			}
			keys = append(keys, append([]byte(nil), mIt.Item().Key()...))
		}

		// 删除二级索引 key（si:app:<app_id>:...）。
		iIt := txn.NewIterator(badger.DefaultIteratorOptions)
		defer iIt.Close()
		iPrefix := []byte(fmt.Sprintf("si:app:%s:", appID))
		for iIt.Seek(iPrefix); iIt.ValidForPrefix(iPrefix); iIt.Next() {
			keys = append(keys, append([]byte(nil), iIt.Item().Key()...))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return keys, nil
}

func deleteBadgerKeys(keys [][]byte) error {
	kv := store.GetStore()
	if kv == nil || len(keys) == 0 {
		return nil
	}
	const batchSize = 500
	for start := 0; start < len(keys); start += batchSize {
		end := start + batchSize
		if end > len(keys) {
			end = len(keys)
		}
		if err := kv.Update(func(txn *badger.Txn) error {
			for _, key := range keys[start:end] {
				if err := txn.Delete(key); err != nil && err != badger.ErrKeyNotFound {
					return err
				}
			}
			return nil
		}); err != nil {
			return err
		}
	}
	return nil
}

func collectAppKnowledgeCollections(appID string) []string {
	if store.DB == nil {
		return nil
	}
	var rows []models.KnowledgeBase
	if err := store.DB.Model(&models.KnowledgeBase{}).
		Select("collection").
		Where("app_id = ?", appID).
		Find(&rows).Error; err != nil {
		logger.Errorf("collect app knowledge collections failed app_id=%s err=%v", appID, err)
		return nil
	}
	seen := make(map[string]struct{}, len(rows))
	result := make([]string, 0, len(rows))
	for _, row := range rows {
		collection := strings.TrimSpace(row.Collection)
		if collection == "" {
			continue
		}
		if _, ok := seen[collection]; ok {
			continue
		}
		seen[collection] = struct{}{}
		result = append(result, collection)
	}
	return result
}

func cleanupKnowledgeCollections(collections []string) {
	if len(collections) == 0 {
		return
	}
	vdb := service.GetVectorStore()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for _, collection := range collections {
		if err := vdb.DeleteCollection(ctx, collection); err != nil {
			logger.Errorf("delete knowledge collection failed collection=%s err=%v", collection, err)
		}
	}
}

// GetApps 获取应用列表
func (ac *AppController) GetApps(c *gin.Context) {
	userName, role := getAuthUser(c)
	role = strings.TrimSpace(strings.ToLower(role))
	if role != "admin" && role != "agent" {
		logger.Errorf("get apps role denied role=%s user=%s", role, userName)
		response.ResponseError(c, http.StatusForbidden, response.ErrCodePermissionRoleDenied)
		return
	}

	// 解析查询参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	keyword := c.Query("keyword")
	statusStr := c.Query("status")
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 500 {
		pageSize = 10
	}

	// 构建查询
	query := store.DB.Model(&models.App{})

	// 非管理员仅允许查看自己可访问的应用列表，避免前端 403。
	if role != "admin" {
		us := service.GetUserService()
		if us == nil {
			logger.Errorf("get apps user service unavailable user=%s", userName)
			response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeUserServiceUnavailable)
			return
		}
		user, err := us.GetUser(userName)
		if err != nil || user == nil {
			logger.Errorf("get apps current user not found user=%s err=%v", userName, err)
			response.ResponseError(c, http.StatusUnauthorized, response.ErrCodeAuthContextMissing)
			return
		}

		rawApps := strings.TrimSpace(user.Apps)
		allowedAppIDs := make([]string, 0, 16)
		if rawApps != "" {
			var parsed []string
			if err := json.Unmarshal([]byte(rawApps), &parsed); err == nil {
				seen := make(map[string]struct{}, len(parsed))
				hasAll := false
				for _, item := range parsed {
					appID := strings.TrimSpace(item)
					if appID == "" {
						continue
					}
					if strings.EqualFold(appID, "all") {
						hasAll = true
						break
					}
					if _, ok := seen[appID]; ok {
						continue
					}
					seen[appID] = struct{}{}
					allowedAppIDs = append(allowedAppIDs, appID)
				}
				if !hasAll {
					if len(allowedAppIDs) == 0 {
						response.ResponseSuccess(c, gin.H{
							"data":  []models.App{},
							"total": 0,
						})
						return
					}
					query = query.Where("app_id IN ?", allowedAppIDs)
				}
			} else {
				// 历史脏数据兜底：非结构化 apps 文本只做 all 检测，其它场景直接拒绝返回空集。
				if !strings.Contains(strings.ToLower(rawApps), "all") {
					response.ResponseSuccess(c, gin.H{
						"data":  []models.App{},
						"total": 0,
					})
					return
				}
			}
		} else {
			response.ResponseSuccess(c, gin.H{
				"data":  []models.App{},
				"total": 0,
			})
			return
		}
	}

	// 关键词搜索
	if keyword != "" {
		query = query.Where("name LIKE ? OR app_id LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}

	// 状态筛选
	if statusStr != "" {
		status, err := strconv.Atoi(statusStr)
		if err == nil {
			query = query.Where("status = ?", status)
		}
	}

	// 计算总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		logger.Errorf("count apps failed user=%s role=%s err=%v", userName, role, err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeAppQueryFailed)
		return
	}

	// 分页查询
	offset := (page - 1) * pageSize
	var apps []models.App
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&apps).Error; err != nil {
		logger.Errorf("get apps failed user=%s role=%s err=%v", userName, role, err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeAppListFailed)
		return
	}

	logger.Infof("get apps successful user=%s role=%s page=%d page_size=%d total=%d", userName, role, page, pageSize, total)
	response.ResponseSuccess(c, gin.H{
		"data":  apps,
		"total": total,
	})
}

// CreateApp 创建应用
func (ac *AppController) CreateApp(c *gin.Context) {
	// 检查管理员权限
	if !IsAdmin(c) {
		logger.Errorf("permission denied, not admin")
		response.ResponseError(c, http.StatusForbidden, response.ErrCodePermissionAdminRequired)
		return
	}

	var req AppRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Errorf("create app request parameter error: %v", err)
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeAppCreateInvalidParams)
		return
	}

	// 生成 AppID（如果未提供）
	appID := req.AppID
	if appID == "" {
		appID = models.GenAppID()
	}

	// 检查 AppID 是否已存在
	var existingApp models.App
	if err := store.DB.Where("app_id = ?", appID).First(&existingApp).Error; err == nil {
		logger.Errorf("app id already exists: %s", appID)
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeAppIDDuplicated)
		return
	}

	// 创建应用
	app := models.App{
		Name:        req.Name,
		AppID:       appID,
		Logo:        req.Logo,
		AllowDomain: req.AllowDomain,
		WelcomeMsg:  req.WelcomeMsg,
		Contact:     req.Contact,
		Status:      req.Status,
	}

	if err := store.DB.Create(&app).Error; err != nil {
		logger.Errorf("create app failed: %v", err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeAppCreateFailed)
		return
	}

	logger.Infof("create app successful: %s", app.Name)
	RecordAudit(c, "app.create", "app", app.AppID, "success", app.Name)
	response.ResponseSuccess(c, app)
}

// UpdateApp 更新应用
func (ac *AppController) UpdateApp(c *gin.Context) {
	// 检查管理员权限
	if !IsAdmin(c) {
		logger.Errorf("permission denied, not admin")
		response.ResponseError(c, http.StatusForbidden, response.ErrCodePermissionAdminRequired)
		return
	}

	var req struct {
		AppID       string  `json:"app_id" binding:"required"`
		Name        *string `json:"name"`
		Logo        *string `json:"logo"`
		AllowDomain *string `json:"allow_domain"`
		WelcomeMsg  *string `json:"welcome_msg"`
		Contact     *string `json:"contact"`
		Status      *int    `json:"status"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Errorf("update app request parameter error: %v", err)
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeAppUpdateInvalidParams)
		return
	}

	// 检查应用是否存在
	var app models.App
	if err := store.DB.Where("app_id = ?", req.AppID).First(&app).Error; err != nil {
		logger.Errorf("app not found: %s", req.AppID)
		response.ResponseError(c, http.StatusNotFound, response.ErrCodeAppNotFound)
		return
	}

	// 更新应用（支持部分更新）
	updates := map[string]interface{}{}
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Logo != nil {
		updates["logo"] = *req.Logo
	}
	if req.AllowDomain != nil {
		updates["allow_domain"] = *req.AllowDomain
	}
	if req.WelcomeMsg != nil {
		updates["welcome_msg"] = *req.WelcomeMsg
	}
	if req.Contact != nil {
		updates["contact"] = *req.Contact
	}
	if req.Status != nil && (*req.Status == 0 || *req.Status == 1) {
		updates["status"] = *req.Status
	}
	if len(updates) == 0 {
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeAppUpdateNoChanges)
		return
	}

	if err := store.DB.Model(&app).Updates(updates).Error; err != nil {
		logger.Errorf("update app failed: %v", err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeAppUpdateFailed)
		return
	}

	// 重新获取更新后的数据
	if err := store.DB.Where("app_id = ?", req.AppID).First(&app).Error; err != nil {
		logger.Errorf("get updated app failed: %v", err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeAppQueryFailed)
		return
	}

	logger.Infof("update app successful: %s", req.AppID)
	RecordAudit(c, "app.update", "app", req.AppID, "success", app.Name)
	response.ResponseSuccess(c, app)
}

// DeleteApp 删除应用
func (ac *AppController) DeleteApp(c *gin.Context) {
	// 检查管理员权限
	if !IsAdmin(c) {
		logger.Errorf("permission denied, not admin")
		response.ResponseError(c, http.StatusForbidden, response.ErrCodePermissionAdminRequired)
		return
	}

	// 优先从 JSON body 读取，兼容历史 query 传参。
	var req struct {
		AppID string `json:"app_id"`
	}
	_ = c.ShouldBindJSON(&req)
	appID := strings.TrimSpace(req.AppID)
	if appID == "" {
		appID = strings.TrimSpace(c.Query("app_id"))
	}
	if appID == "" {
		logger.Errorf("app_id is required")
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeAppDeleteInvalidParams)
		return
	}

	// 检查应用是否存在
	var app models.App
	if err := store.DB.Where("app_id = ?", appID).First(&app).Error; err != nil {
		logger.Errorf("app not found: %s", appID)
		response.ResponseError(c, http.StatusNotFound, response.ErrCodeAppNotFound)
		return
	}
	collections := collectAppKnowledgeCollections(appID)

	keys, err := collectBadgerDeleteKeysByApp(appID)
	if err != nil {
		logger.Errorf("collect app badger keys failed app_id=%s err=%v", appID, err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeAppDeleteFailed)
		return
	}

	if err := ac.hardDeleteAppRelatedRows(appID); err != nil {
		logger.Errorf("delete app related rows failed app_id=%s err=%v", appID, err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeAppDeleteFailed)
		return
	}

	// 硬删除应用主体记录。
	if err := store.DB.Unscoped().Delete(&app).Error; err != nil {
		logger.Errorf("delete app row failed app_id=%s err=%v", appID, err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeAppDeleteFailed)
		return
	}

	// 删除 Badger 中关联会话与消息（best effort）。
	if err := deleteBadgerKeys(keys); err != nil {
		logger.Errorf("delete app badger keys failed app_id=%s err=%v", appID, err)
	}
	cleanupKnowledgeCollections(collections)

	logger.Infof("delete app successful: %s", appID)
	RecordAudit(c, "app.delete", "app", appID, "success", app.Name)
	response.ResponseSuccess(c, gin.H{"message": "delete successful"})
}

// GetConfig 获取应用配置（前端 widget 接入接口）
func (ac *AppController) GetConfig(c *gin.Context) {
	// 获取请求参数
	appID := c.Query("appid")
	if appID == "" {
		logger.Errorf("appid is required")
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeAppConfigInvalidParams)
		return
	}

	// 获取 Origin 和 Referer 头
	origin := c.GetHeader("Origin")
	referer := c.GetHeader("Referer")

	// 检查请求来源
	if origin == "" && referer == "" {
		logger.Errorf("origin or referer is required")
		response.ResponseError(c, http.StatusForbidden, response.ErrCodeAppConfigSourceRequired)
		return
	}

	// 获取应用信息
	var app models.App
	if err := store.DB.Where("app_id = ? AND status = ?", appID, 1).First(&app).Error; err != nil {
		logger.Errorf("app not found or disabled: %s", appID)
		response.ResponseError(c, http.StatusForbidden, response.ErrCodeAppNotFoundOrDisabled)
		return
	}

	// 检查域名是否在允许列表中
	if !models.IsDomainAllowed(origin, referer, app.AllowDomain) {
		logger.Errorf("domain not allowed: origin=%s, referer=%s", origin, referer)
		response.ResponseError(c, http.StatusForbidden, response.ErrCodeAppDomainNotAllowed)
		return
	}

	// 返回配置信息
	response.ResponseSuccess(c, gin.H{
		"name":        app.Name,
		"logo":        app.Logo,
		"welcome_msg": app.WelcomeMsg,
		"online":      models.HasOnlineAgentForApp(app.AppID),
	})
}
