package utils

import (
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"kefu-server/config"
	"kefu-server/utils/logger"
)

type Claims struct {
	UserID   uint   `json:"user_id"`
	UserName string `json:"user_name"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

const defaultJWTSecret = "kefu-default-jwt-secret-change-in-production"

// getJWTSecret 获取 JWT 签名密钥（优先运行配置，其次默认值）。
func getJWTSecret() string {
	cfg := config.GetConfig()
	if cfg != nil && strings.TrimSpace(cfg.Security.JWTSecret) != "" {
		return strings.TrimSpace(cfg.Security.JWTSecret)
	}
	return defaultJWTSecret
}

// GenerateToken 生成登录令牌。
func GenerateToken(userID uint, userName, role string) (string, error) {
	secret := getJWTSecret()
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		UserID:   userID,
		UserName: userName,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ParseToken 解析并校验 JWT。
func ParseToken(tokenStr string) (*Claims, error) {
	secret := getJWTSecret()
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		logger.Errorf("jwt parse failed: %v", err)
		return nil, err
	}
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	logger.Errorf("jwt claims invalid")
	return nil, err
}

// IsTokenExpired 判断错误是否由 token 过期引起。
func IsTokenExpired(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, jwt.ErrTokenExpired)
}
