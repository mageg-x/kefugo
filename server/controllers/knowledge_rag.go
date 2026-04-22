package controllers

import (
	"regexp"
	"sort"
	"strings"
	"unicode"

	"kefu-server/models"
	"kefu-server/store"
	"kefu-server/utils/logger"
)

// ragChunk RAG检索结果片段结构
type ragChunk struct {
	Title      string `json:"title"`       // 文档标题
	Content    string `json:"content"`     // 片段内容
	Tags       string `json:"tags"`        // 标签
	SourceType string `json:"source_type"` // 来源类型：manual/faq/vector
	SourceName string `json:"source_name"` // 来源名称
	Score      int    `json:"score"`       // 相关性评分
}

// stripHTMLRegex 用于去除HTML标签的正则表达式
var stripHTMLRegex = regexp.MustCompile(`<[^>]+>`)

// normalizeRAGText 对文本进行标准化处理：小写、去多余空白、去除特殊字符
// 用于提高RAG检索的匹配精度
func normalizeRAGText(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return ""
	}
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\t", " ")
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	return strings.TrimSpace(s)
}

// stripHTML 去除文本中的HTML标签
// 用于处理HTML格式的文档内容
func stripHTML(s string) string {
	if strings.TrimSpace(s) == "" {
		return ""
	}
	return strings.TrimSpace(stripHTMLRegex.ReplaceAllString(s, " "))
}

// tokenizeQuery 将查询文本分词为检索词列表
// 支持中英文混合分词，过滤短词和标点符号
// 返回的词列表已去重
func tokenizeQuery(q string) []string {
	n := normalizeRAGText(q)
	if n == "" {
		return nil
	}
	// 使用unicode分割处理中英文标点
	parts := strings.FieldsFunc(n, func(r rune) bool {
		if unicode.IsSpace(r) {
			return true
		}
		switch r {
		case ',', '.', ';', ':', '?', '!', '，', '。', '；', '：', '？', '！', '、', '|', '/', '\\', '-', '_', '+', '(', ')', '[', ']', '{', '}', '"', '\'', '`':
			return true
		default:
			return false
		}
	})
	seen := make(map[string]struct{}, len(parts)+1)
	out := make([]string, 0, len(parts)+1)
	// 保留完整查询作为检索词（长度>=2）
	if len([]rune(n)) >= 2 {
		seen[n] = struct{}{}
		out = append(out, n)
	}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if len([]rune(p)) < 2 {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	return out
}

// chunkByRune 按字符数将长文本切分为重叠的片段
// chunkSize: 每个片段的字符数（默认280）
// overlap: 相邻片段的重叠字符数（默认40）
// 用于将文档切分为适合检索的小块
func chunkByRune(text string, chunkSize int, overlap int) []string {
	t := strings.TrimSpace(text)
	if t == "" {
		return nil
	}
	if chunkSize <= 0 {
		chunkSize = 280
	}
	if overlap < 0 {
		overlap = 0
	}
	runes := []rune(t)
	if len(runes) <= chunkSize {
		return []string{t}
	}
	step := chunkSize - overlap
	if step <= 0 {
		step = chunkSize
	}
	out := make([]string, 0, (len(runes)/step)+1)
	for i := 0; i < len(runes); i += step {
		end := i + chunkSize
		if end > len(runes) {
			end = len(runes)
		}
		chunk := strings.TrimSpace(string(runes[i:end]))
		if chunk != "" {
			out = append(out, chunk)
		}
		if end == len(runes) {
			break
		}
	}
	return out
}

// scoreChunk 计算文档片段与查询词的相关性评分
// 评分规则：标题匹配+10分，标签匹配+6分，正文匹配+4分，首词匹配额外+12分
func scoreChunk(tokens []string, title string, body string, tags string) int {
	if len(tokens) == 0 {
		return 0
	}
	titleN := normalizeRAGText(title)
	bodyN := normalizeRAGText(body)
	tagsN := normalizeRAGText(tags)
	if bodyN == "" {
		return 0
	}
	score := 0
	for _, tk := range tokens {
		if tk == "" {
			continue
		}
		if strings.Contains(titleN, tk) {
			score += 10
		}
		if strings.Contains(tagsN, tk) {
			score += 6
		}
		if strings.Contains(bodyN, tk) {
			score += 4
		}
	}
	query := ""
	if len(tokens) > 0 {
		query = tokens[0]
	}
	if query != "" && strings.Contains(bodyN, query) {
		score += 12
	}
	return score
}

// queryKnowledgeChunks 基于关键词检索知识库相关文章和FAQ片段
// appID: 应用ID
// query: 查询文本
// topK: 返回的最大结果数（默认5，最大12）
// 返回与查询相关的文档片段列表，按相关性评分降序排列
func queryKnowledgeChunks(appID string, query string, topK int) []ragChunk {
	appID = strings.TrimSpace(appID)
	if appID == "" || strings.TrimSpace(query) == "" || store.DB == nil {
		logger.Warnf("query knowledge chunks invalid params app_id=%s query=%s", appID, query)
		return nil
	}
	if topK <= 0 {
		topK = 5
	}
	if topK > 12 {
		topK = 12
	}
	tokens := tokenizeQuery(query)
	if len(tokens) == 0 {
		logger.Warnf("query knowledge chunks tokenize empty app_id=%s query=%s", appID, query)
		return nil
	}

	// 检索知识库文章
	var docs []models.KnowledgeArticle
	if err := store.DB.Where("app_id = ? AND enabled = ?", appID, true).Order("updated_at DESC").Find(&docs).Error; err != nil {
		logger.Errorf("query knowledge chunks list docs failed app_id=%s err=%v", appID, err)
		return nil
	}

	// 对文章内容进行分块和评分
	candidates := make([]ragChunk, 0, len(docs)*2)
	for _, doc := range docs {
		body := strings.TrimSpace(doc.Content)
		if body == "" {
			continue
		}
		chunks := chunkByRune(body, 300, 40)
		if len(chunks) == 0 {
			continue
		}
		sourceType := strings.TrimSpace(doc.SourceType)
		if sourceType == "" {
			sourceType = "manual"
		}
		sourceName := strings.TrimSpace(doc.SourceName)
		if sourceName == "" {
			sourceName = doc.Title
		}
		for _, chunk := range chunks {
			score := scoreChunk(tokens, doc.Title, chunk, doc.Tags)
			if score <= 0 {
				continue
			}
			candidates = append(candidates, ragChunk{
				Title:      doc.Title,
				Content:    chunk,
				Tags:       doc.Tags,
				SourceType: sourceType,
				SourceName: sourceName,
				Score:      score,
			})
		}
	}

	// 检索FAQ
	var faqs []models.FAQItem
	if err := store.DB.Where("app_id = ? AND enabled = ?", appID, true).Order("updated_at DESC").Find(&faqs).Error; err == nil {
		for _, faq := range faqs {
			content := strings.TrimSpace(faq.Question + "\n" + faq.Answer)
			score := scoreChunk(tokens, faq.Question, content, faq.Category)
			if score <= 0 {
				continue
			}
			candidates = append(candidates, ragChunk{
				Title:      faq.Question,
				Content:    strings.TrimSpace(faq.Answer),
				Tags:       faq.Category,
				SourceType: "faq",
				SourceName: "faq",
				Score:      score,
			})
		}
	}

	if len(candidates) == 0 {
		logger.Infof("query knowledge chunks no candidates app_id=%s query=%s", appID, query)
		return nil
	}

	// 按评分降序排序，评分相同时内容长的优先
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Score == candidates[j].Score {
			return len([]rune(candidates[i].Content)) > len([]rune(candidates[j].Content))
		}
		return candidates[i].Score > candidates[j].Score
	})
	if len(candidates) > topK {
		candidates = candidates[:topK]
	}
	return candidates
}

// composeAISuggestion 根据检索到的知识片段组合成AI回复建议
// style: 回复风格（professional/friendly/formal）
// prompt: 系统提示词约束
// visitorQuestion: 访客问题
// chunks: 检索到的相关片段
// 返回组合后的建议回复文本
func composeAISuggestion(style string, prompt string, visitorQuestion string, chunks []ragChunk) string {
	style = strings.TrimSpace(strings.ToLower(style))
	q := strings.TrimSpace(visitorQuestion)
	if q == "" {
		q = "请描述一下您的具体问题，我会给出可执行的解决建议。"
	}

	// 根据风格选择开场白
	prefix := "已收到您的问题。"
	switch style {
	case "friendly":
		prefix = "收到啦，我来帮你快速处理这个问题。"
	case "formal":
		prefix = "您好，您的问题已收到，现为您提供处理建议。"
	}

	lines := []string{prefix}
	if len(chunks) > 0 {
		lines = append(lines, "根据当前知识库，建议如下：")
		for idx, item := range chunks {
			if idx >= 3 {
				break
			}
			content := strings.TrimSpace(item.Content)
			if len([]rune(content)) > 90 {
				content = string([]rune(content)[:90]) + "..."
			}
			lines = append(lines, "- "+content)
		}
		lines = append(lines, "如需我继续处理，我可以直接按上述方案为您推进。")
	} else {
		lines = append(lines,
			"为了更快定位，请补充：订单号/账号、发生时间、报错截图（如有）。",
			"收到后我会给出明确处理步骤和预计时效。",
		)
	}

	// 添加系统约束提示
	if p := strings.TrimSpace(prompt); p != "" {
		lines = append(lines, "")
		lines = append(lines, "[执行约束] "+p)
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}
