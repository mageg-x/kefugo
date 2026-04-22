package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"kefu-server/models"
	"kefu-server/store"
)

// RebuildChunksFromRawContent 按文档 raw 内容重新切片入库（用于编辑/重建）。
// 说明：通过 VectorStore 抽象写入向量后端，默认实现为 sqlite 内置存储。
func RebuildChunksFromRawContent(ctx context.Context, vc VectorStore, doc *models.KnowledgeDocument) (int, error) {
	if vc == nil || doc == nil {
		return 0, fmt.Errorf("invalid rebuild input")
	}
	if strings.TrimSpace(doc.RawContent) == "" {
		return 0, nil
	}

	var existed []models.KnowledgeChunk
	if err := store.DB.Where("document_id = ?", doc.ID).Find(&existed).Error; err != nil {
		return 0, fmt.Errorf("load existing chunks failed: %w", err)
	}
	for _, chunk := range existed {
		_ = vc.DeleteVector(ctx, doc.VectorCollection, chunk.VectorID)
	}
	if err := store.DB.Unscoped().Where("document_id = ?", doc.ID).Delete(&models.KnowledgeChunk{}).Error; err != nil {
		return 0, fmt.Errorf("delete old chunks failed: %w", err)
	}

	parts := splitTextForKnowledge(strings.TrimSpace(doc.RawContent), 450, 80)
	now := time.Now()
	created := 0
	for idx, part := range parts {
		vectorID := fmt.Sprintf("kb_%d_doc_%d_chunk_%d_%d", doc.BaseID, doc.ID, idx+1, now.UnixNano())
		meta := map[string]interface{}{
			"app_id":      doc.AppID,
			"base_id":     doc.BaseID,
			"document_id": doc.ID,
			"chunk_seq":   idx + 1,
			"doc_name":    doc.Name,
		}
		if err := vc.InsertText(ctx, doc.VectorCollection, vectorID, part, meta); err != nil {
			return created, err
		}
		row := models.KnowledgeChunk{
			BaseID:       doc.BaseID,
			DocumentID:   doc.ID,
			AppID:        doc.AppID,
			VectorID:     vectorID,
			ChunkSeq:     idx + 1,
			Content:      part,
			ContentChars: len([]rune(part)),
		}
		if err := store.DB.Create(&row).Error; err != nil {
			return created, fmt.Errorf("save chunk row failed: %w", err)
		}
		created++
	}

	doc.ChunkCount = created
	doc.Status = "indexed"
	doc.ErrorMessage = ""
	doc.LastIndexedAt = &now
	if err := store.DB.Save(doc).Error; err != nil {
		return created, fmt.Errorf("update document index result failed: %w", err)
	}
	return created, nil
}

func splitTextForKnowledge(text string, chunkSize, overlap int) []string {
	clean := strings.TrimSpace(text)
	if clean == "" {
		return nil
	}
	if chunkSize <= 0 {
		chunkSize = 450
	}
	if overlap < 0 {
		overlap = 0
	}
	runes := []rune(clean)
	if len(runes) <= chunkSize {
		return []string{clean}
	}
	step := chunkSize - overlap
	if step <= 0 {
		step = chunkSize
	}
	result := make([]string, 0, len(runes)/step+1)
	for i := 0; i < len(runes); i += step {
		end := i + chunkSize
		if end > len(runes) {
			end = len(runes)
		}
		part := strings.TrimSpace(string(runes[i:end]))
		if part != "" {
			result = append(result, part)
		}
		if end >= len(runes) {
			break
		}
	}
	return result
}
