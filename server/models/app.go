package models

import (
	"encoding/json"
	"fmt"
	"kefu-server/store"
	"kefu-server/utils"
	"kefu-server/utils/logger"
	"strings"

	"gorm.io/gorm"
)

type App struct {
	gorm.Model
	Name        string `gorm:"size:255" json:"name"`
	Logo        string `gorm:"size:255" json:"logo"`
	AppID       string `gorm:"size:255" json:"app_id"`
	Status      int    `gorm:"size:255" json:"status"` // 1=启用, 0=禁用
	AllowDomain string `gorm:"size:255" json:"allow_domain"`
	WelcomeMsg  string `gorm:"size:255" json:"welcome_msg"`
	Contact     string `gorm:"size:255" json:"contact"` // 联系人
}

// GenAppID 生成唯一的 AppID
func GenAppID() string {
	// 生成基于时间戳和随机数的 AppID
	timestamp := utils.GenerateTimestamp()
	random := utils.GenerateRandomString(8)
	return "zerospace_" + timestamp + "_" + random
}

// CreateDefaultApps 在空库初始化默认应用，便于本地部署后直接联调 SDK。
func CreateDefaultApps(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}

	const demoAppID = "demo_kefu_app"
	var existing App
	if err := db.Where("app_id = ?", demoAppID).First(&existing).Error; err == nil {
		return nil
	} else if err != gorm.ErrRecordNotFound {
		logger.Errorf("default app query failed app_id=%s err=%v", demoAppID, err)
		return err
	}

	defaultApp := App{
		Name:        "演示客服应用",
		AppID:       demoAppID,
		Status:      1,
		AllowDomain: "localhost,127.0.0.1",
		WelcomeMsg:  "您好，欢迎咨询，我们会尽快回复您。",
		Contact:     "admin",
	}
	if err := db.Create(&defaultApp).Error; err != nil {
		logger.Errorf("default app create failed err=%v", err)
		return err
	}
	logger.Infof("default app created app_id=%s", defaultApp.AppID)
	return nil
}

func GetApp(appID string) *App {
	var app App
	if err := store.DB.Where("app_id = ? AND status = ?", appID, 1).First(&app).Error; err != nil {
		logger.Errorf("app query failed app_id=%s err=%v", appID, err)
		return nil
	}
	return &app
}

// HasOnlineAgentForApp 判断目标应用是否存在可服务在线客服。
func HasOnlineAgentForApp(appID string) bool {
	if strings.TrimSpace(appID) == "" || store.DB == nil {
		return false
	}
	var users []User
	if err := store.DB.Where("role IN ? AND status = ? AND active = ?", []string{"agent", "admin"}, 1, true).Find(&users).Error; err != nil {
		logger.Errorf("app online agent query failed app_id=%s err=%v", appID, err)
		return false
	}
	target := strings.ToLower(strings.TrimSpace(appID))
	for _, user := range users {
		if strings.TrimSpace(user.Apps) == "" {
			continue
		}
		var apps []string
		if err := json.Unmarshal([]byte(user.Apps), &apps); err == nil {
			for _, app := range apps {
				trimmed := strings.ToLower(strings.TrimSpace(app))
				if trimmed == "all" || trimmed == target {
					return true
				}
			}
			continue
		}
		lower := strings.ToLower(user.Apps)
		if strings.Contains(lower, "\"all\"") || strings.Contains(lower, "\""+target+"\"") {
			return true
		}
	}
	return false
}

// IsDomainAllowed 检查域名是否在允许列表中
func IsDomainAllowed(origin, referer, allowDomain string) bool {
	if allowDomain == "" {
		return false
	}

	// 提取域名
	domain := utils.ExtractDomain(origin)
	if domain == "" {
		domain = utils.ExtractDomain(referer)
	}

	if domain == "" {
		return false
	}

	// 检查域名是否在允许列表中
	allowedDomains := strings.Split(allowDomain, ",")
	for _, allowedDomain := range allowedDomains {
		allowedDomain = strings.TrimSpace(allowedDomain)
		if allowedDomain == domain {
			return true
		}
		// 检查是否为通配符域名（如 *.pages.dev）
		if strings.HasPrefix(allowedDomain, "*") {
			wildcardDomain := strings.TrimPrefix(allowedDomain, "*")
			if strings.HasSuffix(domain, wildcardDomain) {
				return true
			}
		}
	}

	return false
}
