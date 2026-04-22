package models

import "time"

// APIModelConfig 存储外部大模型 API 配置。
type APIModelConfig struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Provider    string    `gorm:"size:32;index;not null" json:"provider"`
	APIKey      string    `gorm:"size:512;not null" json:"api_key"`
	BaseURL     string    `gorm:"size:512" json:"base_url"`
	ModelName   string    `gorm:"size:128;not null" json:"model_name"`
	TimeoutSec  int       `gorm:"not null;default:60" json:"timeout_sec"`
	Temperature float64   `gorm:"not null;default:0.7" json:"temperature"`
	TopP        float64   `gorm:"not null;default:0.9" json:"top_p"`
	MaxTokens   int       `gorm:"not null;default:512" json:"max_tokens"`
	Status      int       `gorm:"not null;default:1;index" json:"status"`
}
