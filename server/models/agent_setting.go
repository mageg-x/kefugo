package models

import "gorm.io/gorm"

// AgentSetting 存储客服坐席个人设置（包含功能设置与 AI 设置）。
// 该表按 user_name 唯一索引，保证一个账号只对应一份可读写配置。
type AgentSetting struct {
	gorm.Model
	UserName string `gorm:"size:64;uniqueIndex;not null" json:"user_name"`

	// 功能设置
	SoundEnabled           bool `gorm:"not null;default:true" json:"sound_enabled"`
	DesktopNotifyEnabled   bool `gorm:"not null;default:false" json:"desktop_notify_enabled"`
	TypingIndicatorEnabled bool `gorm:"not null;default:true" json:"typing_indicator_enabled"`
	EnterToSend            bool `gorm:"not null;default:true" json:"enter_to_send"`

	// AI 设置
	AIEnabled bool   `gorm:"not null;default:false" json:"ai_enabled"`
	AIModel   string `gorm:"size:64;not null;default:gpt-4o-mini" json:"ai_model"`
	AIStyle   string `gorm:"size:32;not null;default:professional" json:"ai_style"`
	AIPrompt  string `gorm:"type:text" json:"ai_prompt"`
}
