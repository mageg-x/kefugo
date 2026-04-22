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
		}
	}
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

	ctx.JSON(http.StatusOK, gin.H{"code": 0, "data": resp})
}

func (c *WecomController) SaveConfig(ctx *gin.Context) {
	var req wecomConfigRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	cfg := loadSystemSettingsFromDB()
	cfg.WecomCorpID = strings.TrimSpace(req.CorpID)
	cfg.WecomAgentID = req.AgentID
	cfg.WecomSecret = strings.TrimSpace(req.Secret)

	cfgBytes, err := json.Marshal(cfg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "配置序列化失败"})
		return
	}

	var setting models.SystemSetting
	result := store.DB.Where("key = ?", "system_settings").First(&setting)
	if result.Error != nil {
		setting.Key = "system_settings"
		setting.Value = string(cfgBytes)
		if err := store.DB.Create(&setting).Error; err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "保存配置失败"})
			return
		}
	} else {
		setting.Value = string(cfgBytes)
		if err := store.DB.Save(&setting).Error; err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新配置失败"})
			return
		}
	}

	setSystemSettingsCache(cfg)
	ctx.JSON(http.StatusOK, gin.H{"code": 0, "message": "保存成功"})
}

func (c *WecomController) TestConnection(ctx *gin.Context) {
	var req wecomTestRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	corpID := strings.TrimSpace(req.CorpID)
	secret := strings.TrimSpace(req.Secret)
	if corpID == "" || secret == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "CorpID 和 Secret 不能为空"})
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
		ctx.JSON(http.StatusOK, gin.H{"code": 0, "data": resp})
		return
	}

	resp := wecomTestResponse{
		Success: true,
		Name:    "企业微信连接成功",
	}
	ctx.JSON(http.StatusOK, gin.H{"code": 0, "data": resp})
}

func (c *WecomController) GetQrcode(ctx *gin.Context) {
	uid, ok := getAuthUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未登录"})
		return
	}

	cfg := getSystemSettingsCached()
	if cfg.WecomCorpID == "" || cfg.WecomSecret == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "error": "CONFIG_MISSING", "message": "企业微信功能未配置，请联系管理员"})
		return
	}

	state := generateBindState(uid)

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

	ctx.JSON(http.StatusOK, gin.H{"code": 0, "data": resp})
}

func (c *WecomController) GetBindStatus(ctx *gin.Context) {
	state := ctx.Query("state")
	if state == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "缺少 state 参数"})
		return
	}

	bindStateCache.mu.RLock()
	item, exists := bindStateCache.items[state]
	bindStateCache.mu.RUnlock()

	if !exists {
		ctx.JSON(http.StatusOK, gin.H{"code": 0, "data": wecomBindStatusResponse{Status: "expired", Error: "二维码已过期"}})
		return
	}

	if time.Now().After(item.expires) {
		ctx.JSON(http.StatusOK, gin.H{"code": 0, "data": wecomBindStatusResponse{Status: "expired", Error: "二维码已过期"}})
		return
	}

	resp := wecomBindStatusResponse{
		Status: item.status,
		UserID: item.userID,
	}

	ctx.JSON(http.StatusOK, gin.H{"code": 0, "data": resp})
}

func (c *WecomController) GetBindInfo(ctx *gin.Context) {
	uid, ok := getAuthUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未登录"})
		return
	}

	var user models.User
	if err := store.DB.First(&user, uid).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "查询用户失败"})
		return
	}

	resp := wecomBindInfoResponse{
		IsBound:  user.WecomBindStatus == 1,
		UserID:   user.WecomUserID,
		BindTime: user.WecomBindTime,
	}

	ctx.JSON(http.StatusOK, gin.H{"code": 0, "data": resp})
}

func (c *WecomController) Unbind(ctx *gin.Context) {
	uid, ok := getAuthUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未登录"})
		return
	}

	if err := store.DB.Model(&models.User{}).Where("id = ?", uid).Updates(map[string]interface{}{
		"wecom_userid":      "",
		"wecom_bind_status": 0,
		"wecom_bind_time":   nil,
	}).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "解绑失败"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"code": 0, "message": "解绑成功"})
}

func (c *WecomController) Callback(ctx *gin.Context) {
	code := ctx.Query("code")
	state := ctx.Query("state")

	if code == "" || state == "" {
		ctx.String(http.StatusBadRequest, "缺少必要参数")
		return
	}

	bindStateCache.mu.RLock()
	item, exists := bindStateCache.items[state]
	bindStateCache.mu.RUnlock()

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

	bindStateCache.mu.Lock()
	bindStateCache.items[state].status = "success"
	bindStateCache.items[state].userID = userID
	bindStateCache.mu.Unlock()

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
