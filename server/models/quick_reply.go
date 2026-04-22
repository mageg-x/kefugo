package models

import "gorm.io/gorm"

type QuickReply struct {
	gorm.Model
	Owner      string `gorm:"size:128;index" json:"owner"` // agent username
	Title      string `gorm:"size:255;not null" json:"title"`
	Category   string `gorm:"size:64;not null" json:"category"`
	Content    string `gorm:"type:text;not null" json:"content"`
	UsageCount int64  `gorm:"default:0" json:"usage_count"`
}
