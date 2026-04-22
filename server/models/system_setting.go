package models

import "gorm.io/gorm"

// SystemSetting 存储系统级配置（键值对）。
// 当前主要承载管理后台“系统设置”页的持久化内容。
type SystemSetting struct {
	gorm.Model
	Key   string `gorm:"size:64;uniqueIndex;not null" json:"key"`
	Value string `gorm:"type:text;not null" json:"value"`
}
