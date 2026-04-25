package models

import (
	"fmt"
	"time"
)

type AIModelType string

const (
	AIModelTypeChat      AIModelType = "chat"
	AIModelTypeEmbedding AIModelType = "embedding"
	AIModelTypeRerank    AIModelType = "rerank"
)

func ValidAIModelType(s string) bool {
	switch AIModelType(s) {
	case AIModelTypeChat, AIModelTypeEmbedding, AIModelTypeRerank:
		return true
	default:
		return false
	}
}

type APIModelConfig struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Name        string    `gorm:"size:128" json:"name"`
	ModelType   string    `gorm:"size:32;index;not null;default:chat" json:"model_type"`
	Provider    string    `gorm:"size:32;index;not null" json:"provider"`
	APIKey      string    `gorm:"size:512;not null" json:"api_key"`
	BaseURL     string    `gorm:"size:512" json:"base_url"`
	ModelName   string    `gorm:"size:128;not null" json:"model_name"`
	Dims        int       `gorm:"default:384" json:"dims"`
	TopK        int       `gorm:"default:20" json:"top_k"`
	TopN        int       `gorm:"default:5" json:"top_n"`
	TimeoutSec  int       `gorm:"not null;default:60" json:"timeout_sec"`
	Temperature float64   `gorm:"not null;default:0.7" json:"temperature"`
	TopP        float64   `gorm:"not null;default:0.9" json:"top_p"`
	MaxTokens   int       `gorm:"not null;default:512" json:"max_tokens"`
	IsDefault   bool      `gorm:"default:false" json:"is_default"`
	Status      int       `gorm:"not null;default:1;index" json:"status"`
}

func (c APIModelConfig) GetVecTableName() string {
	if c.Dims == 384 {
		return "knowledge_vec_index"
	}
	return fmt.Sprintf("knowledge_vec_%d", c.Dims)
}
