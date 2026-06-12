// Package jwt 提供 JWT 令牌生成、解析和刷新工具。
//
// 使用 golang-jwt/v5 库实现，支持访问令牌和刷新令牌。
// Claims 包含 UserID、Username、Roles，与 TECH.md §9.2 对齐。
package jwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims JWT 声明
type Claims struct {
	UserID      int64    `json:"user_id"`
	Username    string   `json:"username"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"` // 从 Role.Permissions 解析，避免中间件硬编码
	TokenType   string   `json:"token_type"`  // "access" 或 "refresh"，用于区分令牌类型
	jwt.RegisteredClaims
}

// GenerateAccessToken 生成访问令牌
func GenerateAccessToken(userID int64, username string, roles []string, permissions []string, secret string, expire time.Duration) (string, error) {
	return generateToken(userID, username, roles, permissions, "access", secret, expire)
}

// GenerateRefreshToken 生成刷新令牌
func GenerateRefreshToken(userID int64, username string, roles []string, permissions []string, secret string, expire time.Duration) (string, error) {
	return generateToken(userID, username, roles, permissions, "refresh", secret, expire)
}

// ParseToken 解析并验证令牌
func ParseToken(tokenString string, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// generateToken 内部令牌生成函数
func generateToken(userID int64, username string, roles []string, permissions []string, tokenType string, secret string, expire time.Duration) (string, error) {
	claims := &Claims{
		UserID:      userID,
		Username:    username,
		Roles:       roles,
		Permissions: permissions,
		TokenType:   tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expire)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}
