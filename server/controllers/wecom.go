package controllers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xen0n/go-workwx"

	"kefu-server/models"
	"kefu-server/store"
	"kefu-server/utils/logger"
	"kefu-server/utils/response"
)

type WecomController struct{}

type wecomConfigRequest struct {
	CorpID  string `json:"corpId"`
	AgentID int    `json:"agentId"`
	Secret  string `json:"secret"`
}

type wecomConfigResponse struct {
	CorpID      string `json:"corpId"`
	AgentID     int    `json:"agentId"`
	HasSecret   bool   `json:"hasSecret"`
	CallbackURL string `json:"callbackUrl"`
}

type wecomTestRequest struct {
	CorpID  string `json:"corpId"`
	AgentID int    `json:"agentId"`
	Secret  string `json:"secret"`
}

type wecomTestResponse struct {
	Success bool   `json:"success"`
	Name    string `json:"name"`
	Error   string `json:"error"`
}

type wecomQrcodeResponse struct {
	CorpID      string `json:"corpId"`
	AgentID     int    `json:"agentId"`
	RedirectURI string `json:"redirectUri"`
	State       string `json:"state"`
}

type wecomBindStatusResponse struct {
	Status string `json:"status"`
	UserID string `json:"userId,omitempty"`
	Error  string `json:"error,omitempty"`
}

type wecomBindInfoResponse struct {
	IsBound  bool       `json:"isBound"`
	UserID   string     `json:"userId,omitempty"`
	BindTime *time.Time `json:"bindTime,omitempty"`
}

var bindStateCache = struct {
	mu    sync.RWMutex
	items map[string]*bindState
}{
	items: make(map[string]*bindState),
}

type bindState struct {
	status  string
	userID  string
	agentID uint
	expires time.Time
}

type bindStateRecord struct {
	Status  string    `json:"status"`
	UserID  string    `json:"user_id"`
	AgentID uint      `json:"agent_id"`
	Expires time.Time `json:"expires"`
}

const wecomBindStateSettingPrefix = "wecom_bind_state:"

func init() {
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			cleanExpiredBindStates()
		}
	}()
}

func cleanExpiredBindStates() {
	bindStateCache.mu.Lock()
	defer bindStateCache.mu.Unlock()
	now := time.Now()
	for key, state := range bindStateCache.items {
		if now.After(state.expires) {
			delete(bindStateCache.items, key)
			deletePersistedWecomBindState(key)
		}
	}
}

func wecomBindStateSettingKey(state string) string {
	return wecomBindStateSettingPrefix + state
}

func persistWecomBindState(state string, item *bindState) {
	if store.DB == nil || strings.TrimSpace(state) == "" || item == nil {
		return
	}

	record := bindStateRecord{
		Status:  item.status,
		UserID:  item.userID,
		AgentID: item.agentID,
		Expires: item.expires,
	}
	valueBytes, err := json.Marshal(record)
	if err != nil {
		logger.Errorf("wecom bind state marshal failed state=%s err=%v", state, err)
		return
	}

	key := wecomBindStateSettingKey(state)
	var setting models.SystemSetting
	result := store.DB.Where("key = ?", key).First(&setting)
	if result.Error != nil {
		setting.Key = key
		setting.Value = string(valueBytes)
		if err := store.DB.Create(&setting).Error; err != nil {
			logger.Errorf("wecom bind state create failed state=%s err=%v", state, err)
		}
		return
	}

	setting.Value = string(valueBytes)
	if err := store.DB.Save(&setting).Error; err != nil {
		logger.Errorf("wecom bind state update failed state=%s err=%v", state, err)
	}
}

func loadPersistedWecomBindState(state string) (*bindState, bool) {
	if store.DB == nil || strings.TrimSpace(state) == "" {
		return nil, false
	}

	var setting models.SystemSetting
	if err := store.DB.Where("key = ?", wecomBindStateSettingKey(state)).First(&setting).Error; err != nil {
		return nil, false
	}

	var record bindStateRecord
	if err := json.Unmarshal([]byte(setting.Value), &record); err != nil {
		logger.Errorf("wecom bind state unmarshal failed state=%s err=%v", state, err)
		deletePersistedWecomBindState(state)
		return nil, false
	}
	if time.Now().After(record.Expires) {
		deletePersistedWecomBindState(state)
		return nil, false
	}

	return &bindState{
		status:  record.Status,
		userID:  record.UserID,
		agentID: record.AgentID,
		expires: record.Expires,
	}, true
}

func deletePersistedWecomBindState(state string) {
	if store.DB == nil || strings.TrimSpace(state) == "" {
		return
	}
	if err := store.DB.Where("key = ?", wecomBindStateSettingKey(state)).Delete(&models.SystemSetting{}).Error; err != nil {
		logger.Errorf("wecom bind state delete failed state=%s err=%v", state, err)
	}
}

func getWecomBindState(state string) (*bindState, bool) {
	bindStateCache.mu.RLock()
	item, exists := bindStateCache.items[state]
	bindStateCache.mu.RUnlock()
	if exists {
		if time.Now().After(item.expires) {
			bindStateCache.mu.Lock()
			delete(bindStateCache.items, state)
			bindStateCache.mu.Unlock()
			deletePersistedWecomBindState(state)
			return nil, false
		}
		return item, true
	}

	item, exists = loadPersistedWecomBindState(state)
	if !exists {
		return nil, false
	}

	bindStateCache.mu.Lock()
	bindStateCache.items[state] = item
	bindStateCache.mu.Unlock()
	return item, true
}

func (c *WecomController) GetConfig(ctx *gin.Context) {
	cfg := getSystemSettingsCached()

	scheme := "http"
	if ctx.Request.TLS != nil || ctx.GetHeader("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	host := ctx.Request.Host
	callbackURL := fmt.Sprintf("%s://%s/api/wecom/callback", scheme, host)

	resp := wecomConfigResponse{
		CorpID:      cfg.WecomCorpID,
		AgentID:     cfg.WecomAgentID,
		HasSecret:   cfg.WecomSecret != "",
		CallbackURL: callbackURL,
	}

	response.ResponseSuccess(ctx, resp)
}

func (c *WecomController) SaveConfig(ctx *gin.Context) {
	var req wecomConfigRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.ResponseError(ctx, http.StatusBadRequest, response.ErrCodeWecomConfigInvalid)
		return
	}

	cfg := loadSystemSettingsFromDB()
	cfg.WecomCorpID = strings.TrimSpace(req.CorpID)
	cfg.WecomAgentID = req.AgentID
	cfg.WecomSecret = strings.TrimSpace(req.Secret)

	cfgBytes, err := json.Marshal(cfg)
	if err != nil {
		response.ResponseError(ctx, http.StatusInternalServerError, response.ErrCodeWecomConfigSerializeFailed)
		return
	}

	var setting models.SystemSetting
	result := store.DB.Where("key = ?", "system_settings").First(&setting)
	if result.Error != nil {
		setting.Key = "system_settings"
		setting.Value = string(cfgBytes)
		if err := store.DB.Create(&setting).Error; err != nil {
			response.ResponseError(ctx, http.StatusInternalServerError, response.ErrCodeWecomConfigSaveFailed)
			return
		}
	} else {
		setting.Value = string(cfgBytes)
		if err := store.DB.Save(&setting).Error; err != nil {
			response.ResponseError(ctx, http.StatusInternalServerError, response.ErrCodeWecomConfigUpdateFailed)
			return
		}
	}

	setSystemSettingsCache(cfg)
	response.ResponseSuccessWithMsg(ctx, "保存成功", nil)
}

func (c *WecomController) TestConnection(ctx *gin.Context) {
	var req wecomTestRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.ResponseError(ctx, http.StatusBadRequest, response.ErrCodeWecomConfigInvalid)
		return
	}

	corpID := strings.TrimSpace(req.CorpID)
	secret := strings.TrimSpace(req.Secret)
	if corpID == "" || secret == "" {
		response.ResponseError(ctx, http.StatusBadRequest, response.ErrCodeWecomCredentialsRequired)
		return
	}

	app := workwx.New(corpID)
	appApp := app.WithApp(secret, int64(req.AgentID))

	_, err := appApp.ListDepts(1)
	if err != nil {
		resp := wecomTestResponse{
			Success: false,
			Error:   fmt.Sprintf("连接测试失败: %v", err),
		}
		response.ResponseSuccess(ctx, resp)
		return
	}

	resp := wecomTestResponse{
		Success: true,
		Name:    "企业微信连接成功",
	}
	response.ResponseSuccess(ctx, resp)
}

func (c *WecomController) GetQrcode(ctx *gin.Context) {
	uid, ok := getAuthUserID(ctx)
	if !ok {
		response.ResponseError(ctx, http.StatusUnauthorized, response.ErrCodeAuthUnauthorized)
		return
	}

	cfg := getSystemSettingsCached()
	if cfg.WecomCorpID == "" || cfg.WecomSecret == "" {
		response.ResponseError(ctx, http.StatusBadRequest, response.ErrCodeWecomNotConfigured)
		return
	}

	state := generateBindState(uid)
	item := &bindState{
		status:  "pending",
		agentID: uid,
		expires: time.Now().Add(5 * time.Minute),
	}
	bindStateCache.mu.Lock()
	bindStateCache.items[state] = item
	bindStateCache.mu.Unlock()
	persistWecomBindState(state, item)

	scheme := "http"
	if ctx.Request.TLS != nil || ctx.GetHeader("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	host := ctx.Request.Host
	redirectURI := fmt.Sprintf("%s://%s/api/wecom/callback", scheme, host)

	resp := wecomQrcodeResponse{
		CorpID:      cfg.WecomCorpID,
		AgentID:     cfg.WecomAgentID,
		RedirectURI: redirectURI,
		State:       state,
	}

	response.ResponseSuccess(ctx, resp)
}

func (c *WecomController) GetBindStatus(ctx *gin.Context) {
	state := ctx.Query("state")
	if state == "" {
		response.ResponseError(ctx, http.StatusBadRequest, response.ErrCodeWecomBindStateRequired)
		return
	}

	item, exists := getWecomBindState(state)
	if !exists {
		response.ResponseSuccess(ctx, wecomBindStatusResponse{Status: "expired", Error: "二维码已过期"})
		return
	}

	resp := wecomBindStatusResponse{
		Status: item.status,
		UserID: item.userID,
	}

	response.ResponseSuccess(ctx, resp)
}

func (c *WecomController) GetBindInfo(ctx *gin.Context) {
	uid, ok := getAuthUserID(ctx)
	if !ok {
		response.ResponseError(ctx, http.StatusUnauthorized, response.ErrCodeAuthUnauthorized)
		return
	}

	var user models.User
	if err := store.DB.First(&user, uid).Error; err != nil {
		response.ResponseError(ctx, http.StatusInternalServerError, response.ErrCodeWecomBindInfoFailed)
		return
	}

	resp := wecomBindInfoResponse{
		IsBound:  user.WecomBindStatus == 1,
		UserID:   user.WecomUserID,
		BindTime: user.WecomBindTime,
	}

	response.ResponseSuccess(ctx, resp)
}

func (c *WecomController) Unbind(ctx *gin.Context) {
	uid, ok := getAuthUserID(ctx)
	if !ok {
		response.ResponseError(ctx, http.StatusUnauthorized, response.ErrCodeAuthUnauthorized)
		return
	}

	if err := store.DB.Model(&models.User{}).Where("id = ?", uid).Updates(map[string]interface{}{
		"wecom_userid":      "",
		"wecom_bind_status": 0,
		"wecom_bind_time":   nil,
	}).Error; err != nil {
		response.ResponseError(ctx, http.StatusInternalServerError, response.ErrCodeWecomUnbindFailed)
		return
	}

	response.ResponseSuccessWithMsg(ctx, "解绑成功", nil)
}

func (c *WecomController) Callback(ctx *gin.Context) {
	code := ctx.Query("code")
	state := ctx.Query("state")

	if code == "" || state == "" {
		ctx.String(http.StatusBadRequest, "缺少必要参数")
		return
	}

	item, exists := getWecomBindState(state)
	if !exists || time.Now().After(item.expires) {
		ctx.String(http.StatusBadRequest, "二维码已过期，请重新扫码")
		return
	}

	cfg := getSystemSettingsCached()
	if cfg.WecomCorpID == "" || cfg.WecomSecret == "" {
		ctx.String(http.StatusInternalServerError, "企业微信配置缺失")
		return
	}

	app := workwx.New(cfg.WecomCorpID)
	appApp := app.WithApp(cfg.WecomSecret, int64(cfg.WecomAgentID))

	userInfo, err := appApp.GetUserInfoByCode(code)
	if err != nil {
		logger.Errorf("get wecom user info failed: %v", err)
		ctx.String(http.StatusInternalServerError, "获取用户信息失败")
		return
	}

	userID := userInfo.UserID
	if userID == "" {
		userID = userInfo.OpenID
	}

	now := time.Now()
	if err := store.DB.Model(&models.User{}).Where("id = ?", item.agentID).Updates(map[string]interface{}{
		"wecom_userid":      userID,
		"wecom_bind_status": 1,
		"wecom_bind_time":   now,
	}).Error; err != nil {
		logger.Errorf("update user wecom bind info failed: %v", err)
		ctx.String(http.StatusInternalServerError, "绑定失败")
		return
	}

	successState := &bindState{
		status:  "success",
		userID:  userID,
		agentID: item.agentID,
		expires: item.expires,
	}
	bindStateCache.mu.Lock()
	bindStateCache.items[state] = successState
	bindStateCache.mu.Unlock()
	persistWecomBindState(state, successState)

	ctx.String(http.StatusOK, "绑定成功！您可以关闭此页面。")
}

func generateBindState(agentID uint) string {
	timestamp := time.Now().Unix()
	randomBytes := make([]byte, 8)
	rand.Read(randomBytes)
	random := hex.EncodeToString(randomBytes)
	state := fmt.Sprintf("bind_%d_%d_%s", agentID, timestamp, random)

	bindStateCache.mu.Lock()
	bindStateCache.items[state] = &bindState{
		status:  "pending",
		agentID: agentID,
		expires: time.Now().Add(5 * time.Minute),
	}
	bindStateCache.mu.Unlock()

	return state
}

func SendWecomMessage(wecomUserID, title, content string) error {
	cfg := getSystemSettingsCached()
	if cfg.WecomCorpID == "" || cfg.WecomSecret == "" {
		return fmt.Errorf("企业微信未配置")
	}

	app := workwx.New(cfg.WecomCorpID)
	appApp := app.WithApp(cfg.WecomSecret, int64(cfg.WecomAgentID))

	recipient := &workwx.Recipient{
		UserIDs: []string{wecomUserID},
	}

	msgContent := fmt.Sprintf("%s\n\n%s", title, content)
	err := appApp.SendTextMessage(recipient, msgContent, false)
	if err != nil {
		return fmt.Errorf("发送消息失败: %v", err)
	}

	return nil
}

func init() {
	_ = fmt.Sprintf("")
}
