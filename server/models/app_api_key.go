package models

import "gorm.io/gorm"

// AppAPIKey 存储应用级 API 密钥。
// 仅管理员可管理，用于服务端鉴权、集成调用与密钥轮换。
type AppAPIKey struct {
	gorm.Model
	AppID   string `gorm:"size:128;index;not null" json:"app_id"`
	Name    string `gorm:"size:128;not null" json:"name"`
	KeyID   string `gorm:"size:64;uniqueIndex;not null" json:"key_id"`
	Secret  string `gorm:"size:255;not null" json:"secret"`
	Enabled bool   `gorm:"not null;default:true" json:"enabled"`
}
