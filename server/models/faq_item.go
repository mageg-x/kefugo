package models

import "gorm.io/gorm"

// FAQItem 表示应用内 FAQ 条目。
type FAQItem struct {
	gorm.Model
	AppID    string `gorm:"size:128;index;not null" json:"app_id"`
	Question string `gorm:"size:255;not null" json:"question"`
	Answer   string `gorm:"type:text;not null" json:"answer"`
	Category string `gorm:"size:64;index" json:"category"`
	Enabled  bool   `gorm:"not null;default:true" json:"enabled"`
}
