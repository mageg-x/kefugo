package notification

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/xen0n/go-workwx"
)

func init() {
	RegisterNotifier(ChannelWecom, func() Notifier {
		return &WecomNotifier{}
	})
}

type WecomNotifier struct{}

func (n *WecomNotifier) Type() ChannelType {
	return ChannelWecom
}

func (n *WecomNotifier) Name() string {
	return "企业微信"
}

type WecomConfig struct {
	CorpID  string `json:"corpId"`
	AgentID int    `json:"agentId"`
	Secret  string `json:"secret"`
}

func (n *WecomNotifier) ValidateConfig(config map[string]interface{}) error {
	cfg := n.parseConfig(config)
	if cfg.CorpID == "" {
		return fmt.Errorf("企业ID不能为空")
	}
	if cfg.AgentID == 0 {
		return fmt.Errorf("应用ID不能为空")
	}
	if cfg.Secret == "" {
		return fmt.Errorf("应用密钥不能为空")
	}
	return nil
}

func (n *WecomNotifier) parseConfig(config map[string]interface{}) *WecomConfig {
	cfg := &WecomConfig{}
	if corpID, ok := config["corpId"].(string); ok {
		cfg.CorpID = strings.TrimSpace(corpID)
	}
	if agentID, ok := config["agentId"].(float64); ok {
		cfg.AgentID = int(agentID)
	}
	if secret, ok := config["secret"].(string); ok {
		cfg.Secret = strings.TrimSpace(secret)
	}
	return cfg
}

func (n *WecomNotifier) TestConnection(config map[string]interface{}) error {
	cfg := n.parseConfig(config)
	if err := n.ValidateConfig(config); err != nil {
		return err
	}

	app := workwx.New(cfg.CorpID)
	appApp := app.WithApp(cfg.Secret, int64(cfg.AgentID))

	_, err := appApp.ListDepts(1)
	if err != nil {
		return fmt.Errorf("连接测试失败: %v", err)
	}

	return nil
}

func (n *WecomNotifier) Send(ctx context.Context, to string, notification *Notification) error {
	cfg, ok := ctx.Value("wecom_config").(*WecomConfig)
	if !ok {
		return fmt.Errorf("wecom config not found in context")
	}

	app := workwx.New(cfg.CorpID)
	appApp := app.WithApp(cfg.Secret, int64(cfg.AgentID))

	recipient := &workwx.Recipient{
		UserIDs: []string{to},
	}

	msgContent := fmt.Sprintf("%s\n\n%s", notification.Title, notification.Content)
	err := appApp.SendTextMessage(recipient, msgContent, false)
	if err != nil {
		return fmt.Errorf("发送消息失败: %v", err)
	}

	return nil
}

func (n *WecomNotifier) GetBindURL(config map[string]interface{}, userID uint, callbackURL string) (string, string, error) {
	cfg := n.parseConfig(config)
	if err := n.ValidateConfig(config); err != nil {
		return "", "", err
	}

	state := n.generateState(userID)

	bindURL := fmt.Sprintf(
		"https://open.work.weixin.qq.com/wwopen/sso/qrConnect?appid=%s&agentid=%d&redirect_uri=%s&state=%s",
		cfg.CorpID,
		cfg.AgentID,
		fmt.Sprintf("%%3A%%2F%%2F%s", callbackURL),
		state,
	)

	return bindURL, state, nil
}

func (n *WecomNotifier) HandleCallback(ctx context.Context, config map[string]interface{}, code, state string) (string, error) {
	cfg := n.parseConfig(config)

	app := workwx.New(cfg.CorpID)
	appApp := app.WithApp(cfg.Secret, int64(cfg.AgentID))

	userInfo, err := appApp.GetUserInfoByCode(code)
	if err != nil {
		return "", fmt.Errorf("获取用户信息失败: %v", err)
	}

	if userInfo.UserID != "" {
		return userInfo.UserID, nil
	}
	return userInfo.OpenID, nil
}

func (n *WecomNotifier) generateState(agentID uint) string {
	timestamp := time.Now().Unix()
	randomBytes := make([]byte, 8)
	rand.Read(randomBytes)
	random := hex.EncodeToString(randomBytes)
	return fmt.Sprintf("bind_%d_%d_%s", agentID, timestamp, random)
}

func SendWecomMessageWithContext(ctx context.Context, corpID, secret string, agentID int, wecomUserID, title, content string) error {
	cfg := &WecomConfig{
		CorpID:  corpID,
		AgentID: agentID,
		Secret:  secret,
	}

	notifier := &WecomNotifier{}
	childCtx := context.WithValue(ctx, "wecom_config", cfg)

	return notifier.Send(childCtx, wecomUserID, &Notification{
		Title:   title,
		Content: content,
	})
}
