package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"kefu-server/models"
	"kefu-server/service/ai/rerank"
	"kefu-server/store"
	"kefu-server/utils"
	"kefu-server/utils/logger"
)

var ErrNoEnabledAPIModel = errors.New("enabled api model not found")

type ExternalChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ExternalLLMClient struct {
	cfg      models.APIModelConfig
	endpoint string
	client   *http.Client
}

func normalizeAPIProvider(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "openai":
		return "openai"
	case "qwen":
		return "qwen"
	case "deepseek":
		return "deepseek"
	case "zhipu":
		return "zhipu"
	case "ollama":
		return "ollama"
	case "cohere":
		return "cohere"
	case "jina":
		return "jina"
	default:
		return ""
	}
}

func defaultAPIBaseURL(provider string) string {
	switch normalizeAPIProvider(provider) {
	case "openai":
		return "https://api.openai.com/v1"
	case "qwen":
		return "https://dashscope.aliyuncs.com/compatible-mode/v1"
	case "deepseek":
		return "https://api.deepseek.com/v1"
	case "zhipu":
		return "https://open.bigmodel.cn/api/paas/v4"
	case "ollama":
		return "http://localhost:11434"
	case "cohere":
		return "https://api.cohere.com"
	case "jina":
		return "https://api.jina.ai/v1"
	default:
		return ""
	}
}

var malformedPortDotBeforeSlashPattern = regexp.MustCompile(`:(\d+)\./`)
var malformedPortDotAtEndPattern = regexp.MustCompile(`:(\d+)\.$`)

func sanitizeAPIBaseURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	raw = malformedPortDotBeforeSlashPattern.ReplaceAllString(raw, ":$1/")
	raw = malformedPortDotAtEndPattern.ReplaceAllString(raw, ":$1")
	raw = strings.TrimSpace(strings.TrimRight(raw, "/"))
	return raw
}

func normalizeAPIBaseURL(provider, baseURL string) string {
	baseURL = sanitizeAPIBaseURL(baseURL)
	if baseURL == "" {
		baseURL = defaultAPIBaseURL(provider)
	}
	baseURL = sanitizeAPIBaseURL(baseURL)
	return baseURL
}

func clampAPIModelConfig(cfg models.APIModelConfig) models.APIModelConfig {
	cfg.Provider = normalizeAPIProvider(cfg.Provider)
	cfg.BaseURL = normalizeAPIBaseURL(cfg.Provider, cfg.BaseURL)
	cfg.ModelName = strings.TrimSpace(cfg.ModelName)
	cfg.APIKey = strings.TrimSpace(cfg.APIKey)
	if cfg.TimeoutSec <= 0 {
		cfg.TimeoutSec = 60
	}
	if cfg.TimeoutSec > 600 {
		cfg.TimeoutSec = 600
	}
	if cfg.Temperature < 0 {
		cfg.Temperature = 0
	}
	if cfg.Temperature > 2 {
		cfg.Temperature = 2
	}
	if cfg.TopP <= 0 || cfg.TopP > 1 {
		cfg.TopP = 0.9
	}
	if cfg.MaxTokens <= 0 {
		cfg.MaxTokens = 512
	}
	if cfg.MaxTokens > 8192 {
		cfg.MaxTokens = 8192
	}
	if cfg.Status != 1 {
		cfg.Status = 0
	}
	return cfg
}

func NewExternalLLMClient(cfg models.APIModelConfig) (*ExternalLLMClient, error) {
	cfg = clampAPIModelConfig(cfg)
	if cfg.Provider == "" {
		return nil, fmt.Errorf("api model provider invalid")
	}
	if cfg.ModelName == "" {
		return nil, fmt.Errorf("api model name is empty")
	}
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("api base_url is empty")
	}
	parsedURL, err := url.Parse(cfg.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("api base_url invalid: %w", err)
	}
	if !parsedURL.IsAbs() || strings.TrimSpace(parsedURL.Host) == "" {
		return nil, fmt.Errorf("api base_url invalid: absolute url required")
	}
	return &ExternalLLMClient{
		cfg:      cfg,
		endpoint: cfg.BaseURL + "/chat/completions",
		client: &http.Client{
			Timeout: time.Duration(cfg.TimeoutSec) * time.Second,
		},
	}, nil
}

type externalChatCompletionRequest struct {
	Model       string                `json:"model"`
	Messages    []ExternalChatMessage `json:"messages"`
	Temperature float64               `json:"temperature,omitempty"`
	TopP        float64               `json:"top_p,omitempty"`
	MaxTokens   int                   `json:"max_tokens,omitempty"`
	Stream      bool                  `json:"stream"`
}

type externalChatCompletionResponse struct {
	Choices []struct {
		Message struct {
			Content interface{} `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

type externalChatCompletionStreamResponse struct {
	Choices []struct {
		Delta struct {
			Content interface{} `json:"content"`
		} `json:"delta"`
		Message struct {
			Content interface{} `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func parseExternalContent(raw interface{}) string {
	switch v := raw.(type) {
	case string:
		return strings.TrimSpace(v)
	case []interface{}:
		parts := make([]string, 0, len(v))
		for _, item := range v {
			obj, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			text, _ := obj["text"].(string)
			if strings.TrimSpace(text) != "" {
				parts = append(parts, strings.TrimSpace(text))
			}
		}
		return strings.TrimSpace(strings.Join(parts, "\n"))
	default:
		return ""
	}
}

func (c *ExternalLLMClient) Generate(ctx context.Context, messages []ExternalChatMessage) (string, error) {
	if c == nil {
		return "", fmt.Errorf("external llm client is nil")
	}
	if len(messages) == 0 {
		return "", fmt.Errorf("messages empty")
	}
	reqBody := externalChatCompletionRequest{
		Model:       c.cfg.ModelName,
		Messages:    messages,
		Temperature: c.cfg.Temperature,
		TopP:        c.cfg.TopP,
		MaxTokens:   c.cfg.MaxTokens,
		Stream:      false,
	}
	raw, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey := strings.TrimSpace(c.cfg.APIKey); apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var out externalChatCompletionResponse
	if err := json.Unmarshal(body, &out); err != nil {
		if resp.StatusCode >= 400 {
			return "", fmt.Errorf("llm http %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
		}
		return "", err
	}
	if resp.StatusCode >= 400 {
		msg := ""
		if out.Error != nil {
			msg = strings.TrimSpace(out.Error.Message)
		}
		if msg == "" {
			msg = strings.TrimSpace(string(body))
		}
		if msg == "" {
			msg = resp.Status
		}
		return "", fmt.Errorf("llm http %d: %s", resp.StatusCode, msg)
	}
	if out.Error != nil && strings.TrimSpace(out.Error.Message) != "" {
		return "", fmt.Errorf("%s", strings.TrimSpace(out.Error.Message))
	}
	if len(out.Choices) == 0 {
		return "", fmt.Errorf("llm empty choices")
	}
	text := parseExternalContent(out.Choices[0].Message.Content)
	if text == "" {
		return "", fmt.Errorf("llm empty content")
	}
	return text, nil
}

func (c *ExternalLLMClient) GenerateStream(
	ctx context.Context,
	messages []ExternalChatMessage,
	onDelta func(current string),
) (string, error) {
	if c == nil {
		return "", fmt.Errorf("external llm client is nil")
	}
	if len(messages) == 0 {
		return "", fmt.Errorf("messages empty")
	}
	reqBody := externalChatCompletionRequest{
		Model:       c.cfg.ModelName,
		Messages:    messages,
		Temperature: c.cfg.Temperature,
		TopP:        c.cfg.TopP,
		MaxTokens:   c.cfg.MaxTokens,
		Stream:      true,
	}
	raw, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey := strings.TrimSpace(c.cfg.APIKey); apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	req.Header.Set("Accept", "text/event-stream")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		text := strings.TrimSpace(string(body))
		if text == "" {
			text = resp.Status
		}
		return "", fmt.Errorf("llm http %d: %s", resp.StatusCode, text)
	}

	scanner := bufio.NewScanner(resp.Body)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 2*1024*1024)

	var merged strings.Builder
	emit := func(part string) {
		part = strings.TrimSpace(part)
		if part == "" {
			return
		}
		merged.WriteString(part)
		if onDelta != nil {
			onDelta(merged.String())
		}
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}
		if strings.HasPrefix(line, "data:") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		}
		if line == "" {
			continue
		}
		if line == "[DONE]" {
			break
		}
		var out externalChatCompletionStreamResponse
		if err := json.Unmarshal([]byte(line), &out); err != nil {
			continue
		}
		if out.Error != nil && strings.TrimSpace(out.Error.Message) != "" {
			return "", fmt.Errorf("%s", strings.TrimSpace(out.Error.Message))
		}
		if len(out.Choices) == 0 {
			continue
		}
		part := parseExternalContent(out.Choices[0].Delta.Content)
		if part == "" {
			part = parseExternalContent(out.Choices[0].Message.Content)
		}
		emit(part)
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	text := strings.TrimSpace(merged.String())
	if text == "" {
		return "", fmt.Errorf("llm empty content")
	}
	return text, nil
}

func GetEnabledAPIModelConfig() (models.APIModelConfig, error) {
	return GetEnabledAPIModelConfigByType(string(models.AIModelTypeChat))
}

func GetEnabledAPIModelConfigByType(modelType string) (models.APIModelConfig, error) {
	if store.DB == nil {
		return models.APIModelConfig{}, fmt.Errorf("database not initialized")
	}
	var item models.APIModelConfig
	query := store.DB.Where("status = ?", 1)
	if modelType != "" {
		query = query.Where("model_type = ?", modelType)
	}
	if err := query.Order("is_default DESC, updated_at DESC").First(&item).Error; err != nil {
		return models.APIModelConfig{}, ErrNoEnabledAPIModel
	}
	item = clampAPIModelConfig(item)
	if item.Provider == "" || item.ModelName == "" {
		return models.APIModelConfig{}, fmt.Errorf("enabled api model invalid")
	}
	item.APIKey = utils.DecryptAPIKey(item.APIKey)
	return item, nil
}

func RerankHits(ctx context.Context, query string, hits []VectorHit, topN int) []VectorHit {
	cfg, err := GetEnabledAPIModelConfigByType(string(models.AIModelTypeRerank))
	if err != nil {
		return hits
	}
	provider, err := rerank.NewProvider(&cfg)
	if err != nil {
		return hits
	}
	if cfg.TopN > 0 {
		topN = cfg.TopN
	}
	if topN <= 0 {
		topN = 5
	}
	candidates := make([]rerank.Candidate, len(hits))
	for i, h := range hits {
		candidates[i] = rerank.Candidate{
			ID:      h.ID,
			Content: h.Content,
			Score:   h.Score,
		}
	}
	reranked, err := provider.Rerank(query, candidates, topN)
	if err != nil {
		logger.Warnf("rerank failed, using original order: %v", err)
		return hits
	}
	out := make([]VectorHit, 0, len(reranked))
	for _, r := range reranked {
		for _, h := range hits {
			if h.ID == r.ID {
				h.Score = r.Score
				out = append(out, h)
				break
			}
		}
	}
	if len(out) == 0 {
		return hits
	}
	return out
}

const (
	maxRAGSnippetChars  = 220
	maxRAGTotalCtxChars = 1000
	maxRAGSnippets      = 4
)

func clampRunes(raw string, limit int) string {
	if limit <= 0 {
		return ""
	}
	rs := []rune(strings.TrimSpace(raw))
	if len(rs) <= limit {
		return string(rs)
	}
	return strings.TrimSpace(string(rs[:limit])) + "..."
}

func buildRAGPrompt(query string, hits []VectorHit) string {
	parts := make([]string, 0, len(hits)+10)
	parts = append(parts,
		"你是客服机器人助手。",
		"仅基于提供的知识片段作答，不要编造事实。",
		"先给结论，再给最多 4 条关键要点。",
		"尽量简洁，避免重复铺垫；证据不足就直接说明信息不足。",
		"",
		"知识片段：",
	)
	totalCtxChars := 0
	used := 0
	for idx, hit := range hits {
		if used >= maxRAGSnippets || totalCtxChars >= maxRAGTotalCtxChars {
			break
		}
		content := strings.TrimSpace(hit.Content)
		if content == "" {
			continue
		}
		content = clampRunes(content, maxRAGSnippetChars)
		nextLen := len([]rune(content))
		if nextLen <= 0 {
			continue
		}
		if totalCtxChars+nextLen > maxRAGTotalCtxChars {
			left := maxRAGTotalCtxChars - totalCtxChars
			if left <= 32 {
				break
			}
			content = clampRunes(content, left)
			nextLen = len([]rune(content))
		}
		docName := "unknown_doc"
		if hit.Metadata != nil {
			if v, ok := hit.Metadata["doc_name"].(string); ok && strings.TrimSpace(v) != "" {
				docName = strings.TrimSpace(v)
			}
		}
		parts = append(parts, fmt.Sprintf("[%d][%s] %s", idx+1, docName, content))
		totalCtxChars += nextLen
		used++
	}
	parts = append(parts, "", "用户问题：", query, "", "请回答：")
	return strings.Join(parts, "\n")
}

func buildRAGSources(hits []VectorHit) []map[string]string {
	sources := make([]map[string]string, 0, len(hits))
	for _, hit := range hits {
		docName := ""
		if hit.Metadata != nil {
			if name, ok := hit.Metadata["doc_name"].(string); ok {
				docName = strings.TrimSpace(name)
			}
		}
		sources = append(sources, map[string]string{
			"doc_name":  docName,
			"vector_id": strings.TrimSpace(hit.ID),
		})
	}
	return sources
}

type RAGAnswer struct {
	Answer  string              `json:"answer"`
	Sources []map[string]string `json:"sources"`
	Chunks  []VectorHit         `json:"chunks"`
}

func AnswerWithAPIModelWithSystemPrompt(
	ctx context.Context,
	cfg models.APIModelConfig,
	query string,
	hits []VectorHit,
	systemPrompt string,
) (*RAGAnswer, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("empty query")
	}
	systemPrompt = strings.TrimSpace(systemPrompt)
	if systemPrompt == "" {
		systemPrompt = "你是一个专业客服助手，请严格基于知识片段回答问题。"
	}
	client, err := NewExternalLLMClient(cfg)
	if err != nil {
		return nil, err
	}
	prompt := buildRAGPrompt(query, hits)
	answer, err := client.Generate(ctx, []ExternalChatMessage{
		{
			Role:    "system",
			Content: systemPrompt,
		},
		{
			Role:    "user",
			Content: prompt,
		},
	})
	if err != nil {
		return nil, err
	}
	answer = strings.TrimSpace(answer)
	if answer == "" {
		answer = "我不知道。"
	}
	return &RAGAnswer{
		Answer:  answer,
		Sources: buildRAGSources(hits),
		Chunks:  hits,
	}, nil
}

func AnswerWithAPIModel(
	ctx context.Context,
	cfg models.APIModelConfig,
	query string,
	hits []VectorHit,
) (*RAGAnswer, error) {
	return AnswerWithAPIModelWithSystemPrompt(ctx, cfg, query, hits, "")
}

func AnswerWithEnabledAPIModel(ctx context.Context, query string, hits []VectorHit) (*RAGAnswer, models.APIModelConfig, error) {
	cfg, err := GetEnabledAPIModelConfig()
	if err != nil {
		return nil, models.APIModelConfig{}, err
	}
	answer, err := AnswerWithAPIModel(ctx, cfg, query, hits)
	if err != nil {
		return nil, models.APIModelConfig{}, err
	}
	return answer, cfg, nil
}

func AnswerWithEnabledAPIModelWithSystemPrompt(ctx context.Context, query string, hits []VectorHit, systemPrompt string) (*RAGAnswer, models.APIModelConfig, error) {
	cfg, err := GetEnabledAPIModelConfig()
	if err != nil {
		return nil, models.APIModelConfig{}, err
	}
	answer, err := AnswerWithAPIModelWithSystemPrompt(ctx, cfg, query, hits, systemPrompt)
	if err != nil {
		return nil, models.APIModelConfig{}, err
	}
	return answer, cfg, nil
}

func AnswerWithAPIModelStreamWithSystemPrompt(
	ctx context.Context,
	cfg models.APIModelConfig,
	query string,
	hits []VectorHit,
	systemPrompt string,
	onDelta func(current string),
) (*RAGAnswer, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("empty query")
	}
	systemPrompt = strings.TrimSpace(systemPrompt)
	if systemPrompt == "" {
		systemPrompt = "你是一个专业客服助手，请严格基于知识片段回答问题。"
	}
	client, err := NewExternalLLMClient(cfg)
	if err != nil {
		return nil, err
	}
	prompt := buildRAGPrompt(query, hits)
	answer, err := client.GenerateStream(ctx, []ExternalChatMessage{
		{
			Role:    "system",
			Content: systemPrompt,
		},
		{
			Role:    "user",
			Content: prompt,
		},
	}, onDelta)
	if err != nil {
		return nil, err
	}
	answer = strings.TrimSpace(answer)
	if answer == "" {
		answer = "我不知道。"
	}
	return &RAGAnswer{
		Answer:  answer,
		Sources: buildRAGSources(hits),
		Chunks:  hits,
	}, nil
}

func AnswerWithEnabledAPIModelStreamWithSystemPrompt(
	ctx context.Context,
	query string,
	hits []VectorHit,
	systemPrompt string,
	onDelta func(current string),
) (*RAGAnswer, models.APIModelConfig, error) {
	cfg, err := GetEnabledAPIModelConfig()
	if err != nil {
		return nil, models.APIModelConfig{}, err
	}
	answer, err := AnswerWithAPIModelStreamWithSystemPrompt(ctx, cfg, query, hits, systemPrompt, onDelta)
	if err != nil {
		return nil, models.APIModelConfig{}, err
	}
	return answer, cfg, nil
}

func TestAPIModelConnection(ctx context.Context, cfg models.APIModelConfig) (string, int64, error) {
	cfg = clampAPIModelConfig(cfg)
	client, err := NewExternalLLMClient(cfg)
	if err != nil {
		return "", 0, err
	}
	startAt := time.Now()
	text, err := client.Generate(ctx, []ExternalChatMessage{
		{
			Role:    "system",
			Content: "你是一个健康检查助手。",
		},
		{
			Role:    "user",
			Content: "请回复：ok",
		},
	})
	if err != nil {
		return "", time.Since(startAt).Milliseconds(), err
	}
	return text, time.Since(startAt).Milliseconds(), nil
}
