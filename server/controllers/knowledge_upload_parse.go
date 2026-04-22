package controllers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"kefu-server/config"
)

func extractKnowledgeContentFromURL(fileURL string, name string) (content string, sourceType string, sourceName string, err error) {
	cfg := config.GetConfig()
	uploadDir := ""
	if cfg != nil {
		uploadDir = strings.TrimSpace(cfg.Admin.UploadDir)
	}
	if uploadDir == "" {
		uploadDir = "data/uploads"
	}
	if !strings.HasPrefix(fileURL, "/uploads/") {
		return "", "", "", fmt.Errorf("unsupported file url")
	}
	fileName := strings.TrimPrefix(fileURL, "/uploads/")
	fileName = filepath.Base(fileName)
	if strings.TrimSpace(fileName) == "" {
		return "", "", "", fmt.Errorf("invalid file url")
	}
	fullPath := filepath.Join(uploadDir, fileName)
	body, readErr := os.ReadFile(fullPath)
	if readErr != nil {
		return "", "", "", fmt.Errorf("read upload file failed")
	}

	ext := strings.ToLower(strings.TrimSpace(filepath.Ext(fileName)))
	sourceType = "upload"
	sourceName = strings.TrimSpace(name)
	if sourceName == "" {
		sourceName = fileName
	}

	switch ext {
	case ".txt", ".md", ".csv", ".json", ".log", ".html", ".htm":
		t := string(body)
		if !utf8.ValidString(t) {
			return "", "", "", fmt.Errorf("file encoding is not utf-8")
		}
		if ext == ".html" || ext == ".htm" {
			t = stripHTML(t)
		}
		content = strings.TrimSpace(t)
	default:
		// 非文本先按 utf8 尝试，失败则提示不支持。
		t := string(body)
		if !utf8.ValidString(t) {
			return "", "", "", fmt.Errorf("file type not supported for text parsing")
		}
		content = strings.TrimSpace(t)
	}
	if len([]rune(content)) > 30000 {
		content = string([]rune(content)[:30000])
	}
	return content, sourceType, sourceName, nil
}
