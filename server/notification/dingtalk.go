package notification

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func init() {
	RegisterNotifier(ChannelDing, func() Notifier {
		return &DingTalkNotifier{}
	})
}

type DingTalkNotifier struct{}

func (n *DingTalkNotifier) Type() ChannelType {
	return ChannelDing
}

func (n *DingTalkNotifier) Name() string {
	return "钉钉"
}

type DingTalkConfig struct {
	AppKey    string `json:"appKey"`
	AppSecret string `json:"appSecret"`
	AgentID   string `json:"agentId"`
}

func (n *DingTalkNotifier) ValidateConfig(config map[string]interface{}) error {
	cfg := n.parseConfig(config)
	if cfg.AppKey == "" {
		return fmt.Errorf("AppKey不能为空")
	}
	if cfg.AppSecret == "" {
		return fmt.Errorf("AppSecret不能为空")
	}
	return nil
}

func (n *DingTalkNotifier) parseConfig(config map[string]interface{}) *DingTalkConfig {
	cfg := &DingTalkConfig{}
	if appKey, ok := config["appKey"].(string); ok {
		cfg.AppKey = strings.TrimSpace(appKey)
	}
	if appSecret, ok := config["appSecret"].(string); ok {
		cfg.AppSecret = strings.TrimSpace(appSecret)
	}
	if agentID, ok := config["agentId"].(string); ok {
		cfg.AgentID = strings.TrimSpace(agentID)
	}
	return cfg
}

func (n *DingTalkNotifier) TestConnection(config map[string]interface{}) error {
	cfg := n.parseConfig(config)
	if err := n.ValidateConfig(config); err != nil {
		return err
	}

	accessToken, err := n.getAccessToken(cfg.AppKey, cfg.AppSecret)
	if err != nil {
		return err
	}

	corpName, err := n.getCorpInfo(accessToken)
	if err != nil {
		return err
	}

	return fmt.Errorf("连接成功: %s", corpName)
}

func (n *DingTalkNotifier) Send(ctx context.Context, to string, notification *Notification) error {
	cfg, ok := ctx.Value("dingtalk_config").(*DingTalkConfig)
	if !ok {
		return fmt.Errorf("dingtalk config not found in context")
	}

	accessToken, err := n.getAccessToken(cfg.AppKey, cfg.AppSecret)
	if err != nil {
		return fmt.Errorf("获取access_token失败: %v", err)
	}

	message := map[string]interface{}{
		"agent_id": cfg.AgentID,
		"userid_list": to,
		"msg": map[string]interface{}{
			"msgtype": "text",
			"text": map[string]string{
				"content": fmt.Sprintf("%s\n\n%s", notification.Title, notification.Content),
			},
		},
	}

	messageBytes, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %v", err)
	}

	apiURL := fmt.Sprintf("https://oapi.dingtalk.com/topapi/message/corpconversation/asyncsend_v2?access_token=%s", accessToken)
	resp, err := http.Post(apiURL, "application/json", strings.NewReader(string(messageBytes)))
	if err != nil {
		return fmt.Errorf("发送消息失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %v", err)
	}

	var result struct {
		Errcode int    `json:"errcode"`
		Errmsg  string `json:"errmsg"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("解析响应失败: %v", err)
	}

	if result.Errcode != 0 {
		return fmt.Errorf("钉钉错误: %s (code: %d)", result.Errmsg, result.Errcode)
	}

	return nil
}

func (n *DingTalkNotifier) GetBindURL(config map[string]interface{}, userID uint, callbackURL string) (string, string, error) {
	cfg := n.parseConfig(config)
	if err := n.ValidateConfig(config); err != nil {
		return "", "", err
	}

	state := n.generateState(userID)
	redirectURI := url.QueryEscape(callbackURL)

	bindURL := fmt.Sprintf(
		"https://login.dingtalk.com/oauth2/auth?redirect_uri=%s&client_id=%s&response_type=code&scope=openid&state=%s&prompt=consent",
		redirectURI,
		cfg.AppKey,
		state,
	)

	return bindURL, state, nil
}

func (n *DingTalkNotifier) HandleCallback(ctx context.Context, config map[string]interface{}, code, state string) (string, error) {
	cfg := n.parseConfig(config)

	userID, err := n.getUserIDByCode(cfg.AppKey, cfg.AppSecret, code)
	if err != nil {
		return "", err
	}

	return userID, nil
}

func (n *DingTalkNotifier) getAccessToken(appKey, appSecret string) (string, error) {
	apiURL := "https://api.dingtalk.com/v1.0/oauth2/accessToken"
	
	payload := map[string]string{
		"appKey":    appKey,
		"appSecret": appSecret,
	}
	
	payloadBytes, _ := json.Marshal(payload)
	
	resp, err := http.Post(apiURL, "application/json", strings.NewReader(string(payloadBytes)))
	if err != nil {
		return "", fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %v", err)
	}

	var result struct {
		AccessToken string `json:"accessToken"`
		ExpireIn    int    `json:"expireIn"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("解析响应失败: %v", err)
	}

	return result.AccessToken, nil
}

func (n *DingTalkNotifier) getCorpInfo(accessToken string) (string, error) {
	return "钉钉企业", nil
}

func (n *DingTalkNotifier) getUserIDByCode(appKey, appSecret, code string) (string, error) {
	accessToken, err := n.getAccessToken(appKey, appSecret)
	if err != nil {
		return "", err
	}

	apiURL := fmt.Sprintf("https://api.dingtalk.com/v1.0/contact/users/me?access_token=%s", accessToken)
	req, _ := http.NewRequest("GET", apiURL, nil)
	req.Header.Set("x-acs-dingtalk-access-token", accessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %v", err)
	}

	var result struct {
		OpenID string `json:"openId"`
		UnionID string `json:"unionId"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("解析响应失败: %v", err)
	}

	if result.OpenID != "" {
		return result.OpenID, nil
	}
	return result.UnionID, nil
}

func (n *DingTalkNotifier) generateState(agentID uint) string {
	timestamp := time.Now().Unix()
	randomBytes := make([]byte, 8)
	rand.Read(randomBytes)
	random := hex.EncodeToString(randomBytes)
	return fmt.Sprintf("ding_bind_%d_%d_%s", agentID, timestamp, random)
}

func generateDingTalkSignature(timestamp, secret string) string {
	stringToSign := fmt.Sprintf("%s\n%s", timestamp, secret)
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(stringToSign))
	return hex.EncodeToString(h.Sum(nil))
}
