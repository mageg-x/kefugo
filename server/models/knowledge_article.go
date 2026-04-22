package models

import "gorm.io/gorm"

// KnowledgeArticle 表示知识库文章（按应用隔离）。
type KnowledgeArticle struct {
	gorm.Model
	AppID      string `gorm:"size:128;index;not null" json:"app_id"`
	Title      string `gorm:"size:255;not null" json:"title"`
	Content    string `gorm:"type:text;not null" json:"content"`
	Tags       string `gorm:"size:255" json:"tags"`
	Enabled    bool   `gorm:"not null;default:true" json:"enabled"`
	SourceType string `gorm:"size:32;index;default:manual" json:"source_type"`
	SourceName string `gorm:"size:255" json:"source_name"`
}
