package controllers

import "strings"

type ragChunk struct {
	Title      string `json:"title"`
	Content    string `json:"content"`
	Tags       string `json:"tags"`
	SourceType string `json:"source_type"`
	SourceName string `json:"source_name"`
	Score      int    `json:"score"`
}

func composeAISuggestion(style string, prompt string, visitorQuestion string, chunks []ragChunk) string {
	style = strings.TrimSpace(strings.ToLower(style))
	q := strings.TrimSpace(visitorQuestion)
	if q == "" {
		q = "请描述一下您的具体问题，我会给出可执行的解决建议。"
	}

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

	if p := strings.TrimSpace(prompt); p != "" {
		lines = append(lines, "")
		lines = append(lines, "[执行约束] "+p)
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}
