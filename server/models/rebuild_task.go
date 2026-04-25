package models

import "time"

type RebuildTaskStatus string

const (
	RebuildTaskPending   RebuildTaskStatus = "pending"
	RebuildTaskRunning   RebuildTaskStatus = "running"
	RebuildTaskCompleted RebuildTaskStatus = "completed"
	RebuildTaskFailed    RebuildTaskStatus = "failed"
)

type RebuildTask struct {
	ID           uint              `gorm:"primaryKey" json:"id"`
	ConfigID     uint              `gorm:"not null;index" json:"config_id"`
	Status       string            `gorm:"size:32;not null;default:pending" json:"status"`
	Progress     int               `gorm:"default:0" json:"progress"`
	TotalDocs    int               `gorm:"default:0" json:"total_docs"`
	DoneDocs     int               `gorm:"default:0" json:"done_docs"`
	ErrorMessage string            `gorm:"type:text" json:"error_message"`
	CreatedAt    time.Time         `json:"created_at"`
	CompletedAt  *time.Time        `json:"completed_at"`
	Config       APIModelConfig    `gorm:"foreignKey:ConfigID" json:"-"`
}
