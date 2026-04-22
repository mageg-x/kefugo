package utils

import (
	"math/rand"
	"net/url"
	"strconv"
	"time"
)

// GenerateTimestamp 生成时间戳字符串
func GenerateTimestamp() string {
	return strconv.FormatInt(time.Now().Unix(), 10)
}

// GenerateRandomString 生成指定长度的随机字符串
func GenerateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// ExtractDomain 从 URL 中提取域名
func ExtractDomain(urlStr string) string {
	if urlStr == "" {
		return ""
	}

	// 解析 URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return ""
	}

	return parsedURL.Hostname()
}
