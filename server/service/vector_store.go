package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"unicode"

	"kefu-server/models"
	"kefu-server/service/ai/embedding"
	"kefu-server/store"
	"kefu-server/utils/logger"
)

type VectorHit struct {
	ID       string                 `json:"id"`
	Score    float64                `json:"score"`
	Metadata map[string]interface{} `json:"metadata"`
	Content  string                 `json:"content"`
}

type VectorStore interface {
	Health(ctx context.Context) error
	EnsureCollection(ctx context.Context, name string) error
	DeleteCollection(ctx context.Context, name string) error
	InsertText(ctx context.Context, collection, vectorID, text string, metadata map[string]interface{}) error
	DeleteVector(ctx context.Context, collection, vectorID string) error
	SearchText(ctx context.Context, collection, query string, topK int) ([]VectorHit, error)
}

type VectorStoreHealth struct {
	Status  string `json:"status"`
	Ready   bool   `json:"ready"`
	Backend string `json:"backend"`
	Mode    string `json:"mode"`
	Message string `json:"message,omitempty"`
}

var (
	vectorStoreOnce sync.Once
	vectorStoreInst VectorStore
)

func GetVectorStore() VectorStore {
	vectorStoreOnce.Do(func() {
		vectorStoreInst = NewSQLiteVecStore()
	})
	return vectorStoreInst
}

func GetVectorStoreHealth(ctx context.Context) VectorStoreHealth {
	if ctx == nil {
		ctx = context.Background()
	}
	health := VectorStoreHealth{
		Status:  "ok",
		Ready:   true,
		Backend: "sqlite-vec",
		Mode:    "vector",
	}
	if err := GetVectorStore().Health(ctx); err != nil {
		health.Status = "degraded"
		health.Ready = false
		health.Backend = "sqlite"
		health.Mode = "fallback"
		health.Message = err.Error()
	}
	return health
}

type SQLiteVecStore struct {
	once        sync.Once
	ready       bool
	lastInitErr error
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
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("collection is empty")
	}
	if store.DB == nil {
		return fmt.Errorf("database not initialized")
	}
	if err := s.Health(ctx); err != nil {
		logger.Warnf("vector store ensure collection fallback collection=%s err=%v", name, err)
	}
	row := models.KnowledgeVectorCollection{Name: name}
	return store.DB.WithContext(ctx).Where("name = ?", name).FirstOrCreate(&row).Error
}

func (s *SQLiteVecStore) DeleteCollection(ctx context.Context, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	if store.DB == nil {
		return fmt.Errorf("database not initialized")
	}
	if err := s.Health(ctx); err != nil {
		logger.Warnf("vector store delete collection fallback collection=%s err=%v", name, err)
		tx := store.DB.WithContext(ctx)
		if err := tx.Where("collection = ?", name).Delete(&models.KnowledgeVectorEntry{}).Error; err != nil {
			return err
		}
		return tx.Where("name = ?", name).Delete(&models.KnowledgeVectorCollection{}).Error
	}
	sqlDB, err := store.DB.DB()
	if err != nil {
		return err
	}

	vecTables := s.getAllVecTableNames(ctx, sqlDB)
	metaTables := s.getAllMetaTableNames(ctx, sqlDB)

	for _, vt := range vecTables {
		ids, _ := s.getVectorIDsByCollection(ctx, sqlDB, vt, name)
		for _, id := range ids {
			_, _ = sqlDB.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s WHERE vector_id = ?", vt), id)
		}
	}
	for _, mt := range metaTables {
		_, _ = sqlDB.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s WHERE collection = ?", mt), name)
	}

	tx := store.DB.WithContext(ctx)
	if err := tx.Where("collection = ?", name).Delete(&models.KnowledgeVectorEntry{}).Error; err != nil {
		return err
	}
	return tx.Where("name = ?", name).Delete(&models.KnowledgeVectorCollection{}).Error
}

func (s *SQLiteVecStore) InsertText(ctx context.Context, collection, vectorID, text string, metadata map[string]interface{}) error {
	collection = strings.TrimSpace(collection)
	vectorID = strings.TrimSpace(vectorID)
	text = strings.TrimSpace(text)
	if collection == "" || vectorID == "" || text == "" {
		return fmt.Errorf("invalid vector input")
	}
	if store.DB == nil {
		return fmt.Errorf("database not initialized")
	}

	metaRaw := "{}"
	if metadata != nil {
		if b, err := json.Marshal(metadata); err == nil {
			metaRaw = string(b)
		}
	}
	if err := s.EnsureCollection(ctx, collection); err != nil {
		return err
	}

	row := models.KnowledgeVectorEntry{
		Collection: collection,
		VectorID:   vectorID,
		Content:    text,
		Metadata:   metaRaw,
	}

	if err := s.Health(ctx); err != nil {
		logger.Warnf("vector store insert fallback collection=%s vector_id=%s err=%v", collection, vectorID, err)
		return store.DB.WithContext(ctx).Where("vector_id = ?", vectorID).Assign(row).FirstOrCreate(&row).Error
	}

	embProvider, embDims, embErr := s.getActiveEmbeddingProvider(ctx)
	var feature []float32
	var dims int

	if embErr == nil && embProvider != nil {
		vec, err := embProvider.GetEmbedding(ctx, text)
		if err != nil {
			logger.Warnf("embedding provider failed, falling back to word-hash: %v", err)
			feature = textToFeatureVec32(text)
			dims = 384
		} else {
			feature = vec
			dims = embDims
		}
	} else {
		feature = textToFeatureVec32(text)
		dims = 384
	}

	vecTableName, metaTableName, err := s.ensureVecTable(ctx, dims)
	if err != nil {
		return err
	}

	vecLiteral := float32SliceToSQLiteLiteral(feature)

	sqlDB, err := store.DB.DB()
	if err != nil {
		return err
	}
	if _, err := sqlDB.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s WHERE vector_id = ?", vecTableName), vectorID); err != nil {
		return err
	}
	if _, err := sqlDB.ExecContext(ctx, fmt.Sprintf("INSERT INTO %s(vector_id, embedding) VALUES(?, vec_f32(?))", vecTableName), vectorID, vecLiteral); err != nil {
		return err
	}
	if _, err := sqlDB.ExecContext(ctx, fmt.Sprintf(`
		INSERT INTO %s(vector_id, collection, metadata, content)
		VALUES(?, ?, ?, ?)
		ON CONFLICT(vector_id) DO UPDATE SET
			collection=excluded.collection,
			metadata=excluded.metadata,
			content=excluded.content
	`, metaTableName), vectorID, collection, metaRaw, text); err != nil {
		return err
	}

	return store.DB.WithContext(ctx).Where("vector_id = ?", vectorID).Assign(row).FirstOrCreate(&row).Error
}

func (s *SQLiteVecStore) DeleteVector(ctx context.Context, collection, vectorID string) error {
	vectorID = strings.TrimSpace(vectorID)
	if vectorID == "" {
		return nil
	}
	collection = strings.TrimSpace(collection)
	if store.DB == nil {
		return fmt.Errorf("database not initialized")
	}
	if err := s.Health(ctx); err != nil {
		logger.Warnf("vector store delete vector fallback collection=%s vector_id=%s err=%v", collection, vectorID, err)
		tx := store.DB.WithContext(ctx)
		if collection == "" {
			return tx.Where("vector_id = ?", vectorID).Delete(&models.KnowledgeVectorEntry{}).Error
		}
		return tx.Where("collection = ? AND vector_id = ?", collection, vectorID).Delete(&models.KnowledgeVectorEntry{}).Error
	}
	sqlDB, err := store.DB.DB()
	if err != nil {
		return err
	}

	vecTables := s.getAllVecTableNames(ctx, sqlDB)
	metaTables := s.getAllMetaTableNames(ctx, sqlDB)

	for _, vt := range vecTables {
		_, _ = sqlDB.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s WHERE vector_id = ?", vt), vectorID)
	}

	for _, mt := range metaTables {
		if collection == "" {
			_, _ = sqlDB.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s WHERE vector_id = ?", mt), vectorID)
		} else {
			_, _ = sqlDB.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s WHERE collection = ? AND vector_id = ?", mt), collection, vectorID)
		}
	}

	tx := store.DB.WithContext(ctx)
	if collection == "" {
		return tx.Where("vector_id = ?", vectorID).Delete(&models.KnowledgeVectorEntry{}).Error
	}
	return tx.Where("collection = ? AND vector_id = ?", collection, vectorID).Delete(&models.KnowledgeVectorEntry{}).Error
}

func (s *SQLiteVecStore) SearchText(ctx context.Context, collection, query string, topK int) ([]VectorHit, error) {
	collection = strings.TrimSpace(collection)
	query = strings.TrimSpace(query)
	if collection == "" || query == "" {
		return []VectorHit{}, nil
	}
	if store.DB == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	if topK <= 0 {
		topK = 5
	}
	if topK > 20 {
		topK = 20
	}
	if err := s.Health(ctx); err != nil {
		logger.Warnf("vector store search fallback collection=%s err=%v", collection, err)
		return s.searchByContentFallback(ctx, collection, query, topK)
	}

	return s.searchBySQLiteVec(ctx, collection, query, topK)
}

func (s *SQLiteVecStore) searchByContentFallback(ctx context.Context, collection, query string, topK int) ([]VectorHit, error) {
	var rows []models.KnowledgeVectorEntry
	if err := store.DB.WithContext(ctx).Where("collection = ?", collection).Find(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return []VectorHit{}, nil
	}

	queryVec := textToFeatureVec32(query)
	type scoredHit struct {
		hit   VectorHit
		score float64
	}
	scored := make([]scoredHit, 0, len(rows))
	for _, row := range rows {
		content := strings.TrimSpace(row.Content)
		if content == "" {
			continue
		}
		meta := map[string]interface{}{}
		if strings.TrimSpace(row.Metadata) != "" {
			_ = json.Unmarshal([]byte(row.Metadata), &meta)
		}
		score := cosineScore32(queryVec, textToFeatureVec32(content))
		if score <= 0 {
			continue
		}
		scored = append(scored, scoredHit{
			score: score,
			hit: VectorHit{
				ID:       strings.TrimSpace(row.VectorID),
				Score:    score,
				Metadata: meta,
				Content:  content,
			},
		})
	}
	sort.Slice(scored, func(i, j int) bool {
		if scored[i].score == scored[j].score {
			return scored[i].hit.ID < scored[j].hit.ID
		}
		return scored[i].score > scored[j].score
	})
	if len(scored) > topK {
		scored = scored[:topK]
	}
	out := make([]VectorHit, 0, len(scored))
	for _, item := range scored {
		out = append(out, item.hit)
	}
	return out, nil
}

func (s *SQLiteVecStore) searchBySQLiteVec(ctx context.Context, collection, query string, topK int) ([]VectorHit, error) {
	sqlDB, err := store.DB.DB()
	if err != nil {
		return nil, err
	}

	embProvider, embDims, embErr := s.getActiveEmbeddingProvider(ctx)
	var feature []float32
	var dims int

	if embErr == nil && embProvider != nil {
		vec, err := embProvider.GetEmbedding(ctx, query)
		if err != nil {
			logger.Warnf("embedding provider failed for search, falling back to word-hash: %v", err)
			feature = textToFeatureVec32(query)
			dims = 384
		} else {
			feature = vec
			dims = embDims
		}
	} else {
		feature = textToFeatureVec32(query)
		dims = 384
	}

	vecTableName, metaTableName, err := s.ensureVecTable(ctx, dims)
	if err != nil {
		return nil, err
	}

	vecLiteral := float32SliceToSQLiteLiteral(feature)

	rows, err := sqlDB.QueryContext(ctx, fmt.Sprintf(`
		SELECT m.vector_id, m.metadata, m.content, vec_distance_l2(v.embedding, vec_f32(?)) AS distance
		FROM %s v
		JOIN %s m ON m.vector_id = v.vector_id
		WHERE m.collection = ?
		ORDER BY distance
		LIMIT ?
	`, vecTableName, metaTableName), vecLiteral, collection, topK)
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

func (s *SQLiteVecStore) getActiveEmbeddingProvider(ctx context.Context) (embedding.Provider, int, error) {
	cfg, err := GetEnabledAPIModelConfigByType(string(models.AIModelTypeEmbedding))
	if err != nil {
		return nil, 0, err
	}
	provider, err := embedding.NewProvider(&cfg)
	if err != nil {
		return nil, 0, err
	}
	return provider, cfg.Dims, nil
}

func (s *SQLiteVecStore) ensureVecTable(ctx context.Context, dims int) (vecTable string, metaTable string, err error) {
	sqlDB, err := store.DB.DB()
	if err != nil {
		return "", "", err
	}

	if dims == 384 {
		return "knowledge_vec_index", "knowledge_vec_index_meta", nil
	}

	vecTable = fmt.Sprintf("knowledge_vec_%d", dims)
	metaTable = fmt.Sprintf("knowledge_vec_%d_meta", dims)

	createVec := fmt.Sprintf(`CREATE VIRTUAL TABLE IF NOT EXISTS %s USING vec0(
		vector_id TEXT PRIMARY KEY,
		embedding float[%d]
	)`, vecTable, dims)
	if _, err := sqlDB.ExecContext(ctx, createVec); err != nil {
		return "", "", fmt.Errorf("create vec table %s failed: %w", vecTable, err)
	}

	createMeta := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		vector_id TEXT PRIMARY KEY,
		collection TEXT NOT NULL,
		metadata TEXT,
		content TEXT
	)`, metaTable)
	if _, err := sqlDB.ExecContext(ctx, createMeta); err != nil {
		return "", "", fmt.Errorf("create meta table %s failed: %w", metaTable, err)
	}

	createIdx := fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_%s_collection ON %s(collection)`, metaTable, metaTable)
	if _, err := sqlDB.ExecContext(ctx, createIdx); err != nil {
		return "", "", err
	}

	var existing models.VecTable
	result := store.DB.WithContext(ctx).Where("table_name = ?", vecTable).First(&existing)
	if result.Error != nil {
		cfg, _ := GetEnabledAPIModelConfigByType(string(models.AIModelTypeEmbedding))
		configID := uint(0)
		if cfg.ID > 0 {
			configID = cfg.ID
		}
		vt := models.VecTable{
			TableName: vecTable,
			Dims:      dims,
			ConfigID:  configID,
		}
		_ = store.DB.WithContext(ctx).Where("table_name = ?", vecTable).FirstOrCreate(&vt).Error
	}

	return vecTable, metaTable, nil
}

func (s *SQLiteVecStore) getAllVecTableNames(ctx context.Context, sqlDB *sql.DB) []string {
	names := []string{"knowledge_vec_index"}
	rows, err := sqlDB.QueryContext(ctx, "SELECT table_name FROM vec_tables")
	if err != nil {
		return names
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err == nil && name != "" {
			names = append(names, name)
		}
	}
	return names
}

func (s *SQLiteVecStore) getAllMetaTableNames(ctx context.Context, sqlDB *sql.DB) []string {
	names := []string{"knowledge_vec_index_meta"}
	rows, err := sqlDB.QueryContext(ctx, "SELECT table_name FROM vec_tables")
	if err != nil {
		return names
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err == nil && name != "" {
			metaName := name + "_meta"
			names = append(names, metaName)
		}
	}
	return names
}

func (s *SQLiteVecStore) getVectorIDsByCollection(ctx context.Context, sqlDB *sql.DB, vecTable, collection string) ([]string, error) {
	metaTable := vecTable + "_meta"
	if vecTable == "knowledge_vec_index" {
		metaTable = "knowledge_vec_index_meta"
	}
	rows, err := sqlDB.QueryContext(ctx, fmt.Sprintf("SELECT vector_id FROM %s WHERE collection = ?", metaTable), collection)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := make([]string, 0, 128)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, strings.TrimSpace(id))
		}
	}
	return ids, nil
}

func textToFeatureVec32(text string) []float32 {
	m := textToFeatureVec(text)
	if m == nil {
		return make([]float32, 384)
	}
	dim := 384
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
	out := make([]float32, dim)
	for i, v := range values {
		out[i] = float32(v)
	}
	return out
}

func float32SliceToSQLiteLiteral(vec []float32) string {
	parts := make([]string, 0, len(vec))
	for _, v := range vec {
		parts = append(parts, strconv.FormatFloat(float64(v), 'f', 6, 64))
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func cosineScore32(a, b []float32) float64 {
	if len(a) == 0 || len(a) != len(b) {
		return 0
	}
	dot := 0.0
	normA := 0.0
	normB := 0.0
	for i := range a {
		av := float64(a[i])
		bv := float64(b[i])
		dot += av * bv
		normA += av * av
		normB += bv * bv
	}
	if normA <= 0 || normB <= 0 {
		return 0
	}
	score := dot / (math.Sqrt(normA) * math.Sqrt(normB))
	if score < 0 {
		return 0
	}
	return score
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
