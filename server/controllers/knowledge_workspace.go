package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"kefu-server/config"
	"kefu-server/models"
	"kefu-server/service"
	"kefu-server/store"
	"kefu-server/utils/logger"
	"kefu-server/utils/response"
)

// KnowledgeWorkspaceController 提供知识库工作区 API。
type KnowledgeWorkspaceController struct{}

var nonCollectionChars = regexp.MustCompile(`[^a-zA-Z0-9_]+`)

func (kc *KnowledgeWorkspaceController) buildVectorStore() service.VectorStore {
	return service.GetVectorStore()
}

func normalizeCollectionName(appID string, baseID uint) string {
	appSafe := strings.ToLower(strings.TrimSpace(appID))
	appSafe = strings.ReplaceAll(appSafe, "-", "_")
	appSafe = nonCollectionChars.ReplaceAllString(appSafe, "_")
	appSafe = strings.Trim(appSafe, "_")
	if appSafe == "" {
		appSafe = "app"
	}
	return fmt.Sprintf("kb_%s_%d", appSafe, baseID)
}

func (kc *KnowledgeWorkspaceController) hasAppAccess(c *gin.Context, appID string) bool {
	appID = strings.TrimSpace(appID)
	if appID == "" {
		return false
	}
	if IsAdmin(c) {
		return true
	}
	userName, _ := getAuthUser(c)
	if strings.TrimSpace(userName) == "" || store.DB == nil {
		return false
	}
	var user models.User
	if err := store.DB.Where("username = ?", userName).First(&user).Error; err != nil {
		return false
	}
	return isAgentForApp(user.Apps, appID)
}

func (kc *KnowledgeWorkspaceController) getBaseByID(c *gin.Context) (*models.KnowledgeBase, bool) {
	baseID, err := strconv.ParseUint(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || baseID == 0 {
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeKnowledgeBaseDeleteInvalid)
		return nil, false
	}
	var base models.KnowledgeBase
	if err := store.DB.Where("id = ?", uint(baseID)).First(&base).Error; err != nil {
		response.ResponseError(c, http.StatusNotFound, response.ErrCodeKnowledgeBaseNotFound)
		return nil, false
	}
	if !kc.hasAppAccess(c, base.AppID) {
		response.ResponseError(c, http.StatusForbidden, response.ErrCodeKnowledgeBaseAppAccessDenied)
		return nil, false
	}
	return &base, true
}

// ListBases 返回知识库列表（按当前账号可访问 app 过滤）。
func (kc *KnowledgeWorkspaceController) ListBases(c *gin.Context) {
	appID := strings.TrimSpace(c.Query("app_id"))
	if appID != "" && !kc.hasAppAccess(c, appID) {
		response.ResponseError(c, http.StatusForbidden, response.ErrCodeKnowledgeBaseAppAccessDenied)
		return
	}

	query := store.DB.Model(&models.KnowledgeBase{})
	if appID != "" {
		query = query.Where("app_id = ?", appID)
	}
	var rows []models.KnowledgeBase
	if err := query.Order("updated_at DESC").Find(&rows).Error; err != nil {
		logger.Errorf("knowledge base list failed app_id=%s err=%v", appID, err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeKnowledgeBaseListFailed)
		return
	}
	if !IsAdmin(c) {
		filtered := make([]models.KnowledgeBase, 0, len(rows))
		for _, row := range rows {
			if kc.hasAppAccess(c, row.AppID) {
				filtered = append(filtered, row)
			}
		}
		rows = filtered
	}
	response.ResponseSuccess(c, gin.H{"data": rows})
}

// CreateBase 新建知识库（客服与管理员）。
// 说明：
// 1. 先写入本地数据库并立即返回，避免前端因外部向量库初始化耗时而超时；
// 2. 向量集合初始化改为后台异步执行，失败仅记录日志，不回滚知识库记录。
func (kc *KnowledgeWorkspaceController) CreateBase(c *gin.Context) {
	var req struct {
		AppID       string `json:"app_id" binding:"required"`
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Errorf("knowledge base create params invalid err=%v", err)
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeKnowledgeBaseCreateInvalid)
		return
	}
	req.AppID = strings.TrimSpace(req.AppID)
	req.Name = strings.TrimSpace(req.Name)
	req.Description = strings.TrimSpace(req.Description)
	if req.AppID == "" || req.Name == "" {
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeKnowledgeBaseCreateInvalid)
		return
	}
	if !kc.hasAppAccess(c, req.AppID) {
		response.ResponseError(c, http.StatusForbidden, response.ErrCodeKnowledgeBaseAppAccessDenied)
		return
	}

	var app models.App
	if err := store.DB.Where("app_id = ?", req.AppID).First(&app).Error; err != nil {
		response.ResponseError(c, http.StatusNotFound, response.ErrCodeAppNotFound)
		return
	}
	base := models.KnowledgeBase{
		AppID:       req.AppID,
		Name:        req.Name,
		Description: req.Description,
		Collection:  "",
	}
	if err := store.DB.Create(&base).Error; err != nil {
		logger.Errorf("knowledge base create failed app_id=%s err=%v", req.AppID, err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeKnowledgeBaseCreateFailed)
		return
	}
	base.Collection = normalizeCollectionName(base.AppID, base.ID)
	if err := store.DB.Save(&base).Error; err != nil {
		logger.Errorf("knowledge base update collection failed id=%d err=%v", base.ID, err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeKnowledgeBaseCreateFailed)
		return
	}

	// 异步确保向量集合存在，不阻塞接口响应。
	collection := base.Collection
	baseID := base.ID
	go func() {
		vdb := kc.buildVectorStore()
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		if err := vdb.EnsureCollection(ctx, collection); err != nil {
			logger.Errorf("knowledge base async ensure collection failed id=%d collection=%s err=%v", baseID, collection, err)
			return
		}
		logger.Infof("knowledge base async ensure collection success id=%d collection=%s", baseID, collection)
	}()

	RecordAudit(c, "knowledge_base.create", "knowledge_base", strconv.FormatUint(uint64(base.ID), 10), "success", base.Name)
	response.ResponseSuccess(c, gin.H{"item": base})
}

// UpdateBase 更新知识库元信息（客服与管理员）。
func (kc *KnowledgeWorkspaceController) UpdateBase(c *gin.Context) {
	var req struct {
		ID          uint   `json:"id" binding:"required"`
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeKnowledgeBaseUpdateInvalid)
		return
	}
	var base models.KnowledgeBase
	if err := store.DB.Where("id = ?", req.ID).First(&base).Error; err != nil {
		response.ResponseError(c, http.StatusNotFound, response.ErrCodeKnowledgeBaseNotFound)
		return
	}
	if !kc.hasAppAccess(c, base.AppID) {
		response.ResponseError(c, http.StatusForbidden, response.ErrCodeKnowledgeBaseAppAccessDenied)
		return
	}
	base.Name = strings.TrimSpace(req.Name)
	base.Description = strings.TrimSpace(req.Description)
	if base.Name == "" {
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeKnowledgeBaseUpdateInvalid)
		return
	}
	if err := store.DB.Save(&base).Error; err != nil {
		logger.Errorf("knowledge base update failed id=%d err=%v", req.ID, err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeKnowledgeBaseUpdateFailed)
		return
	}
	RecordAudit(c, "knowledge_base.update", "knowledge_base", strconv.FormatUint(uint64(base.ID), 10), "success", base.Name)
	response.ResponseSuccess(c, gin.H{"item": base})
}

// DeleteBase 删除知识库（客服与管理员，硬删除）。
func (kc *KnowledgeWorkspaceController) DeleteBase(c *gin.Context) {
	idRaw := strings.TrimSpace(c.Query("id"))
	id, err := strconv.ParseUint(idRaw, 10, 64)
	if err != nil || id == 0 {
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeKnowledgeBaseDeleteInvalid)
		return
	}
	var base models.KnowledgeBase
	if err := store.DB.Where("id = ?", uint(id)).First(&base).Error; err != nil {
		response.ResponseError(c, http.StatusNotFound, response.ErrCodeKnowledgeBaseNotFound)
		return
	}
	if !kc.hasAppAccess(c, base.AppID) {
		response.ResponseError(c, http.StatusForbidden, response.ErrCodeKnowledgeBaseAppAccessDenied)
		return
	}
	var docs []models.KnowledgeDocument
	if err := store.DB.Where("base_id = ?", base.ID).Find(&docs).Error; err == nil {
		vdb := kc.buildVectorStore()
		for _, doc := range docs {
			var chunks []models.KnowledgeChunk
			_ = store.DB.Where("document_id = ?", doc.ID).Find(&chunks).Error
			for _, chunk := range chunks {
				_ = vdb.DeleteVector(c.Request.Context(), base.Collection, chunk.VectorID)
			}
			_ = store.DB.Unscoped().Where("document_id = ?", doc.ID).Delete(&models.KnowledgeChunk{}).Error
		}
	}
	_ = store.DB.Unscoped().Where("base_id = ?", base.ID).Delete(&models.KnowledgeDocument{}).Error
	_ = store.DB.Unscoped().Where("base_id = ?", base.ID).Delete(&models.KnowledgeQAFeedback{}).Error
	if err := store.DB.Unscoped().Delete(&models.KnowledgeBase{}, base.ID).Error; err != nil {
		logger.Errorf("knowledge base delete failed id=%d err=%v", base.ID, err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeKnowledgeBaseDeleteFailed)
		return
	}
	RecordAudit(c, "knowledge_base.delete", "knowledge_base", idRaw, "success", base.Name)
	response.ResponseSuccess(c, gin.H{"message": "delete successful"})
}

// ListDocuments 返回知识库文档列表。
func (kc *KnowledgeWorkspaceController) ListDocuments(c *gin.Context) {
	base, ok := kc.getBaseByID(c)
	if !ok {
		return
	}
	keyword := strings.TrimSpace(c.Query("keyword"))
	query := store.DB.Model(&models.KnowledgeDocument{}).Where("base_id = ?", base.ID)
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("name LIKE ? OR file_name LIKE ?", like, like)
	}
	var docs []models.KnowledgeDocument
	if err := query.Order("updated_at DESC").Find(&docs).Error; err != nil {
		logger.Errorf("knowledge document list failed base_id=%d err=%v", base.ID, err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeKnowledgeDocumentListFailed)
		return
	}
	response.ResponseSuccess(c, gin.H{"data": docs})
}

func normalizeUploadDir() string {
	cfg := config.GetConfig()
	if cfg != nil && strings.TrimSpace(cfg.Admin.UploadDir) != "" {
		return strings.TrimSpace(cfg.Admin.UploadDir)
	}
	return filepath.Clean("data/uploads")
}

func allowKnowledgeFileExt(name string) bool {
	ext := strings.ToLower(strings.TrimSpace(filepath.Ext(name)))
	switch ext {
	case ".txt", ".md", ".pdf", ".docx", ".csv", ".tsv", ".xlsx":
		return true
	default:
		return false
	}
}

// UploadDocument 上传并入库文档（客服与管理员）。
func (kc *KnowledgeWorkspaceController) UploadDocument(c *gin.Context) {
	base, ok := kc.getBaseByID(c)
	if !ok {
		return
	}
	fileHeader, err := c.FormFile("file")
	if err != nil {
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeKnowledgeDocumentUploadInvalid)
		return
	}
	if fileHeader.Size <= 0 {
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeUploadEmptyFile)
		return
	}
	if !allowKnowledgeFileExt(fileHeader.Filename) {
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeKnowledgeDocumentFormatInvalid)
		return
	}

	uploadDir := normalizeUploadDir()
	if mkErr := os.MkdirAll(uploadDir, 0755); mkErr != nil {
		logger.Errorf("knowledge upload mkdir failed dir=%s err=%v", uploadDir, mkErr)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeUploadMkdirFailed)
		return
	}
	safeName := fmt.Sprintf("kb_%d_%d_%s", base.ID, time.Now().UnixNano(), filepath.Base(fileHeader.Filename))
	savePath := filepath.Join(uploadDir, safeName)
	if err := c.SaveUploadedFile(fileHeader, savePath); err != nil {
		logger.Errorf("knowledge upload save file failed path=%s err=%v", savePath, err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeUploadSaveFailed)
		return
	}
	fileURL := "/uploads/" + safeName
	content, _, sourceName, parseErr := extractKnowledgeContentFromURL(fileURL, fileHeader.Filename)
	if parseErr != nil {
		logger.Errorf("knowledge upload parse failed file=%s err=%v", fileHeader.Filename, parseErr)
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeKnowledgeDocumentParseFailed)
		return
	}
	if strings.TrimSpace(content) == "" {
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeKnowledgeDocumentParseFailed)
		return
	}

	doc := models.KnowledgeDocument{
		BaseID:           base.ID,
		AppID:            base.AppID,
		Name:             strings.TrimSpace(fileHeader.Filename),
		FileName:         filepath.Base(fileHeader.Filename),
		FileSize:         fileHeader.Size,
		FileType:         strings.TrimPrefix(strings.ToLower(filepath.Ext(fileHeader.Filename)), "."),
		FileURL:          fileURL,
		Status:           "indexing",
		ErrorMessage:     "",
		ChunkCount:       0,
		RawContent:       content,
		VectorCollection: base.Collection,
	}
	if strings.TrimSpace(doc.Name) == "" {
		doc.Name = sourceName
	}
	if strings.TrimSpace(doc.FileType) == "" {
		doc.FileType = strings.TrimPrefix(strings.ToLower(filepath.Ext(fileHeader.Filename)), ".")
	}
	if err := store.DB.Create(&doc).Error; err != nil {
		logger.Errorf("knowledge document create failed base_id=%d err=%v", base.ID, err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeKnowledgeDocumentSaveFailed)
		return
	}

	vdb := kc.buildVectorStore()
	indexCtx, cancel := knowledgeIndexContext()
	defer cancel()
	if err := vdb.EnsureCollection(indexCtx, base.Collection); err != nil {
		doc.Status = "failed"
		doc.ErrorMessage = err.Error()
		_ = store.DB.Save(&doc).Error
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeKnowledgeVectorCollectionFailed)
		return
	}
	_, rebuildErr := service.RebuildChunksFromRawContent(indexCtx, vdb, &doc)
	if rebuildErr != nil {
		doc.Status = "failed"
		doc.ErrorMessage = rebuildErr.Error()
		_ = store.DB.Save(&doc).Error
		logger.Errorf("knowledge document index failed doc_id=%d err=%v", doc.ID, rebuildErr)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeKnowledgeDocumentIndexFailed)
		return
	}
	RecordAudit(c, "knowledge_document.upload", "knowledge_document", strconv.FormatUint(uint64(doc.ID), 10), "success", doc.Name)
	response.ResponseSuccess(c, gin.H{"item": doc})
}

// ReindexDocument 重新索引文档（客服与管理员）。
func (kc *KnowledgeWorkspaceController) ReindexDocument(c *gin.Context) {
	base, ok := kc.getBaseByID(c)
	if !ok {
		return
	}
	var req struct {
		DocumentID uint `json:"document_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.DocumentID == 0 {
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeKnowledgeDocumentReindexFailed)
		return
	}
	var doc models.KnowledgeDocument
	if err := store.DB.Where("id = ? AND base_id = ?", req.DocumentID, base.ID).First(&doc).Error; err != nil {
		response.ResponseError(c, http.StatusNotFound, response.ErrCodeKnowledgeDocumentNotFound)
		return
	}
	vdb := kc.buildVectorStore()
	indexCtx, cancel := knowledgeIndexContext()
	defer cancel()
	if err := vdb.EnsureCollection(indexCtx, base.Collection); err != nil {
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeKnowledgeVectorCollectionFailed)
		return
	}
	_, err := service.RebuildChunksFromRawContent(indexCtx, vdb, &doc)
	if err != nil {
		logger.Errorf("knowledge document reindex failed doc_id=%d err=%v", doc.ID, err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeKnowledgeDocumentReindexFailed)
		return
	}
	RecordAudit(c, "knowledge_document.reindex", "knowledge_document", strconv.FormatUint(uint64(doc.ID), 10), "success", doc.Name)
	response.ResponseSuccess(c, gin.H{"item": doc})
}

// DeleteDocument 删除文档（客服与管理员，硬删除）。
func (kc *KnowledgeWorkspaceController) DeleteDocument(c *gin.Context) {
	base, ok := kc.getBaseByID(c)
	if !ok {
		return
	}
	docID, err := strconv.ParseUint(strings.TrimSpace(c.Param("docID")), 10, 64)
	if err != nil || docID == 0 {
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeKnowledgeDocumentDeleteFailed)
		return
	}
	var doc models.KnowledgeDocument
	if err := store.DB.Where("id = ? AND base_id = ?", uint(docID), base.ID).First(&doc).Error; err != nil {
		response.ResponseError(c, http.StatusNotFound, response.ErrCodeKnowledgeDocumentNotFound)
		return
	}
	var chunks []models.KnowledgeChunk
	_ = store.DB.Where("document_id = ?", doc.ID).Find(&chunks).Error
	vdb := kc.buildVectorStore()
	for _, chunk := range chunks {
		_ = vdb.DeleteVector(c.Request.Context(), base.Collection, chunk.VectorID)
	}
	_ = store.DB.Unscoped().Where("document_id = ?", doc.ID).Delete(&models.KnowledgeChunk{}).Error
	if err := store.DB.Unscoped().Delete(&models.KnowledgeDocument{}, doc.ID).Error; err != nil {
		logger.Errorf("knowledge document delete failed doc_id=%d err=%v", doc.ID, err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeKnowledgeDocumentDeleteFailed)
		return
	}
	if strings.HasPrefix(doc.FileURL, "/uploads/") {
		uploadDir := normalizeUploadDir()
		name := filepath.Base(strings.TrimPrefix(doc.FileURL, "/uploads/"))
		_ = os.Remove(filepath.Join(uploadDir, name))
	}
	RecordAudit(c, "knowledge_document.delete", "knowledge_document", strconv.FormatUint(uint64(doc.ID), 10), "success", doc.Name)
	response.ResponseSuccess(c, gin.H{"message": "delete successful"})
}

// ListChunks 返回知识片段列表。
func (kc *KnowledgeWorkspaceController) ListChunks(c *gin.Context) {
	base, ok := kc.getBaseByID(c)
	if !ok {
		return
	}
	keyword := strings.TrimSpace(c.Query("keyword"))
	docIDRaw := strings.TrimSpace(c.Query("document_id"))
	query := store.DB.Model(&models.KnowledgeChunk{}).Where("base_id = ?", base.ID)
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("content LIKE ?", like)
	}
	if docIDRaw != "" {
		docID, err := strconv.ParseUint(docIDRaw, 10, 64)
		if err == nil && docID > 0 {
			query = query.Where("document_id = ?", uint(docID))
		}
	}
	var rows []models.KnowledgeChunk
	if err := query.Order("updated_at DESC").Limit(500).Find(&rows).Error; err != nil {
		logger.Errorf("knowledge chunk list failed base_id=%d err=%v", base.ID, err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeKnowledgeChunkListFailed)
		return
	}
	docIDs := make([]uint, 0, len(rows))
	for _, row := range rows {
		docIDs = append(docIDs, row.DocumentID)
	}
	docMap := map[uint]string{}
	if len(docIDs) > 0 {
		var docs []models.KnowledgeDocument
		_ = store.DB.Where("id IN ?", docIDs).Find(&docs).Error
		for _, d := range docs {
			docMap[d.ID] = d.Name
		}
	}
	data := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		data = append(data, gin.H{
			"id":            row.ID,
			"base_id":       row.BaseID,
			"document_id":   row.DocumentID,
			"document_name": docMap[row.DocumentID],
			"vector_id":     row.VectorID,
			"chunk_seq":     row.ChunkSeq,
			"content":       row.Content,
			"content_chars": row.ContentChars,
			"avg_score":     row.AvgScore,
			"hit_count":     row.HitCount,
			"updated_at":    row.UpdatedAt,
		})
	}
	response.ResponseSuccess(c, gin.H{"data": data})
}

// UpdateChunk 编辑知识片段（客服与管理员，自动重新向量化）。
func (kc *KnowledgeWorkspaceController) UpdateChunk(c *gin.Context) {
	base, ok := kc.getBaseByID(c)
	if !ok {
		return
	}
	chunkID, err := strconv.ParseUint(strings.TrimSpace(c.Param("chunkID")), 10, 64)
	if err != nil || chunkID == 0 {
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeKnowledgeChunkUpdateInvalid)
		return
	}
	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeKnowledgeChunkUpdateInvalid)
		return
	}
	var chunk models.KnowledgeChunk
	if err := store.DB.Where("id = ? AND base_id = ?", uint(chunkID), base.ID).First(&chunk).Error; err != nil {
		response.ResponseError(c, http.StatusNotFound, response.ErrCodeKnowledgeChunkNotFound)
		return
	}
	content := strings.TrimSpace(req.Content)
	if content == "" {
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeKnowledgeChunkUpdateInvalid)
		return
	}
	vdb := kc.buildVectorStore()
	meta := map[string]interface{}{
		"app_id":      chunk.AppID,
		"base_id":     chunk.BaseID,
		"document_id": chunk.DocumentID,
		"chunk_seq":   chunk.ChunkSeq,
	}
	indexCtx, cancel := knowledgeIndexContext()
	defer cancel()
	if err := vdb.InsertText(indexCtx, base.Collection, chunk.VectorID, content, meta); err != nil {
		logger.Errorf("knowledge chunk update vector failed chunk_id=%d err=%v", chunk.ID, err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeKnowledgeChunkUpdateFailed)
		return
	}
	chunk.Content = content
	chunk.ContentChars = len([]rune(content))
	if err := store.DB.Save(&chunk).Error; err != nil {
		logger.Errorf("knowledge chunk update db failed chunk_id=%d err=%v", chunk.ID, err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeKnowledgeChunkUpdateFailed)
		return
	}
	RecordAudit(c, "knowledge_chunk.update", "knowledge_chunk", strconv.FormatUint(uint64(chunk.ID), 10), "success", "")
	response.ResponseSuccess(c, gin.H{"item": chunk})
}

// DeleteChunk 删除知识片段（客服与管理员，硬删除）。
func (kc *KnowledgeWorkspaceController) DeleteChunk(c *gin.Context) {
	base, ok := kc.getBaseByID(c)
	if !ok {
		return
	}
	chunkID, err := strconv.ParseUint(strings.TrimSpace(c.Param("chunkID")), 10, 64)
	if err != nil || chunkID == 0 {
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeKnowledgeChunkDeleteFailed)
		return
	}
	var chunk models.KnowledgeChunk
	if err := store.DB.Where("id = ? AND base_id = ?", uint(chunkID), base.ID).First(&chunk).Error; err != nil {
		response.ResponseError(c, http.StatusNotFound, response.ErrCodeKnowledgeChunkNotFound)
		return
	}
	vdb := kc.buildVectorStore()
	_ = vdb.DeleteVector(c.Request.Context(), base.Collection, chunk.VectorID)
	if err := store.DB.Unscoped().Delete(&models.KnowledgeChunk{}, chunk.ID).Error; err != nil {
		logger.Errorf("knowledge chunk delete failed id=%d err=%v", chunk.ID, err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeKnowledgeChunkDeleteFailed)
		return
	}
	RecordAudit(c, "knowledge_chunk.delete", "knowledge_chunk", strconv.FormatUint(uint64(chunk.ID), 10), "success", "")
	response.ResponseSuccess(c, gin.H{"message": "delete successful"})
}

func (kc *KnowledgeWorkspaceController) searchBaseChunks(ctx context.Context, base *models.KnowledgeBase, query string, topK int) ([]service.VectorHit, error) {
	vdb := kc.buildVectorStore()
	if err := vdb.EnsureCollection(ctx, base.Collection); err != nil {
		return nil, err
	}
	hits, err := vdb.SearchText(ctx, base.Collection, query, topK)
	if err != nil {
		return nil, err
	}
	return service.RerankHits(ctx, query, hits, topK), nil
}

func writeSSEEvent(c *gin.Context, event string, payload interface{}) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", strings.TrimSpace(event), raw); err != nil {
		return err
	}
	c.Writer.Flush()
	return nil
}

// RetrieveTest 检索测试接口。
func (kc *KnowledgeWorkspaceController) RetrieveTest(c *gin.Context) {
	base, ok := kc.getBaseByID(c)
	if !ok {
		return
	}
	var req struct {
		Query string `json:"query" binding:"required"`
		TopK  int    `json:"top_k"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeKnowledgeRetrieveInvalid)
		return
	}
	query := strings.TrimSpace(req.Query)
	if query == "" {
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeKnowledgeRetrieveInvalid)
		return
	}
	searchCtx, cancel := knowledgeSearchContext()
	defer cancel()

	hits, err := kc.searchBaseChunks(searchCtx, base, query, req.TopK)
	if err != nil {
		logger.Errorf("knowledge retrieve failed base_id=%d err=%v", base.ID, err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeKnowledgeRetrieveFailed)
		return
	}

	ids := make([]string, 0, len(hits))
	for _, h := range hits {
		if strings.TrimSpace(h.ID) != "" {
			ids = append(ids, h.ID)
		}
	}
	chunkMap := map[string]models.KnowledgeChunk{}
	if len(ids) > 0 {
		var chunks []models.KnowledgeChunk
		if err := store.DB.Where("base_id = ? AND vector_id IN ?", base.ID, ids).Find(&chunks).Error; err == nil {
			for _, ck := range chunks {
				chunkMap[ck.VectorID] = ck
			}
		}
	}
	docMap := map[uint]string{}
	if len(chunkMap) > 0 {
		docIDs := make([]uint, 0, len(chunkMap))
		for _, ck := range chunkMap {
			docIDs = append(docIDs, ck.DocumentID)
		}
		var docs []models.KnowledgeDocument
		_ = store.DB.Where("id IN ?", docIDs).Find(&docs).Error
		for _, d := range docs {
			docMap[d.ID] = d.Name
		}
	}

	out := make([]gin.H, 0, len(hits))
	for _, hit := range hits {
		chunk, ok := chunkMap[hit.ID]
		content := strings.TrimSpace(hit.Content)
		if content == "" && ok {
			content = chunk.Content
		}
		docName := ""
		if ok {
			docName = docMap[chunk.DocumentID]
		}
		out = append(out, gin.H{
			"id":            hit.ID,
			"score":         hit.Score,
			"content":       content,
			"document_name": docName,
			"metadata":      hit.Metadata,
		})
		if ok {
			nextHitCount := chunk.HitCount + 1
			nextAvg := hit.Score
			if chunk.HitCount > 0 {
				nextAvg = (chunk.AvgScore*float64(chunk.HitCount) + hit.Score) / float64(nextHitCount)
			}
			_ = store.DB.Model(&models.KnowledgeChunk{}).Where("id = ?", chunk.ID).Updates(map[string]interface{}{
				"hit_count": nextHitCount,
				"avg_score": nextAvg,
			}).Error
		}
	}
	response.ResponseSuccess(c, gin.H{
		"query":   query,
		"results": out,
		"count":   len(out),
	})
}

// QATest 问答测试接口（使用 Eino 编排 + 向量检索）。
func (kc *KnowledgeWorkspaceController) QATest(c *gin.Context) {
	base, ok := kc.getBaseByID(c)
	if !ok {
		return
	}
	var req struct {
		Query string `json:"query" binding:"required"`
		TopK  int    `json:"top_k"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeKnowledgeQAInvalid)
		return
	}
	query := strings.TrimSpace(req.Query)
	if query == "" {
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeKnowledgeQAInvalid)
		return
	}
	qaCtx, cancel := knowledgeQAContextWithParent(c.Request.Context())
	defer cancel()

	hits, err := kc.searchBaseChunks(qaCtx, base, query, req.TopK)
	if err != nil {
		logger.Errorf("knowledge qa retrieve failed base_id=%d err=%v", base.ID, err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeKnowledgeQAFailed)
		return
	}

	answer, modelCfg, err := service.AnswerWithEnabledAPIModel(qaCtx, query, hits)
	if err != nil {
		if errors.Is(err, service.ErrNoEnabledAPIModel) {
			response.ResponseErrorWithMsg(c, http.StatusBadRequest, response.ErrCodeKnowledgeModelProfileNotFound, "no enabled api model, please configure one first")
			return
		}
		logger.Errorf("knowledge qa model infer failed base_id=%d provider=%s model=%s err=%v", base.ID, modelCfg.Provider, modelCfg.ModelName, err)
		response.ResponseErrorWithMsg(c, http.StatusInternalServerError, response.ErrCodeKnowledgeModelInferFailed, err.Error())
		return
	}
	response.ResponseSuccess(c, gin.H{
		"query":          query,
		"answer":         answer.Answer,
		"sources":        answer.Sources,
		"chunks":         answer.Chunks,
		"model_provider": modelCfg.Provider,
		"model_name":     modelCfg.ModelName,
		"model_id":       modelCfg.ID,
	})
}

// QATestStream 问答测试流式接口。
func (kc *KnowledgeWorkspaceController) QATestStream(c *gin.Context) {
	base, ok := kc.getBaseByID(c)
	if !ok {
		return
	}
	var req struct {
		Query string `json:"query" binding:"required"`
		TopK  int    `json:"top_k"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeKnowledgeQAInvalid)
		return
	}
	query := strings.TrimSpace(req.Query)
	if query == "" {
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeKnowledgeQAInvalid)
		return
	}
	qaCtx, cancel := knowledgeQAContextWithParent(c.Request.Context())
	defer cancel()

	hits, err := kc.searchBaseChunks(qaCtx, base, query, req.TopK)
	if err != nil {
		logger.Errorf("knowledge qa stream retrieve failed base_id=%d err=%v", base.ID, err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeKnowledgeQAFailed)
		return
	}

	c.Header("Content-Type", "text/event-stream; charset=utf-8")
	c.Header("Cache-Control", "no-cache, no-transform")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)
	c.Writer.Flush()

	if err := writeSSEEvent(c, "started", gin.H{
		"query":     query,
		"hit_count": len(hits),
	}); err != nil {
		logger.Warnf("knowledge qa stream started write failed base_id=%d err=%v", base.ID, err)
		return
	}

	streamWriteErr := error(nil)
	answer, modelCfg, err := service.AnswerWithEnabledAPIModelStreamWithSystemPrompt(qaCtx, query, hits, "", func(current string) {
		if streamWriteErr != nil {
			return
		}
		if err := writeSSEEvent(c, "delta", gin.H{
			"answer": current,
		}); err != nil {
			streamWriteErr = err
			cancel()
		}
	})
	if streamWriteErr != nil {
		logger.Warnf("knowledge qa stream interrupted base_id=%d err=%v", base.ID, streamWriteErr)
		return
	}
	if err != nil {
		if errors.Is(err, service.ErrNoEnabledAPIModel) {
			_ = writeSSEEvent(c, "error", gin.H{
				"code":    response.ErrCodeKnowledgeModelProfileNotFound,
				"message": "no enabled api model, please configure one first",
			})
			return
		}
		logger.Errorf("knowledge qa stream model infer failed base_id=%d provider=%s model=%s err=%v", base.ID, modelCfg.Provider, modelCfg.ModelName, err)
		_ = writeSSEEvent(c, "error", gin.H{
			"code":    response.ErrCodeKnowledgeModelInferFailed,
			"message": err.Error(),
		})
		return
	}

	_ = writeSSEEvent(c, "final", gin.H{
		"query":          query,
		"answer":         answer.Answer,
		"sources":        answer.Sources,
		"chunks":         answer.Chunks,
		"model_provider": modelCfg.Provider,
		"model_name":     modelCfg.ModelName,
		"model_id":       modelCfg.ID,
	})
}

// SaveFeedback 保存问答反馈。
func (kc *KnowledgeWorkspaceController) SaveFeedback(c *gin.Context) {
	base, ok := kc.getBaseByID(c)
	if !ok {
		return
	}
	var req struct {
		Question   string   `json:"question" binding:"required"`
		Answer     string   `json:"answer" binding:"required"`
		Helpful    *bool    `json:"helpful"`
		SourceDocs []string `json:"source_docs"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeKnowledgeFeedbackInvalid)
		return
	}
	question := strings.TrimSpace(req.Question)
	answer := strings.TrimSpace(req.Answer)
	if question == "" || answer == "" {
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeKnowledgeFeedbackInvalid)
		return
	}
	helpful := false
	if req.Helpful != nil {
		helpful = *req.Helpful
	}
	docsJSON, _ := json.Marshal(req.SourceDocs)
	row := models.KnowledgeQAFeedback{
		BaseID:     base.ID,
		AppID:      base.AppID,
		Question:   question,
		Answer:     answer,
		Helpful:    helpful,
		SourceDocs: string(docsJSON),
	}
	if err := store.DB.Create(&row).Error; err != nil {
		logger.Errorf("knowledge feedback save failed base_id=%d err=%v", base.ID, err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeKnowledgeFeedbackSaveFail)
		return
	}
	response.ResponseSuccess(c, gin.H{"item": row})
}

// ValidateConnectivity 检查向量存储可用性，便于部署排障。
func (kc *KnowledgeWorkspaceController) ValidateConnectivity(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 2500*time.Millisecond)
	defer cancel()
	health := service.GetVectorStoreHealth(ctx)
	response.ResponseSuccess(c, gin.H{
		"status":  health.Status,
		"ready":   health.Ready,
		"backend": health.Backend,
		"mode":    health.Mode,
		"message": health.Message,
	})
}
