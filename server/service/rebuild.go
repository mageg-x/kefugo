package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"kefu-server/models"
	"kefu-server/service/ai/embedding"
	"kefu-server/store"
	"kefu-server/utils"
	"kefu-server/utils/logger"
)

var (
	rebuildMu     sync.Mutex
	rebuildActive = map[uint]bool{}
)

const (
	rebuildBatchSize         = 50
	rebuildProgressStepCount = 5
)

func updateRebuildTaskProgress(taskID uint, doneCount int, totalCount int, status string) {
	progress := 0
	if totalCount > 0 {
		progress = int(float64(doneCount) / float64(totalCount) * 100)
		if progress > 100 {
			progress = 100
		}
	}
	updates := map[string]interface{}{
		"done_docs":  doneCount,
		"total_docs": totalCount,
		"progress":   progress,
	}
	if strings.TrimSpace(status) != "" {
		updates["status"] = status
	}
	_ = store.DB.Model(&models.RebuildTask{}).Where("id = ?", taskID).Updates(updates).Error
}

func shouldReportRebuildProgress(doneCount, lastReportedDone, totalCount int) bool {
	if doneCount <= 0 {
		return false
	}
	if doneCount >= totalCount {
		return true
	}
	if lastReportedDone < 0 {
		return true
	}
	return doneCount-lastReportedDone >= rebuildProgressStepCount
}

func TriggerRebuild(configID uint) (*models.RebuildTask, error) {
	rebuildMu.Lock()
	if rebuildActive[configID] {
		rebuildMu.Unlock()
		return nil, fmt.Errorf("rebuild already running for config %d", configID)
	}
	rebuildActive[configID] = true
	rebuildMu.Unlock()

	var cfg models.APIModelConfig
	if err := store.DB.First(&cfg, configID).Error; err != nil {
		rebuildMu.Lock()
		delete(rebuildActive, configID)
		rebuildMu.Unlock()
		return nil, fmt.Errorf("config not found: %w", err)
	}
	cfg.APIKey = utils.DecryptAPIKey(cfg.APIKey)
	if cfg.ModelType != string(models.AIModelTypeEmbedding) {
		rebuildMu.Lock()
		delete(rebuildActive, configID)
		rebuildMu.Unlock()
		return nil, fmt.Errorf("only embedding config can trigger rebuild")
	}
	if cfg.Status != 1 {
		rebuildMu.Lock()
		delete(rebuildActive, configID)
		rebuildMu.Unlock()
		return nil, fmt.Errorf("config is not enabled")
	}

	var chunkCount int64
	if err := store.DB.Model(&models.KnowledgeChunk{}).Count(&chunkCount).Error; err != nil {
		rebuildMu.Lock()
		delete(rebuildActive, configID)
		rebuildMu.Unlock()
		return nil, fmt.Errorf("count chunks failed: %w", err)
	}

	task := models.RebuildTask{
		ConfigID:  configID,
		Status:    string(models.RebuildTaskPending),
		TotalDocs: int(chunkCount),
		DoneDocs:  0,
		Progress:  0,
	}
	if err := store.DB.Create(&task).Error; err != nil {
		rebuildMu.Lock()
		delete(rebuildActive, configID)
		rebuildMu.Unlock()
		return nil, fmt.Errorf("create rebuild task failed: %w", err)
	}

	go executeRebuild(task.ID, cfg)

	return &task, nil
}

func executeRebuild(taskID uint, cfg models.APIModelConfig) {
	defer func() {
		rebuildMu.Lock()
		delete(rebuildActive, cfg.ID)
		rebuildMu.Unlock()
	}()

	ctx, cancelCtx := context.WithTimeout(context.Background(), 2*time.Hour)
	defer cancelCtx()

	if err := store.DB.Model(&models.RebuildTask{}).Where("id = ?", taskID).
		Updates(map[string]interface{}{"status": string(models.RebuildTaskRunning)}).Error; err != nil {
		logger.Errorf("rebuild task %d update to running failed: %v", taskID, err)
		return
	}

	provider, err := embedding.NewProvider(&cfg)
	if err != nil {
		failRebuildTask(taskID, fmt.Sprintf("create embedding provider failed: %v", err))
		return
	}

	sqlDB, err := store.DB.DB()
	if err != nil {
		failRebuildTask(taskID, fmt.Sprintf("get sql.DB failed: %v", err))
		return
	}

	vdb := GetVectorStore()
	_ = vdb.Health(ctx)

	vecTableName := cfg.GetVecTableName()
	metaTableName := vecTableName + "_meta"
	if vecTableName == "knowledge_vec_index" {
		metaTableName = "knowledge_vec_index_meta"
	}

	createVecSQL := fmt.Sprintf(`CREATE VIRTUAL TABLE IF NOT EXISTS %s USING vec0(
		vector_id TEXT PRIMARY KEY,
		embedding float[%d]
	)`, vecTableName, cfg.Dims)
	if _, err := sqlDB.ExecContext(ctx, createVecSQL); err != nil {
		failRebuildTask(taskID, fmt.Sprintf("create vec table %s failed: %v", vecTableName, err))
		return
	}

	createMetaSQL := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		vector_id TEXT PRIMARY KEY,
		collection TEXT NOT NULL,
		metadata TEXT,
		content TEXT
	)`, metaTableName)
	if _, err := sqlDB.ExecContext(ctx, createMetaSQL); err != nil {
		failRebuildTask(taskID, fmt.Sprintf("create meta table %s failed: %v", metaTableName, err))
		return
	}

	var totalChunks int64
	store.DB.Model(&models.KnowledgeChunk{}).Count(&totalChunks)
	updateRebuildTaskProgress(taskID, 0, int(totalChunks), string(models.RebuildTaskRunning))

	offset := 0
	doneCount := 0
	lastReportedDone := -1

	for {
		var chunks []models.KnowledgeChunk
		if err := store.DB.Order("id ASC").Offset(offset).Limit(rebuildBatchSize).Find(&chunks).Error; err != nil {
			failRebuildTask(taskID, fmt.Sprintf("query chunks failed: %v", err))
			return
		}
		if len(chunks) == 0 {
			break
		}

		texts := make([]string, len(chunks))
		for i, c := range chunks {
			texts[i] = c.Content
		}

		batchMeta, metaErr := loadRebuildBatchMeta(chunks)
		if metaErr != nil {
			failRebuildTask(taskID, fmt.Sprintf("load rebuild batch meta failed: %v", metaErr))
			return
		}

		vecs, batchErr := provider.GetEmbeddings(ctx, texts)
		if batchErr != nil {
			logger.Warnf("rebuild batch embedding failed at offset %d: %v, retrying one by one", offset, batchErr)
			for _, c := range chunks {
				vec, embErr := provider.GetEmbedding(ctx, c.Content)
				if embErr != nil {
					logger.Warnf("rebuild chunk %d embedding failed: %v", c.ID, embErr)
					doneCount++
					if shouldReportRebuildProgress(doneCount, lastReportedDone, int(totalChunks)) {
						updateRebuildTaskProgress(taskID, doneCount, int(totalChunks), string(models.RebuildTaskRunning))
						lastReportedDone = doneCount
					}
					continue
				}
				if writeErr := writeRebuildVector(ctx, sqlDB, c, vec, vecTableName, metaTableName, batchMeta); writeErr != nil {
					logger.Warnf("rebuild chunk %d write failed: %v", c.ID, writeErr)
				}
				doneCount++
				if shouldReportRebuildProgress(doneCount, lastReportedDone, int(totalChunks)) {
					updateRebuildTaskProgress(taskID, doneCount, int(totalChunks), string(models.RebuildTaskRunning))
					lastReportedDone = doneCount
				}
			}
		} else {
			for i, c := range chunks {
				if i < len(vecs) && len(vecs[i]) > 0 {
					if writeErr := writeRebuildVector(ctx, sqlDB, c, vecs[i], vecTableName, metaTableName, batchMeta); writeErr != nil {
						logger.Warnf("rebuild chunk %d write failed: %v", c.ID, writeErr)
					}
				}
				doneCount++
				if shouldReportRebuildProgress(doneCount, lastReportedDone, int(totalChunks)) {
					updateRebuildTaskProgress(taskID, doneCount, int(totalChunks), string(models.RebuildTaskRunning))
					lastReportedDone = doneCount
				}
			}
		}

		offset += rebuildBatchSize
	}

	now := time.Now()
	store.DB.Model(&models.RebuildTask{}).Where("id = ?", taskID).
		Updates(map[string]interface{}{
			"status":       string(models.RebuildTaskCompleted),
			"progress":     100,
			"done_docs":    doneCount,
			"completed_at": &now,
		})

	var existing models.VecTable
	if store.DB.Where("table_name = ?", vecTableName).First(&existing).Error != nil {
		vt := models.VecTable{
			TableName: vecTableName,
			Dims:      cfg.Dims,
			ConfigID:  cfg.ID,
		}
		_ = store.DB.Where("table_name = ?", vecTableName).FirstOrCreate(&vt).Error
	}

	cleanupOldVecTables(ctx, sqlDB, vecTableName)
}

type rebuildChunkMeta struct {
	Collection string
	Metadata   string
}

func loadRebuildBatchMeta(chunks []models.KnowledgeChunk) (map[string]rebuildChunkMeta, error) {
	vectorIDs := make([]string, 0, len(chunks))
	docIDs := make([]uint, 0, len(chunks))
	for _, chunk := range chunks {
		if strings.TrimSpace(chunk.VectorID) != "" {
			vectorIDs = append(vectorIDs, chunk.VectorID)
		}
		if chunk.DocumentID > 0 {
			docIDs = append(docIDs, chunk.DocumentID)
		}
	}

	metaMap := make(map[string]rebuildChunkMeta, len(chunks))
	if len(vectorIDs) > 0 {
		var entries []models.KnowledgeVectorEntry
		if err := store.DB.Where("vector_id IN ?", vectorIDs).Find(&entries).Error; err != nil {
			return nil, err
		}
		for _, entry := range entries {
			metaMap[entry.VectorID] = rebuildChunkMeta{
				Collection: strings.TrimSpace(entry.Collection),
				Metadata:   normalizeRebuildMetadata(entry.Metadata),
			}
		}
	}

	docMap := make(map[uint]models.KnowledgeDocument, len(docIDs))
	if len(docIDs) > 0 {
		var docs []models.KnowledgeDocument
		if err := store.DB.Where("id IN ?", docIDs).Find(&docs).Error; err != nil {
			return nil, err
		}
		for _, doc := range docs {
			docMap[doc.ID] = doc
		}
	}

	for _, chunk := range chunks {
		current := metaMap[chunk.VectorID]
		doc := docMap[chunk.DocumentID]
		if current.Collection == "" {
			current.Collection = strings.TrimSpace(doc.VectorCollection)
		}
		if current.Metadata == "" {
			current.Metadata = buildRebuildMetadata(chunk, doc.Name)
		}
		metaMap[chunk.VectorID] = current
	}

	return metaMap, nil
}

func normalizeRebuildMetadata(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return ""
	}
	return raw
}

func buildRebuildMetadata(chunk models.KnowledgeChunk, docName string) string {
	payload := map[string]interface{}{
		"app_id":      chunk.AppID,
		"base_id":     chunk.BaseID,
		"document_id": chunk.DocumentID,
		"chunk_seq":   chunk.ChunkSeq,
	}
	if strings.TrimSpace(docName) != "" {
		payload["doc_name"] = strings.TrimSpace(docName)
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "{}"
	}
	return string(raw)
}

func writeRebuildVector(
	ctx context.Context,
	sqlDB *sql.DB,
	chunk models.KnowledgeChunk,
	vec []float32,
	vecTable,
	metaTable string,
	metaMap map[string]rebuildChunkMeta,
) error {
	vecLiteral := float32SliceToSQLiteLiteral(vec)
	meta := metaMap[chunk.VectorID]
	if meta.Collection == "" {
		return fmt.Errorf("missing vector collection for vector_id=%s", chunk.VectorID)
	}
	if meta.Metadata == "" {
		meta.Metadata = "{}"
	}

	if _, err := sqlDB.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s WHERE vector_id = ?", vecTable), chunk.VectorID); err != nil {
		return err
	}
	if _, err := sqlDB.ExecContext(ctx, fmt.Sprintf("INSERT INTO %s(vector_id, embedding) VALUES(?, vec_f32(?))", vecTable), chunk.VectorID, vecLiteral); err != nil {
		return err
	}
	if _, err := sqlDB.ExecContext(ctx, fmt.Sprintf(`
		INSERT INTO %s(vector_id, collection, metadata, content)
		VALUES(?, ?, ?, ?)
		ON CONFLICT(vector_id) DO UPDATE SET
			collection=excluded.collection,
			metadata=excluded.metadata,
			content=excluded.content
	`, metaTable), chunk.VectorID, meta.Collection, meta.Metadata, chunk.Content); err != nil {
		return err
	}

	return nil
}

func failRebuildTask(taskID uint, errMsg string) {
	store.DB.Model(&models.RebuildTask{}).Where("id = ?", taskID).
		Updates(map[string]interface{}{
			"status":        string(models.RebuildTaskFailed),
			"error_message": errMsg,
		})
}

func GetRebuildTaskStatus(taskID uint) (*models.RebuildTask, error) {
	var task models.RebuildTask
	if err := store.DB.First(&task, taskID).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

func GetLatestRebuildTask(configID uint) (*models.RebuildTask, error) {
	var task models.RebuildTask
	if err := store.DB.Where("config_id = ?", configID).Order("created_at DESC").First(&task).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

func IsRebuildRunning(configID uint) bool {
	rebuildMu.Lock()
	defer rebuildMu.Unlock()
	return rebuildActive[configID]
}

func cleanupOldVecTables(ctx context.Context, sqlDB *sql.DB, activeTable string) {
	var vecTables []models.VecTable
	if err := store.DB.Find(&vecTables).Error; err != nil {
		return
	}
	for _, vt := range vecTables {
		if vt.TableName == activeTable || vt.TableName == "knowledge_vec_index" {
			continue
		}
		metaTable := vt.TableName + "_meta"
		if _, err := sqlDB.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s", vt.TableName)); err != nil {
			logger.Warnf("cleanup old vec table %s failed: %v", vt.TableName, err)
		}
		if _, err := sqlDB.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s", metaTable)); err != nil {
			logger.Warnf("cleanup old meta table %s failed: %v", metaTable, err)
		}
		store.DB.Delete(&vt)
		logger.Infof("cleaned up old vec table: %s (dims=%d)", vt.TableName, vt.Dims)
	}
}
