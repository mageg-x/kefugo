package models

import "time"

// KnowledgeVectorCollection 表示向量集合元信息。
// 说明：当前默认后端为 sqlite 存储，集合用于逻辑隔离不同知识库。
type KnowledgeVectorCollection struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Name      string    `gorm:"size:255;uniqueIndex;not null" json:"name"`
}

// KnowledgeVectorEntry 表示知识片段向量记录。
// 说明：为了保持单二进制部署，当前实现将向量检索数据持久化到 sqlite。
type KnowledgeVectorEntry struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	Collection string    `gorm:"size:255;index;not null" json:"collection"`
	VectorID   string    `gorm:"size:255;uniqueIndex;not null" json:"vector_id"`
	Content    string    `gorm:"type:text;not null" json:"content"`
	Metadata   string    `gorm:"type:text" json:"metadata"`
}
