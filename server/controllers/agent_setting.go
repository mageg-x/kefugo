package controllers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"kefu-server/models"
	"kefu-server/store"
	"kefu-server/utils/logger"
	"kefu-server/utils/response"
)

const (
	defaultAIModel  = "gpt-4o-mini"
	defaultAIStyle  = "professional"
	defaultAIPrompt = "you are a customer service assistant, answer accurately, briefly and politely."
)

type AgentSettingController struct{}

type agentSettingResponse struct {
	SoundEnabled           bool   `json:"soundEnabled"`
	DesktopNotifyEnabled   bool   `json:"desktopNotifyEnabled"`
	TypingIndicatorEnabled bool   `json:"typingIndicatorEnabled"`
	EnterToSend            bool   `json:"enterToSend"`
	AIEnabled              bool   `json:"aiEnabled"`
	AIModel                string `json:"aiModel"`
	AIStyle                string `json:"aiStyle"`
	AIPrompt               string `json:"aiPrompt"`
}

func normalizeAIStyle(style string) string {
	switch strings.ToLower(strings.TrimSpace(style)) {
	case "professional", "friendly", "formal":
		return strings.ToLower(strings.TrimSpace(style))
	default:
		return defaultAIStyle
	}
}

func normalizeAIModel(model string) string {
	model = strings.TrimSpace(model)
	if model == "" {
		return defaultAIModel
	}
	if len(model) > 64 {
		return model[:64]
	}
	return model
}

func normalizeAIPrompt(prompt string) string {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return defaultAIPrompt
	}
	if len(prompt) > 2000 {
		return prompt[:2000]
	}
	return prompt
}

func getOrCreateAgentSetting(userName string) (*models.AgentSetting, error) {
	item := &models.AgentSetting{}
	if err := store.DB.Where("user_name = ?", userName).First(item).Error; err == nil {
		item.AIModel = normalizeAIModel(item.AIModel)
		item.AIStyle = normalizeAIStyle(item.AIStyle)
		item.AIPrompt = normalizeAIPrompt(item.AIPrompt)
		return item, nil
	}

	item = &models.AgentSetting{
		UserName:               userName,
		SoundEnabled:           true,
		DesktopNotifyEnabled:   false,
		TypingIndicatorEnabled: true,
		EnterToSend:            true,
		AIEnabled:              false,
		AIModel:                defaultAIModel,
		AIStyle:                defaultAIStyle,
		AIPrompt:               defaultAIPrompt,
	}
	if err := store.DB.Create(item).Error; err != nil {
		return nil, err
	}
	return item, nil
}

func toAgentSettingResponse(item *models.AgentSetting) agentSettingResponse {
	return agentSettingResponse{
		SoundEnabled:           item.SoundEnabled,
		DesktopNotifyEnabled:   item.DesktopNotifyEnabled,
		TypingIndicatorEnabled: item.TypingIndicatorEnabled,
		EnterToSend:            item.EnterToSend,
		AIEnabled:              item.AIEnabled,
		AIModel:                normalizeAIModel(item.AIModel),
		AIStyle:                normalizeAIStyle(item.AIStyle),
		AIPrompt:               normalizeAIPrompt(item.AIPrompt),
	}
}

// Get 返回当前登录账号的个人功能设置与 AI 设置。
func (ac *AgentSettingController) Get(c *gin.Context) {
	userName, _ := getAuthUser(c)
	if strings.TrimSpace(userName) == "" {
		response.ResponseError(c, http.StatusUnauthorized, response.ErrCodeAuthContextMissing)
		return
	}
	item, err := getOrCreateAgentSetting(userName)
	if err != nil {
		logger.Errorf("agent setting get failed user=%s err=%v", userName, err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeAdminSettingsSaveFailed)
		return
	}
	response.ResponseSuccess(c, toAgentSettingResponse(item))
}

// Update 更新当前登录账号的个人功能设置与 AI 设置。
func (ac *AgentSettingController) Update(c *gin.Context) {
	userName, _ := getAuthUser(c)
	if strings.TrimSpace(userName) == "" {
		response.ResponseError(c, http.StatusUnauthorized, response.ErrCodeAuthContextMissing)
		return
	}

	var req struct {
		SoundEnabled           bool   `json:"soundEnabled"`
		DesktopNotifyEnabled   bool   `json:"desktopNotifyEnabled"`
		TypingIndicatorEnabled bool   `json:"typingIndicatorEnabled"`
		EnterToSend            bool   `json:"enterToSend"`
		AIEnabled              bool   `json:"aiEnabled"`
		AIModel                string `json:"aiModel"`
		AIStyle                string `json:"aiStyle"`
		AIPrompt               string `json:"aiPrompt"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Errorf("agent setting params invalid user=%s err=%v", userName, err)
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeInvalidParams)
		return
	}

	item, err := getOrCreateAgentSetting(userName)
	if err != nil {
		logger.Errorf("agent setting load failed user=%s err=%v", userName, err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeAdminSettingsSaveFailed)
		return
	}

	item.SoundEnabled = req.SoundEnabled
	item.DesktopNotifyEnabled = req.DesktopNotifyEnabled
	item.TypingIndicatorEnabled = req.TypingIndicatorEnabled
	item.EnterToSend = req.EnterToSend
	item.AIEnabled = req.AIEnabled
	item.AIModel = normalizeAIModel(req.AIModel)
	item.AIStyle = normalizeAIStyle(req.AIStyle)
	item.AIPrompt = normalizeAIPrompt(req.AIPrompt)

	if err := store.DB.Save(item).Error; err != nil {
		logger.Errorf("agent setting save failed user=%s err=%v", userName, err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeAdminSettingsSaveFailed)
		return
	}
	response.ResponseSuccess(c, toAgentSettingResponse(item))
}
