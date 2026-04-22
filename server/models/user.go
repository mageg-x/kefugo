package models

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"

	"kefu-server/utils"
	"kefu-server/utils/logger"
)

type User struct {
	gorm.Model
	Username        string     `gorm:"uniqueIndex;size:50;not null" json:"username"`
	Password        string     `gorm:"size:255;not null" json:"-"`
	Avatar          string     `gorm:"size:255" json:"avatar"`
	Email           string     `gorm:"size:128" json:"email"`
	Phone           string     `gorm:"size:32" json:"phone"`
	Bio             string     `gorm:"size:255" json:"bio"`
	Role            string     `gorm:"size:50;not null" json:"role"`
	Status          int        `gorm:"size:50;not null" json:"status"` // 1、在席 0、离席
	Active          bool       `gorm:"default:true" json:"active"`
	Apps            string     `gorm:"type:text" json:"apps"`
	WecomUserID     string     `gorm:"size:64" json:"wecom_userid"`
	WecomBindStatus int        `gorm:"default:0" json:"wecom_bind_status"`
	WecomBindTime   *time.Time `json:"wecom_bind_time"`
}

// CreateDefaultUsers 在空库初始化默认管理员与客服账号。
func CreateDefaultUsers(db *gorm.DB) error {
	var count int64
	db.Model(&User{}).Count(&count)
	apps, _ := json.Marshal([]string{"all"})
	if count == 0 {
		defaultAdminPassword := "12345678"
		defaultAgentPassword := "12345678"

		adminPassword, err := utils.HashPassword(defaultAdminPassword)
		if err != nil {
			logger.Errorf("default admin password hash failed: %v", err)
			return err
		}
		agentPassword, err := utils.HashPassword(defaultAgentPassword)
		if err != nil {
			logger.Errorf("default agent password hash failed: %v", err)
			return err
		}

		users := []User{
			{
				Username: "admin",
				Password: adminPassword,
				Role:     "admin",
				Avatar:   "https://api.dicebear.com/7.x/avataaars/svg?seed=admin",
				Status:   1,
				Active:   true,
				Apps:     string(apps),
			},
			{
				Username: "客服-小花",
				Password: agentPassword,
				Role:     "agent",
				Avatar:   "https://api.dicebear.com/7.x/avataaars/svg?seed=agent",
				Status:   1,
				Active:   true,
				Apps:     string(apps),
			},
		}

		if err := db.Create(&users).Error; err != nil {
			logger.Errorf("default users create failed: %v", err)
			return err
		}

		logger.Infof("default users created successfully")
		logger.Warnf("default admin password: %s", defaultAdminPassword)
		logger.Warnf("default agent password: %s", defaultAgentPassword)
	}

	return nil
}
