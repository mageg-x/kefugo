package models

import "time"

// KnowledgeBase 表示一个知识库空间（按 app 隔离）。
type KnowledgeBase struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	AppID       string    `gorm:"size:128;index;not null" json:"app_id"`
	Name        string    `gorm:"size:128;not null" json:"name"`
	Description string    `gorm:"size:512" json:"description"`
	Collection  string    `gorm:"size:255;index;not null" json:"collection"`
}

// KnowledgeDocument 表示知识库中的源文档。
type KnowledgeDocument struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	BaseID          uint      `gorm:"index;not null" json:"base_id"`
	AppID           string    `gorm:"size:128;index;not null" json:"app_id"`
	Name            string    `gorm:"size:255;not null" json:"name"`
	FileName        string    `gorm:"size:255" json:"file_name"`
	FileSize        int64     `gorm:"not null;default:0" json:"file_size"`
	FileType        string    `gorm:"size:32" json:"file_type"`
	FileURL         string    `gorm:"size:512" json:"file_url"`
	Status          string    `gorm:"size:32;index;not null;default:pending" json:"status"`
	ErrorMessage    string    `gorm:"size:512" json:"error_message"`
	ChunkCount      int       `gorm:"not null;default:0" json:"chunk_count"`
	RawContent      string    `gorm:"type:text" json:"-"`
	LastIndexedAt   *time.Time `json:"last_indexed_at"`
	VectorCollection string   `gorm:"size:255;index;not null" json:"vector_collection"`
}

// KnowledgeChunk 表示可编辑的知识片段。
type KnowledgeChunk struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	BaseID       uint      `gorm:"index;not null" json:"base_id"`
	DocumentID   uint      `gorm:"index;not null" json:"document_id"`
	AppID        string    `gorm:"size:128;index;not null" json:"app_id"`
	VectorID     string    `gorm:"size:255;uniqueIndex;not null" json:"vector_id"`
	ChunkSeq     int       `gorm:"index;not null;default:0" json:"chunk_seq"`
	Content      string    `gorm:"type:text;not null" json:"content"`
	ContentChars int       `gorm:"not null;default:0" json:"content_chars"`
	AvgScore     float64   `gorm:"not null;default:0" json:"avg_score"`
	HitCount     int       `gorm:"not null;default:0" json:"hit_count"`
}

// KnowledgeQAFeedback 记录问答反馈，便于后续优化。
type KnowledgeQAFeedback struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	BaseID     uint      `gorm:"index;not null" json:"base_id"`
	AppID      string    `gorm:"size:128;index;not null" json:"app_id"`
	Question   string    `gorm:"type:text;not null" json:"question"`
	Answer     string    `gorm:"type:text;not null" json:"answer"`
	Helpful    bool      `gorm:"index;not null;default:false" json:"helpful"`
	SourceDocs string    `gorm:"type:text" json:"source_docs"`
}
