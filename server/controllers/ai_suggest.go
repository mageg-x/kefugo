package controllers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"kefu-server/models"
	"kefu-server/service"
	"kefu-server/store"
	"kefu-server/utils/logger"
	"kefu-server/utils/response"
)

const (
	aiSuggestSourceVectorEino    = "rag-vector-eino"
	aiSuggestSourceLocalFallback = "rag-local-fallback"
	aiSuggestSourceRuleFallback  = "rule-fallback"
)

// AISuggestController 负责 AI 智能回复建议功能
// 提供基于知识库检索和上下文理解的智能回复推荐

type AISuggestController struct{}

// SuggestRequest AI建议请求结构
type SuggestRequest struct {
	SID   string `json:"sid"`    // 会话ID
	AppID string `json:"app_id"` // 应用ID
	Query string `json:"query"`  // 查询内容（可选，优先级高于会话上下文）
}

// Suggest 返回基于会话上下文 + 知识库检索的回复建议
// HTTP POST /api/v1/ai/suggest
// 流程：1.获取会话历史消息 2.检索知识库相关片段 3.使用AI生成建议回复
func (ac *AISuggestController) Suggest(c *gin.Context) {
	var req SuggestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Errorf("ai suggest request params invalid err=%v", err)
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeInvalidParams)
		return
	}

	// 解析并验证请求参数
	sid := strings.TrimSpace(req.SID)
	appID := strings.TrimSpace(req.AppID)
	if sid == "" && appID == "" {
		logger.Errorf("ai suggest sid and app_id both empty")
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeSessionIDRequired)
		return
	}

	// 获取会话历史消息，提取访客最新问题
	ms := getSessionMessages(c, sid, 20)
	visitorText := ""
	for i := len(ms) - 1; i >= 0; i-- {
		m := ms[i]
		if m == nil || strings.TrimSpace(m.Content) == "" {
			continue
		}
		bizType := deduceMessageBusinessType(m)
		if bizType == models.WSMessageTypeVisitor {
			visitorText = strings.TrimSpace(m.Content)
			break
		}
	}

	// 如果请求中直接提供了query，使用query覆盖
	if strings.TrimSpace(req.Query) != "" {
		visitorText = strings.TrimSpace(req.Query)
	}

	// 从sid中提取app_id（如果未提供）
	if appID == "" {
		if parts := strings.Split(sid, ":"); len(parts) == 4 {
			appID = strings.TrimSpace(parts[2])
		}
	}

	// 获取客服的AI配置（模型、风格、提示词）
	style := defaultAIStyle
	model := defaultAIModel
	prompt := defaultAIPrompt
	source := aiSuggestSourceLocalFallback
	userName, _ := getAuthUser(c)
	if strings.TrimSpace(userName) != "" && store.DB != nil {
		item := &models.AgentSetting{}
		if err := store.DB.Where("user_name = ?", userName).First(item).Error; err == nil {
			model = normalizeAIModel(item.AIModel)
			style = normalizeAIStyle(item.AIStyle)
			prompt = normalizeAIPrompt(item.AIPrompt)
		}
	}

	// 准备返回数据
	metaChunks := make([]gin.H, 0, 8)
	suggestion := ""
	chunkCount := 0

	// 查找应用关联的知识库
	var base models.KnowledgeBase
	baseFound := false
	if store.DB != nil && appID != "" {
		if err := store.DB.Where("app_id = ?", appID).Order("updated_at DESC").First(&base).Error; err == nil {
			baseFound = true
		}
	}

	// 优先使用向量检索 + 外部 API 模型生成回复建议
	if baseFound {
		vdb := service.GetVectorStore()

		// 检索相关知识片段
		hits, searchErr := vdb.SearchText(c.Request.Context(), base.Collection, visitorText, 5)
		if searchErr == nil && len(hits) > 0 {
			hits = service.RerankHits(c.Request.Context(), visitorText, hits, 5)
			answer, _, answerErr := service.AnswerWithEnabledAPIModel(c.Request.Context(), visitorText, hits)
			if answerErr == nil && answer != nil {
				suggestion = strings.TrimSpace(answer.Answer)
				source = aiSuggestSourceVectorEino
				chunkCount = len(answer.Chunks)
				for idx, item := range answer.Chunks {
					metaChunks = append(metaChunks, gin.H{
						"rank":        idx + 1,
						"title":       "",
						"content":     item.Content,
						"score":       item.Score,
						"source_type": "vector",
						"source_name": base.Name,
					})
				}
			} else {
				if !errors.Is(answerErr, service.ErrNoEnabledAPIModel) {
					logger.Errorf("ai suggest model answer failed app_id=%s err=%v", appID, answerErr)
				}
			}
		} else {
			logger.Errorf("ai suggest vector search failed app_id=%s err=%v", appID, searchErr)
		}
	}

	// 如果AI生成失败或无结果，使用本地RAG生成建议
	if strings.TrimSpace(suggestion) == "" {
		chunks := queryKnowledgeChunks(appID, visitorText, 5)
		suggestion = composeAISuggestion(style, prompt, visitorText, chunks)
		source = aiSuggestSourceLocalFallback
		chunkCount = len(chunks)
		for idx, item := range chunks {
			metaChunks = append(metaChunks, gin.H{
				"rank":        idx + 1,
				"title":       item.Title,
				"content":     item.Content,
				"score":       item.Score,
				"source_type": item.SourceType,
				"source_name": item.SourceName,
			})
		}
	}

	// 兜底回复
	if strings.TrimSpace(suggestion) == "" {
		source = aiSuggestSourceRuleFallback
		suggestion = "您好，已收到您的问题，我这边马上为您核实并回复您。"
	}
	logger.Infof("ai suggest generated sid=%s app_id=%s chunks=%s", sid, appID, strconv.Itoa(chunkCount))

	response.ResponseSuccess(c, gin.H{
		"suggestion": suggestion,
		"model":      model,
		"style":      style,
		"source":     source,
		"context": gin.H{
			"query":      visitorText,
			"rag_chunks": metaChunks,
		},
	})
}

// getSessionMessages 获取会话最近N条消息
// 用于AI建议时提供上下文理解
func getSessionMessages(c *gin.Context, sid string, limit int) []*models.Message {
	if limit <= 0 {
		limit = 10
	}
	ms := make([]*models.Message, 0, limit)
	messageService := service.GetMsgService()
	if messageService == nil {
		logger.Errorf("get session messages message service unavailable sid=%s", sid)
		return ms
	}
	rows, err := messageService.GetMessagesBySession(sid, limit)
	if err != nil {
		logger.Errorf("get session messages failed sid=%s err=%v", sid, err)
		return ms
	}
	return rows
}
