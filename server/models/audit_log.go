package models

import "gorm.io/gorm"

// AuditLog 记录关键管理操作的审计信息。
// 该模型用于后台审计查询与导出，支持按操作者、动作、目标等维度过滤。
type AuditLog struct {
	gorm.Model
	Operator     string `gorm:"size:128;index" json:"operator"`
	OperatorRole string `gorm:"size:32;index" json:"operator_role"`
	Action       string `gorm:"size:128;index" json:"action"`
	TargetType   string `gorm:"size:64;index" json:"target_type"`
	TargetID     string `gorm:"size:255;index" json:"target_id"`
	Result       string `gorm:"size:32;index" json:"result"`
	Detail       string `gorm:"type:text" json:"detail"`
}
