package controllers

import (
	"encoding/json"
	"strings"
	"sync"
	"time"

	"kefu-server/models"
	"kefu-server/store"
)

type systemSettings struct {
	SystemName       string `json:"systemName"`
	Logo             string `json:"logo"`
	WelcomeMsg       string `json:"welcomeMsg"`
	MaxSessions      int    `json:"maxSessions"`
	Timeout          int    `json:"timeout"`
	AutoAssign       bool   `json:"autoAssign"`
	EmailNotify      bool   `json:"emailNotify"`
	NotifyEmail      string `json:"notifyEmail"`
	SMTPHost         string `json:"smtpHost"`
	SMTPPort         int    `json:"smtpPort"`
	SMTPUser         string `json:"smtpUser"`
	SMTPPassword     string `json:"smtpPassword"`
	SMTPFrom         string `json:"smtpFrom"`
	SessionEncrypt   bool   `json:"sessionEncrypt"`
	IPLimit          bool   `json:"ipLimit"`
	IPWhitelist      string `json:"ipWhitelist"`
	Captcha          bool   `json:"captcha"`
	SensitiveWords   string `json:"sensitiveWords"`
	RateLimitEnabled bool   `json:"rateLimitEnabled"`
	RateLimitRPM     int    `json:"rateLimitRpm"`
	RateLimitBurst   int    `json:"rateLimitBurst"`
	OfflineNotifyURL string `json:"offlineNotifyUrl"`
	WecomCorpID      string `json:"wecomCorpId"`
	WecomAgentID     int    `json:"wecomAgentId"`
	WecomSecret      string `json:"wecomSecret"`
	AIBotEnabled     bool   `json:"aiBotEnabled"`
	AIBotName        string `json:"aiBotName"`
	AIBotModel       string `json:"aiBotModel"`
	AIBotStyle       string `json:"aiBotStyle"`
	AIBotPrompt      string `json:"aiBotPrompt"`
	AIBotTopK        int    `json:"aiBotTopK"`
	AIBotWhenAssigned bool  `json:"aiBotWhenAssigned"`
}

// defaultSystemSettings 返回系统设置默认值。
func defaultSystemSettings() systemSettings {
	return systemSettings{
		SystemName:       "零点客服系统",
		Logo:             "",
		WelcomeMsg:       "您好，有什么可以帮您的吗？",
		MaxSessions:      5,
		Timeout:          180,
		AutoAssign:       true,
		EmailNotify:      false,
		NotifyEmail:      "",
		SMTPHost:         "",
		SMTPPort:         25,
		SMTPUser:         "",
		SMTPPassword:     "",
		SMTPFrom:         "kefu@localhost",
		SessionEncrypt:   true,
		IPLimit:          false,
		IPWhitelist:      "",
		Captcha:          false,
		SensitiveWords:   "",
		RateLimitEnabled: true,
		RateLimitRPM:     120,
		RateLimitBurst:   60,
		OfflineNotifyURL: "",
		AIBotEnabled:     false,
		AIBotName:        "AI机器人",
		AIBotModel:       "",
		AIBotStyle:       defaultAIStyle,
		AIBotPrompt:      "你是一名客服机器人，回答要准确、简洁、礼貌。证据不足时明确说明并建议转人工。",
		AIBotTopK:        5,
		AIBotWhenAssigned: false,
	}
}

var systemSettingsCache = struct {
	mu      sync.RWMutex
	expires time.Time
	value   systemSettings
}{
	value: defaultSystemSettings(),
}

// normalizeSystemSettings 对配置做兜底归一，避免异常值影响运行时逻辑。
func normalizeSystemSettings(cfg systemSettings) systemSettings {
	if cfg.MaxSessions <= 0 {
		cfg.MaxSessions = 5
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 180
	}
	if cfg.RateLimitRPM <= 0 {
		cfg.RateLimitRPM = 120
	}
	if cfg.RateLimitBurst <= 0 {
		cfg.RateLimitBurst = 60
	}
	cfg.AIBotName = strings.TrimSpace(cfg.AIBotName)
	if cfg.AIBotName == "" {
		cfg.AIBotName = "AI机器人"
	}
	if len([]rune(cfg.AIBotName)) > 64 {
		cfg.AIBotName = string([]rune(cfg.AIBotName)[:64])
	}
	cfg.AIBotModel = strings.TrimSpace(cfg.AIBotModel)
	if len([]rune(cfg.AIBotModel)) > 128 {
		cfg.AIBotModel = string([]rune(cfg.AIBotModel)[:128])
	}
	cfg.AIBotStyle = normalizeAIStyle(cfg.AIBotStyle)
	cfg.AIBotPrompt = strings.TrimSpace(cfg.AIBotPrompt)
	if cfg.AIBotPrompt == "" {
		cfg.AIBotPrompt = "你是一名客服机器人，回答要准确、简洁、礼貌。证据不足时明确说明并建议转人工。"
	}
	if len([]rune(cfg.AIBotPrompt)) > 4000 {
		cfg.AIBotPrompt = string([]rune(cfg.AIBotPrompt)[:4000])
	}
	if cfg.AIBotTopK <= 0 {
		cfg.AIBotTopK = 5
	}
	if cfg.AIBotTopK > 20 {
		cfg.AIBotTopK = 20
	}
	return cfg
}

// loadSystemSettingsFromDB 从数据库加载设置，不可用时回退默认值。
func loadSystemSettingsFromDB() systemSettings {
	cfg := defaultSystemSettings()
	if store.DB == nil {
		return cfg
	}

	var row models.SystemSetting
	if err := store.DB.Where("key = ?", "system_settings").First(&row).Error; err != nil {
		return cfg
	}
	if err := json.Unmarshal([]byte(row.Value), &cfg); err != nil {
		return defaultSystemSettings()
	}
	return normalizeSystemSettings(cfg)
}

// getSystemSettingsCached 返回带短 TTL 的系统设置缓存，降低数据库读取压力。
func getSystemSettingsCached() systemSettings {
	now := time.Now()
	systemSettingsCache.mu.RLock()
	if now.Before(systemSettingsCache.expires) {
		cfg := systemSettingsCache.value
		systemSettingsCache.mu.RUnlock()
		return cfg
	}
	systemSettingsCache.mu.RUnlock()

	cfg := loadSystemSettingsFromDB()
	systemSettingsCache.mu.Lock()
	systemSettingsCache.value = cfg
	systemSettingsCache.expires = now.Add(2 * time.Second)
	systemSettingsCache.mu.Unlock()
	return cfg
}

// setSystemSettingsCache 主动刷新系统设置缓存。
func setSystemSettingsCache(cfg systemSettings) {
	cfg = normalizeSystemSettings(cfg)
	systemSettingsCache.mu.Lock()
	systemSettingsCache.value = cfg
	// 主动写缓存时直接立即生效，避免配置变更后的短暂读取延迟。
	systemSettingsCache.expires = time.Now().Add(30 * time.Minute)
	systemSettingsCache.mu.Unlock()
}

// parseSensitiveWords 解析敏感词配置文本，支持多种分隔符并自动去重。
func parseSensitiveWords(raw string) []string {
	parts := strings.FieldsFunc(strings.TrimSpace(raw), func(r rune) bool {
		return r == ',' || r == '\n' || r == '\r' || r == ';' || r == '，' || r == '；'
	})
	result := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, p := range parts {
		word := strings.TrimSpace(p)
		if word == "" {
			continue
		}
		if _, ok := seen[word]; ok {
			continue
		}
		seen[word] = struct{}{}
		result = append(result, word)
	}
	return result
}

// filterSensitiveText 按当前敏感词配置对文本做替换脱敏。
func filterSensitiveText(text string) string {
	raw := strings.TrimSpace(text)
	if raw == "" {
		return text
	}
	cfg := getSystemSettingsCached()
	words := parseSensitiveWords(cfg.SensitiveWords)
	if len(words) == 0 {
		return text
	}
	replaced := text
	for _, word := range words {
		replaced = strings.ReplaceAll(replaced, word, "***")
	}
	return replaced
}

// GetSystemSettingsForMiddleware 为中间件提供只读系统配置快照。
func GetSystemSettingsForMiddleware() systemSettings {
	return getSystemSettingsCached()
}
