package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"kefu-server/models"
	"kefu-server/notification"
	"kefu-server/store"
	"kefu-server/utils/logger"
	"kefu-server/utils/response"
)

type NotificationController struct {
	manager *notification.Manager
}

func NewNotificationController() *NotificationController {
	return &NotificationController{
		manager: notification.GetManager(),
	}
}

func loadNotificationConfigs(manager *notification.Manager) {
	if manager == nil || store.DB == nil {
		return
	}
	var setting models.SystemSetting
	if err := store.DB.Where("key = ?", "notification_settings").First(&setting).Error; err != nil {
		return
	}

	var configData map[string]interface{}
	if err := json.Unmarshal([]byte(setting.Value), &configData); err != nil {
		return
	}

	manager.LoadConfigs(configData)
}

func (c *NotificationController) loadNotificationConfigs() {
	loadNotificationConfigs(c.manager)
}

func cloneConfigMap(source map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{}, len(source))
	for k, v := range source {
		result[k] = v
	}
	return result
}

func mergeSecretConfig(current map[string]interface{}, incoming map[string]interface{}) map[string]interface{} {
	merged := cloneConfigMap(current)
	for k, v := range incoming {
		merged[k] = v
	}
	for _, key := range []string{"secret", "appSecret", "password"} {
		value, ok := merged[key].(string)
		if ok && strings.TrimSpace(value) != "" {
			continue
		}
		if oldValue, ok := current[key].(string); ok && strings.TrimSpace(oldValue) != "" {
			merged[key] = oldValue
		}
	}
	return merged
}

func buildNotificationSettingsPayload(manager *notification.Manager) map[string]interface{} {
	payload := make(map[string]interface{})
	if manager == nil {
		return payload
	}
	for channelType, cfg := range manager.GetAllConfigs() {
		item := map[string]interface{}{
			"enabled": cfg.Enabled,
			"config":  cloneConfigMap(cfg.Config),
		}
		if len(cfg.Metadata) > 0 {
			item["metadata"] = cfg.Metadata
		}
		payload[string(channelType)] = item
	}
	return payload
}

func isSupportedAgentBindChannel(channelType notification.ChannelType) bool {
	return channelType == notification.ChannelWecom
}

func buildBindQrcodePayload(channelType notification.ChannelType, bindURL string, state string) map[string]string {
	payload := map[string]string{
		"url":   bindURL,
		"state": state,
	}
	if channelType != notification.ChannelWecom {
		return payload
	}

	parsedURL, err := url.Parse(bindURL)
	if err != nil {
		return payload
	}
	query := parsedURL.Query()

	if corpID := firstNonEmpty(query.Get("appid"), query.Get("corpid"), query.Get("corpId")); corpID != "" {
		payload["corpId"] = corpID
	}
	if agentID := firstNonEmpty(query.Get("agentid"), query.Get("agentId")); agentID != "" {
		payload["agentId"] = agentID
	}
	if redirectURI := firstNonEmpty(query.Get("redirect_uri"), query.Get("redirectUri")); redirectURI != "" {
		payload["redirectUri"] = redirectURI
	}
	return payload
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

type ChannelConfigRequest struct {
	Channel string                 `json:"channel"`
	Config  map[string]interface{} `json:"config"`
	Enabled bool                   `json:"enabled"`
}

func (c *NotificationController) GetChannels(ctx *gin.Context) {
	c.loadNotificationConfigs()

	channels := notification.GetSupportedChannels()
	configs := c.manager.GetAllConfigs()

	result := make([]map[string]interface{}, 0)
	for _, channelType := range channels {
		notifier, _ := notification.GetNotifier(channelType)

		config := map[string]interface{}{
			"type":    string(channelType),
			"name":    notifier.Name(),
			"enabled": false,
		}

		if cfg, exists := configs[channelType]; exists {
			config["enabled"] = cfg.Enabled
			config["configured"] = true

			safeConfig := make(map[string]interface{})
			for k, v := range cfg.Config {
				if k != "secret" && k != "appSecret" && k != "password" {
					safeConfig[k] = v
				}
			}
			config["config"] = safeConfig
		} else {
			config["configured"] = false
		}

		result = append(result, config)
	}

	response.ResponseSuccess(ctx, result)
}

func (c *NotificationController) GetChannelConfig(ctx *gin.Context) {
	channelStr := ctx.Param("channel")
	channelType := notification.ChannelType(channelStr)

	notifier, exists := notification.GetNotifier(channelType)
	if !exists {
		response.ResponseError(ctx, http.StatusBadRequest, response.ErrCodeNotificationChannelUnsupported)
		return
	}

	c.loadNotificationConfigs()

	config, exists := c.manager.GetConfig(channelType)

	result := map[string]interface{}{
		"type":    string(channelType),
		"name":    notifier.Name(),
		"enabled": false,
	}

	if exists {
		result["enabled"] = config.Enabled

		safeConfig := make(map[string]interface{})
		for k, v := range config.Config {
			if k != "secret" && k != "appSecret" && k != "password" {
				safeConfig[k] = v
			}
		}
		result["config"] = safeConfig
		result["hasSecret"] = hasSecret(config.Config)
	}

	scheme := "http"
	if ctx.Request.TLS != nil || ctx.GetHeader("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	host := ctx.Request.Host
	result["callbackUrl"] = fmt.Sprintf("%s://%s/api/notification/callback/%s", scheme, host, channelStr)

	response.ResponseSuccess(ctx, result)
}

func hasSecret(config map[string]interface{}) bool {
	for _, key := range []string{"secret", "appSecret", "password"} {
		if val, ok := config[key].(string); ok && val != "" {
			return true
		}
	}
	return false
}

func (c *NotificationController) SaveChannelConfig(ctx *gin.Context) {
	channelStr := ctx.Param("channel")
	channelType := notification.ChannelType(channelStr)

	notifier, exists := notification.GetNotifier(channelType)
	if !exists {
		response.ResponseError(ctx, http.StatusBadRequest, response.ErrCodeNotificationChannelUnsupported)
		return
	}

	var req ChannelConfigRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.ResponseError(ctx, http.StatusBadRequest, response.ErrCodeInvalidParams)
		return
	}

	c.loadNotificationConfigs()
	mergedConfig := cloneConfigMap(req.Config)
	if existing, ok := c.manager.GetConfig(channelType); ok && existing != nil {
		mergedConfig = mergeSecretConfig(existing.Config, mergedConfig)
	}

	if err := notifier.ValidateConfig(mergedConfig); err != nil {
		response.ResponseErrorWithMsg(ctx, http.StatusBadRequest, response.ErrCodeInvalidParams, err.Error())
		return
	}

	config := &notification.ChannelConfig{
		Type:    channelType,
		Enabled: req.Enabled,
		Config:  mergedConfig,
	}

	c.manager.UpdateConfig(channelType, config)

	configData := buildNotificationSettingsPayload(c.manager)
	configBytes, err := json.Marshal(configData)
	if err != nil {
		response.ResponseError(ctx, http.StatusInternalServerError, response.ErrCodeSystemSettingsSaveFailed)
		return
	}

	var setting models.SystemSetting
	result := store.DB.Where("key = ?", "notification_settings").First(&setting)
	if result.Error != nil {
		setting.Key = "notification_settings"
		setting.Value = string(configBytes)
		if err := store.DB.Create(&setting).Error; err != nil {
			response.ResponseError(ctx, http.StatusInternalServerError, response.ErrCodeSystemSettingsSaveFailed)
			return
		}
	} else {
		setting.Value = string(configBytes)
		if err := store.DB.Save(&setting).Error; err != nil {
			response.ResponseError(ctx, http.StatusInternalServerError, response.ErrCodeSystemSettingsSaveFailed)
			return
		}
	}

	response.ResponseSuccessWithMsg(ctx, "保存成功", nil)
}

func (c *NotificationController) TestChannel(ctx *gin.Context) {
	channelStr := ctx.Param("channel")
	channelType := notification.ChannelType(channelStr)

	notifier, exists := notification.GetNotifier(channelType)
	if !exists {
		response.ResponseError(ctx, http.StatusBadRequest, response.ErrCodeNotificationChannelUnsupported)
		return
	}

	var req ChannelConfigRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.ResponseError(ctx, http.StatusBadRequest, response.ErrCodeInvalidParams)
		return
	}

	err := notifier.TestConnection(req.Config)
	if err != nil {
		response.ResponseSuccess(ctx, map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	response.ResponseSuccess(ctx, map[string]interface{}{
		"success": true,
		"message": "连接成功",
	})
}

func (c *NotificationController) GetBindQrcode(ctx *gin.Context) {
	channelStr := ctx.Param("channel")
	channelType := notification.ChannelType(channelStr)
	if !isSupportedAgentBindChannel(channelType) {
		response.ResponseError(ctx, http.StatusBadRequest, response.ErrCodeNotificationChannelUnsupported)
		return
	}

	uid, ok := getAuthUserID(ctx)
	if !ok {
		response.ResponseError(ctx, http.StatusUnauthorized, response.ErrCodeAuthUnauthorized)
		return
	}

	c.loadNotificationConfigs()

	config, exists := c.manager.GetConfig(channelType)
	if !exists || !config.Enabled {
		response.ResponseError(ctx, http.StatusBadRequest, response.ErrCodeNotificationChannelNotConfigured)
		return
	}

	scheme := "http"
	if ctx.Request.TLS != nil || ctx.GetHeader("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	host := ctx.Request.Host
	callbackURL := fmt.Sprintf("%s://%s/api/notification/callback/%s", scheme, host, channelStr)

	notifier, _ := notification.GetNotifier(channelType)
	bindURL, state, err := notifier.GetBindURL(config.Config, uid, callbackURL)
	if err != nil {
		response.ResponseErrorWithMsg(ctx, http.StatusInternalServerError, response.ErrCodeNotificationBindFailed, err.Error())
		return
	}

	c.manager.StoreBindState(state, &notification.BindState{
		Status:  "pending",
		AgentID: uid,
		Channel: channelType,
		Expires: time.Now().Add(5 * time.Minute),
	})

	response.ResponseSuccess(ctx, buildBindQrcodePayload(channelType, bindURL, state))
}

func (c *NotificationController) GetBindStatus(ctx *gin.Context) {
	channelType := notification.ChannelType(ctx.Param("channel"))
	if !isSupportedAgentBindChannel(channelType) {
		response.ResponseError(ctx, http.StatusBadRequest, response.ErrCodeNotificationChannelUnsupported)
		return
	}

	state := ctx.Query("state")
	if state == "" {
		response.ResponseError(ctx, http.StatusBadRequest, response.ErrCodeNotificationBindStateRequired)
		return
	}

	bindState, exists := c.manager.GetBindState(state)
	if !exists {
		response.ResponseSuccess(ctx, map[string]interface{}{
			"status": "expired",
			"error":  "二维码已过期",
		})
		return
	}

	if time.Now().After(bindState.Expires) {
		response.ResponseSuccess(ctx, map[string]interface{}{
			"status": "expired",
			"error":  "二维码已过期",
		})
		return
	}

	response.ResponseSuccess(ctx, map[string]interface{}{
		"status": bindState.Status,
		"userId": bindState.UserID,
	})
}

func (c *NotificationController) GetBindInfo(ctx *gin.Context) {
	channelType := notification.ChannelType(ctx.Param("channel"))
	if !isSupportedAgentBindChannel(channelType) {
		response.ResponseError(ctx, http.StatusBadRequest, response.ErrCodeNotificationChannelUnsupported)
		return
	}

	uid, ok := getAuthUserID(ctx)
	if !ok {
		response.ResponseError(ctx, http.StatusUnauthorized, response.ErrCodeAuthUnauthorized)
		return
	}

	var user models.User
	if err := store.DB.First(&user, uid).Error; err != nil {
		response.ResponseError(ctx, http.StatusInternalServerError, response.ErrCodeNotificationBindInfoFailed)
		return
	}

	response.ResponseSuccess(ctx, map[string]interface{}{
		"isBound":  user.WecomBindStatus == 1,
		"userId":   user.WecomUserID,
		"bindTime": user.WecomBindTime,
	})
}

func (c *NotificationController) Unbind(ctx *gin.Context) {
	channelType := notification.ChannelType(ctx.Param("channel"))
	if !isSupportedAgentBindChannel(channelType) {
		response.ResponseError(ctx, http.StatusBadRequest, response.ErrCodeNotificationChannelUnsupported)
		return
	}

	uid, ok := getAuthUserID(ctx)
	if !ok {
		response.ResponseError(ctx, http.StatusUnauthorized, response.ErrCodeAuthUnauthorized)
		return
	}

	updates := map[string]interface{}{
		"wecom_userid":      "",
		"wecom_bind_status": 0,
		"wecom_bind_time":   nil,
	}

	if err := store.DB.Model(&models.User{}).Where("id = ?", uid).Updates(updates).Error; err != nil {
		response.ResponseError(ctx, http.StatusInternalServerError, response.ErrCodeNotificationUnbindFailed)
		return
	}

	response.ResponseSuccessWithMsg(ctx, "解绑成功", nil)
}

func (c *NotificationController) HandleCallback(ctx *gin.Context) {
	channelStr := ctx.Param("channel")
	channelType := notification.ChannelType(channelStr)
	if !isSupportedAgentBindChannel(channelType) {
		ctx.String(http.StatusBadRequest, "当前渠道暂不支持个人绑定")
		return
	}

	code := ctx.Query("code")
	state := ctx.Query("state")

	if code == "" || state == "" {
		ctx.String(http.StatusBadRequest, "缺少必要参数")
		return
	}

	bindState, exists := c.manager.GetBindState(state)
	if !exists || time.Now().After(bindState.Expires) {
		ctx.String(http.StatusBadRequest, "二维码已过期，请重新扫码")
		return
	}
	if bindState.Channel != channelType {
		ctx.String(http.StatusBadRequest, "绑定渠道不匹配")
		return
	}

	c.loadNotificationConfigs()

	userID, err := c.manager.HandleCallback(channelType, code, state)
	if err != nil {
		logger.Errorf("handle callback failed: %v", err)
		ctx.String(http.StatusInternalServerError, "获取用户信息失败")
		return
	}

	updates := map[string]interface{}{
		"wecom_userid":      userID,
		"wecom_bind_status": 1,
		"wecom_bind_time":   time.Now(),
	}

	if err := store.DB.Model(&models.User{}).Where("id = ?", bindState.AgentID).Updates(updates).Error; err != nil {
		logger.Errorf("update user bind info failed: %v", err)
		ctx.String(http.StatusInternalServerError, "绑定失败")
		return
	}

	c.manager.UpdateBindState(state, "success", userID)

	notifier, _ := notification.GetNotifier(channelType)
	ctx.String(http.StatusOK, fmt.Sprintf("%s绑定成功！您可以关闭此页面。", notifier.Name()))
}

func SendNotificationToAgent(agentID uint, title, content string) error {
	var user models.User
	if err := store.DB.First(&user, agentID).Error; err != nil {
		return fmt.Errorf("查询用户失败: %v", err)
	}

	manager := notification.GetManager()
	loadNotificationConfigs(manager)

	for _, channelType := range notification.GetSupportedChannels() {
		if !manager.IsEnabled(channelType) {
			continue
		}
		if channelType != notification.ChannelWecom {
			continue
		}
		if user.WecomBindStatus != 1 || strings.TrimSpace(user.WecomUserID) == "" {
			continue
		}

		config, exists := manager.GetConfig(channelType)
		if !exists {
			continue
		}

		notifier, exists := notification.GetNotifier(channelType)
		if !exists {
			continue
		}

		childCtx := contextWithConfig(context.Background(), channelType, config.Config)

		if err := notifier.Send(childCtx, strings.TrimSpace(user.WecomUserID), &notification.Notification{
			Title:   title,
			Content: content,
		}); err != nil {
			logger.Errorf("send notification via %s failed: %v", channelType, err)
		}
	}

	return nil
}

func contextWithConfig(ctx context.Context, channelType notification.ChannelType, config map[string]interface{}) context.Context {
	switch channelType {
	case notification.ChannelWecom:
		cfg := &notification.WecomConfig{}
		if corpID, ok := config["corpId"].(string); ok {
			cfg.CorpID = strings.TrimSpace(corpID)
		}
		if agentID, ok := config["agentId"].(float64); ok {
			cfg.AgentID = int(agentID)
		}
		if secret, ok := config["secret"].(string); ok {
			cfg.Secret = strings.TrimSpace(secret)
		}
		return context.WithValue(ctx, "wecom_config", cfg)
	default:
		return ctx
	}
}

func init() {
	_ = fmt.Sprintf("")
}
