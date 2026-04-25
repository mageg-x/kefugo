package utils

import (
	"encoding/base64"
	"strings"
)

const apiKeyEncPrefix = "enc:"

func EncryptAPIKey(key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}
	if strings.HasPrefix(key, apiKeyEncPrefix) {
		return key
	}
	encoded := base64.StdEncoding.EncodeToString([]byte(key))
	return apiKeyEncPrefix + encoded
}

func DecryptAPIKey(encrypted string) string {
	encrypted = strings.TrimSpace(encrypted)
	if encrypted == "" {
		return ""
	}
	if !strings.HasPrefix(encrypted, apiKeyEncPrefix) {
		return encrypted
	}
	encoded := strings.TrimPrefix(encrypted, apiKeyEncPrefix)
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return encrypted
	}
	return string(decoded)
}
