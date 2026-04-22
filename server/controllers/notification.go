package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"kefu-server/models"
	"kefu-server/notification"
	"kefu-server/store"
	"kefu-server/utils/logger"
)

type NotificationController struct {
	manager *notification.Manager
}

func NewNotificationController() *NotificationController {
	return &NotificationController{
		manager: notification.GetManager(),
	}
}

func (c *NotificationController) loadNotificationConfigs() {
	var setting models.SystemSetting
	if err := store.DB.Where("key = ?", "notification_settings").First(&setting).Error; err != nil {
		return
	}

	var configData map[string]interface{}
	if err := json.Unmarshal([]byte(setting.Value), &configData); err != nil {
		return
	}

	c.manager.LoadConfigs(configData)
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

	ctx.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
}

func (c *NotificationController) GetChannelConfig(ctx *gin.Context) {
	channelStr := ctx.Param("channel")
	channelType := notification.ChannelType(channelStr)

	notifier, exists := notification.GetNotifier(channelType)
	if !exists {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "不支持的通知渠道"})
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

	ctx.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
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
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "不支持的通知渠道"})
		return
	}

	var req ChannelConfigRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	if err := notifier.ValidateConfig(req.Config); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	c.loadNotificationConfigs()

	config := &notification.ChannelConfig{
		Type:    channelType,
		Enabled: req.Enabled,
		Config:  req.Config,
	}

	c.manager.UpdateConfig(channelType, config)

	configData := make(map[string]interface{})
	for ch, cfg := range c.manager.GetAllConfigs() {
		configData[string(ch)] = cfg.Config
	}

	configBytes, err := json.Marshal(configData)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "配置序列化失败"})
		return
	}

	var setting models.SystemSetting
	result := store.DB.Where("key = ?", "notification_settings").First(&setting)
	if result.Error != nil {
		setting.Key = "notification_settings"
		setting.Value = string(configBytes)
		if err := store.DB.Create(&setting).Error; err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "保存配置失败"})
			return
		}
	} else {
		setting.Value = string(configBytes)
		if err := store.DB.Save(&setting).Error; err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新配置失败"})
			return
		}
	}

	ctx.JSON(http.StatusOK, gin.H{"code": 0, "message": "保存成功"})
}

func (c *NotificationController) TestChannel(ctx *gin.Context) {
	channelStr := ctx.Param("channel")
	channelType := notification.ChannelType(channelStr)

	notifier, exists := notification.GetNotifier(channelType)
	if !exists {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "不支持的通知渠道"})
		return
	}

	var req ChannelConfigRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	err := notifier.TestConnection(req.Config)
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{
			"code": 0,
			"data": map[string]interface{}{
				"success": false,
				"error":   err.Error(),
			},
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": map[string]interface{}{
			"success": true,
			"message": "连接成功",
		},
	})
}

func (c *NotificationController) GetBindQrcode(ctx *gin.Context) {
	channelStr := ctx.Param("channel")
	channelType := notification.ChannelType(channelStr)

	uid, ok := getAuthUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未登录"})
		return
	}

	c.loadNotificationConfigs()

	config, exists := c.manager.GetConfig(channelType)
	if !exists || !config.Enabled {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"error":   "CONFIG_MISSING",
			"message": "通知渠道未配置或未启用，请联系管理员",
		})
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
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.manager.StoreBindState(state, &notification.BindState{
		Status:  "pending",
		AgentID: uid,
		Channel: channelType,
		Expires: time.Now().Add(5 * time.Minute),
	})

	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": map[string]string{
			"url":   bindURL,
			"state": state,
		},
	})
}

func (c *NotificationController) GetBindStatus(ctx *gin.Context) {
	state := ctx.Query("state")
	if state == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "缺少 state 参数"})
		return
	}

	bindState, exists := c.manager.GetBindState(state)
	if !exists {
		ctx.JSON(http.StatusOK, gin.H{
			"code": 0,
			"data": map[string]interface{}{
				"status": "expired",
				"error":  "二维码已过期",
			},
		})
		return
	}

	if time.Now().After(bindState.Expires) {
		ctx.JSON(http.StatusOK, gin.H{
			"code": 0,
			"data": map[string]interface{}{
				"status": "expired",
				"error":  "二维码已过期",
			},
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": map[string]interface{}{
			"status": bindState.Status,
			"userId": bindState.UserID,
		},
	})
}

func (c *NotificationController) GetBindInfo(ctx *gin.Context) {
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

	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": map[string]interface{}{
			"isBound":  user.WecomBindStatus == 1,
			"userId":   user.WecomUserID,
			"bindTime": user.WecomBindTime,
		},
	})
}

func (c *NotificationController) Unbind(ctx *gin.Context) {
	uid, ok := getAuthUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未登录"})
		return
	}

	updates := map[string]interface{}{
		"wecom_userid":      "",
		"wecom_bind_status": 0,
		"wecom_bind_time":   nil,
	}

	if err := store.DB.Model(&models.User{}).Where("id = ?", uid).Updates(updates).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "解绑失败"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"code": 0, "message": "解绑成功"})
}

func (c *NotificationController) HandleCallback(ctx *gin.Context) {
	channelStr := ctx.Param("channel")
	channelType := notification.ChannelType(channelStr)

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
