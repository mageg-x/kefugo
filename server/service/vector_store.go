package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
	"unicode"

	"kefu-server/models"
	"kefu-server/store"
	"kefu-server/utils/logger"
)

// VectorHit 为向量检索统一结果结构。
type VectorHit struct {
	ID       string                 `json:"id"`
	Score    float64                `json:"score"`
	Metadata map[string]interface{} `json:"metadata"`
	Content  string                 `json:"content"`
}

// VectorStore 定义可替换向量存储胶水层接口。
type VectorStore interface {
	Health(ctx context.Context) error
	EnsureCollection(ctx context.Context, name string) error
	DeleteCollection(ctx context.Context, name string) error
	InsertText(ctx context.Context, collection, vectorID, text string, metadata map[string]interface{}) error
	DeleteVector(ctx context.Context, collection, vectorID string) error
	SearchText(ctx context.Context, collection, query string, topK int) ([]VectorHit, error)
}

var (
	vectorStoreOnce sync.Once
	vectorStoreInst VectorStore
)

// GetVectorStore 返回全局向量存储实例。
func GetVectorStore() VectorStore {
	vectorStoreOnce.Do(func() {
		vectorStoreInst = NewSQLiteVecStore()
	})
	return vectorStoreInst
}

// SQLiteVecStore 使用 modernc sqlite + sqlite-vec 扩展实现向量检索。
type SQLiteVecStore struct {
	once         sync.Once
	ready        bool
	lastInitErr  error
}

func NewSQLiteVecStore() *SQLiteVecStore {
	return &SQLiteVecStore{}
}

func (s *SQLiteVecStore) initSchema(ctx context.Context) error {
	if store.DB == nil {
		return fmt.Errorf("database not initialized")
	}
	sqlDB, err := store.DB.DB()
	if err != nil {
		return err
	}

	// 元数据表仍沿用 gorm 管理，便于回溯与调试。
	if err := store.DB.WithContext(ctx).AutoMigrate(&models.KnowledgeVectorCollection{}, &models.KnowledgeVectorEntry{}); err != nil {
		return err
	}

	stmts := []string{
		`CREATE TABLE IF NOT EXISTS knowledge_vec_index_meta (
			vector_id TEXT PRIMARY KEY,
			collection TEXT NOT NULL,
			metadata TEXT,
			content TEXT
		)`,
		`CREATE INDEX IF NOT EXISTS idx_knowledge_vec_index_meta_collection ON knowledge_vec_index_meta(collection)`,
	}
	for _, stmt := range stmts {
		if _, err := sqlDB.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}

	// 强制启用 vec0；若不可用则直接失败，让上层明确感知“向量服务未就绪”。
	if _, err := sqlDB.ExecContext(ctx, `CREATE VIRTUAL TABLE IF NOT EXISTS knowledge_vec_index USING vec0(
		vector_id TEXT PRIMARY KEY,
		embedding float[384]
	)`); err != nil {
		return fmt.Errorf("create vec0 table failed: %w", err)
	}

	var vecVersion string
	if err := sqlDB.QueryRowContext(ctx, `SELECT vec_version()`).Scan(&vecVersion); err != nil {
		return fmt.Errorf("vec0 extension unavailable: %w", err)
	}
	if strings.TrimSpace(vecVersion) == "" {
		return fmt.Errorf("vec0 extension unavailable: empty version")
	}
	return nil
}

func (s *SQLiteVecStore) ensureReady(ctx context.Context) error {
	s.once.Do(func() {
		if err := s.initSchema(ctx); err != nil {
			s.ready = false
			s.lastInitErr = err
			logger.Errorf("vector store init failed err=%v", err)
			return
		}
		s.ready = true
	})
	if s.ready {
		return nil
	}
	if s.lastInitErr != nil {
		return s.lastInitErr
	}
	return fmt.Errorf("vector store not ready")
}

func (s *SQLiteVecStore) Health(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("context is nil")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := s.ensureReady(ctx); err != nil {
		return err
	}
	return nil
}

func (s *SQLiteVecStore) EnsureCollection(ctx context.Context, name string) error {
	if err := s.Health(ctx); err != nil {
		return err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("collection is empty")
	}
	row := models.KnowledgeVectorCollection{Name: name}
	return store.DB.WithContext(ctx).Where("name = ?", name).FirstOrCreate(&row).Error
}

func (s *SQLiteVecStore) DeleteCollection(ctx context.Context, name string) error {
	if err := s.Health(ctx); err != nil {
		return err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	sqlDB, err := store.DB.DB()
	if err != nil {
		return err
	}

	rows, err := sqlDB.QueryContext(ctx, `SELECT vector_id FROM knowledge_vec_index_meta WHERE collection = ?`, name)
	if err != nil {
		return err
	}
	defer rows.Close()
	ids := make([]string, 0, 128)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, strings.TrimSpace(id))
		}
	}

	for _, id := range ids {
		if _, err := sqlDB.ExecContext(ctx, `DELETE FROM knowledge_vec_index WHERE vector_id = ?`, id); err != nil {
			return err
		}
	}
	if _, err := sqlDB.ExecContext(ctx, `DELETE FROM knowledge_vec_index_meta WHERE collection = ?`, name); err != nil {
		return err
	}

	tx := store.DB.WithContext(ctx)
	if err := tx.Where("collection = ?", name).Delete(&models.KnowledgeVectorEntry{}).Error; err != nil {
		return err
	}
	return tx.Where("name = ?", name).Delete(&models.KnowledgeVectorCollection{}).Error
}

func (s *SQLiteVecStore) InsertText(ctx context.Context, collection, vectorID, text string, metadata map[string]interface{}) error {
	if err := s.Health(ctx); err != nil {
		return err
	}
	collection = strings.TrimSpace(collection)
	vectorID = strings.TrimSpace(vectorID)
	text = strings.TrimSpace(text)
	if collection == "" || vectorID == "" || text == "" {
		return fmt.Errorf("invalid vector input")
	}
	if err := s.EnsureCollection(ctx, collection); err != nil {
		return err
	}

	metaRaw := "{}"
	if metadata != nil {
		if b, err := json.Marshal(metadata); err == nil {
			metaRaw = string(b)
		}
	}

	feature := textToFeatureVec(text)
	vecLiteral := featureVecToSQLiteLiteral(feature, 384)

	sqlDB, err := store.DB.DB()
	if err != nil {
		return err
	}
	if _, err := sqlDB.ExecContext(ctx, `DELETE FROM knowledge_vec_index WHERE vector_id = ?`, vectorID); err != nil {
		return err
	}
	if _, err := sqlDB.ExecContext(ctx, `INSERT INTO knowledge_vec_index(vector_id, embedding) VALUES(?, vec_f32(?))`, vectorID, vecLiteral); err != nil {
		return err
	}
	if _, err := sqlDB.ExecContext(ctx, `
		INSERT INTO knowledge_vec_index_meta(vector_id, collection, metadata, content)
		VALUES(?, ?, ?, ?)
		ON CONFLICT(vector_id) DO UPDATE SET
			collection=excluded.collection,
			metadata=excluded.metadata,
			content=excluded.content
	`, vectorID, collection, metaRaw, text); err != nil {
		return err
	}

	row := models.KnowledgeVectorEntry{
		Collection: collection,
		VectorID:   vectorID,
		Content:    text,
		Metadata:   metaRaw,
	}
	return store.DB.WithContext(ctx).Where("vector_id = ?", vectorID).Assign(row).FirstOrCreate(&row).Error
}

func (s *SQLiteVecStore) DeleteVector(ctx context.Context, collection, vectorID string) error {
	if err := s.Health(ctx); err != nil {
		return err
	}
	vectorID = strings.TrimSpace(vectorID)
	if vectorID == "" {
		return nil
	}
	sqlDB, err := store.DB.DB()
	if err != nil {
		return err
	}
	if _, err := sqlDB.ExecContext(ctx, `DELETE FROM knowledge_vec_index WHERE vector_id = ?`, vectorID); err != nil {
		return err
	}
	if strings.TrimSpace(collection) == "" {
		if _, err := sqlDB.ExecContext(ctx, `DELETE FROM knowledge_vec_index_meta WHERE vector_id = ?`, vectorID); err != nil {
			return err
		}
		return store.DB.WithContext(ctx).Where("vector_id = ?", vectorID).Delete(&models.KnowledgeVectorEntry{}).Error
	}
	if _, err := sqlDB.ExecContext(ctx, `DELETE FROM knowledge_vec_index_meta WHERE collection = ? AND vector_id = ?`, strings.TrimSpace(collection), vectorID); err != nil {
		return err
	}
	return store.DB.WithContext(ctx).Where("collection = ? AND vector_id = ?", strings.TrimSpace(collection), vectorID).Delete(&models.KnowledgeVectorEntry{}).Error
}

func (s *SQLiteVecStore) SearchText(ctx context.Context, collection, query string, topK int) ([]VectorHit, error) {
	if err := s.Health(ctx); err != nil {
		return nil, err
	}
	collection = strings.TrimSpace(collection)
	query = strings.TrimSpace(query)
	if collection == "" || query == "" {
		return []VectorHit{}, nil
	}
	if topK <= 0 {
		topK = 5
	}
	if topK > 20 {
		topK = 20
	}

	return s.searchBySQLiteVec(ctx, collection, query, topK)
}

func (s *SQLiteVecStore) searchBySQLiteVec(ctx context.Context, collection, query string, topK int) ([]VectorHit, error) {
	sqlDB, err := store.DB.DB()
	if err != nil {
		return nil, err
	}
	feature := textToFeatureVec(query)
	vecLiteral := featureVecToSQLiteLiteral(feature, 384)

	rows, err := sqlDB.QueryContext(ctx, `
		SELECT m.vector_id, m.metadata, m.content, vec_distance_l2(v.embedding, vec_f32(?)) AS distance
		FROM knowledge_vec_index v
		JOIN knowledge_vec_index_meta m ON m.vector_id = v.vector_id
		WHERE m.collection = ?
		ORDER BY distance
		LIMIT ?
	`, vecLiteral, collection, topK)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]VectorHit, 0, topK)
	for rows.Next() {
		var id, metaRaw, content string
		var distance sql.NullFloat64
		if err := rows.Scan(&id, &metaRaw, &content, &distance); err != nil {
			return nil, err
		}
		meta := map[string]interface{}{}
		if strings.TrimSpace(metaRaw) != "" {
			_ = json.Unmarshal([]byte(metaRaw), &meta)
		}
		score := 0.0
		if distance.Valid {
			score = 1.0 / (1.0 + math.Max(distance.Float64, 0))
		}
		out = append(out, VectorHit{
			ID:       strings.TrimSpace(id),
			Score:    score,
			Metadata: meta,
			Content:  strings.TrimSpace(content),
		})
	}
	return out, nil
}

func textToFeatureVec(text string) map[int]float64 {
	normalized := normalizeFeatureText(text)
	if normalized == "" {
		return nil
	}
	runes := []rune(normalized)
	out := make(map[int]float64, len(runes)*2)
	for _, token := range strings.Fields(normalized) {
		if token == "" {
			continue
		}
		idx := stableFeatureIndex("w:"+token, 384)
		out[idx] += 1
	}
	if len(runes) >= 2 {
		for i := 0; i < len(runes)-1; i++ {
			bigram := string(runes[i : i+2])
			idx := stableFeatureIndex("b:"+bigram, 384)
			out[idx] += 1
		}
	}
	return out
}

func featureVecToSQLiteLiteral(m map[int]float64, dim int) string {
	if dim <= 0 {
		dim = 384
	}
	values := make([]float64, dim)
	for idx, v := range m {
		if idx >= 0 && idx < dim {
			values[idx] = v
		}
	}
	norm := 0.0
	for _, v := range values {
		norm += v * v
	}
	if norm > 0 {
		norm = math.Sqrt(norm)
		for i := range values {
			values[i] = values[i] / norm
		}
	}
	parts := make([]string, 0, dim)
	for _, v := range values {
		parts = append(parts, strconv.FormatFloat(v, 'f', 6, 64))
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func stableFeatureIndex(token string, dim int) int {
	if dim <= 0 {
		dim = 384
	}
	var h uint64 = 1469598103934665603
	const prime uint64 = 1099511628211
	for i := 0; i < len(token); i++ {
		h ^= uint64(token[i])
		h *= prime
	}
	return int(h % uint64(dim))
}

func normalizeFeatureText(text string) string {
	text = strings.TrimSpace(strings.ToLower(text))
	if text == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(text))
	lastSpace := false
	for _, r := range text {
		if unicode.IsSpace(r) {
			if !lastSpace {
				b.WriteRune(' ')
				lastSpace = true
			}
			continue
		}
		switch r {
		case ',', '.', ';', ':', '?', '!', '，', '。', '；', '：', '？', '！', '、', '|', '/', '\\', '-', '_', '+', '(', ')', '[', ']', '{', '}', '"', '\'', '`':
			if !lastSpace {
				b.WriteRune(' ')
				lastSpace = true
			}
			continue
		default:
			b.WriteRune(r)
			lastSpace = false
		}
	}
	return strings.TrimSpace(b.String())
}
