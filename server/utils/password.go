package utils

import "golang.org/x/crypto/bcrypt"

// HashPassword 使用 bcrypt 对明文密码做哈希。
func HashPassword(plain string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

// VerifyPassword 校验明文密码与存储哈希是否匹配。
func VerifyPassword(storedPassword string, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(storedPassword), []byte(plain)) == nil
}
