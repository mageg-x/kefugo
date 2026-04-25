package models

import "time"

type VecTable struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	TableName string    `gorm:"size:64;not null;uniqueIndex" json:"table_name"`
	Dims      int       `gorm:"not null" json:"dims"`
	ConfigID  uint      `gorm:"not null;index" json:"config_id"`
	CreatedAt time.Time `json:"created_at"`
}
