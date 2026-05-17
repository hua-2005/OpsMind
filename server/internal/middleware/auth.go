// JWT 认证中间件：提取 Token → 校验签名 → 校验账号状态（冻结/删除）
package middleware

import (
	"strings"

	"opsmind/server/internal/domain/auth"
	"opsmind/server/internal/pkg/errors"
	"opsmind/server/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

// AuthRequired 验证 Bearer Token，校验账号未被冻结或删除
func AuthRequired(authSvc *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractToken(c)
		if token == "" {
			response.Error(c, errors.New(errors.CodeUnauthorized, "未登录或 Token 缺失"))
			return
		}

		// 检查 Token 是否已登出（黑名单）
		if auth.IsBlacklisted(token) {
			response.Error(c, errors.New(errors.CodeUnauthorized, "Token 已失效，请重新登录"))
			return
		}

		claims, err := authSvc.ParseToken(token)
		if err != nil {
			response.Error(c, errors.New(errors.CodeUnauthorized, "Token 无效或已过期"))
			return
		}

		userID, _ := claims["user_id"].(float64)
		username, _ := claims["username"].(string)
		if userID == 0 {
			response.Error(c, errors.New(errors.CodeUnauthorized, "Token 无效"))
			return
		}

		// 校验账号是否仍然活跃 — 冻结后已签发的 token 立即失效
		active, err := authSvc.IsAccountActive(int64(userID))
		if err != nil || !active {
			response.Error(c, errors.New(errors.CodeAccountFrozen, "账号已被冻结或禁用"))
			return
		}

		c.Set("user_id", int64(userID))
		c.Set("username", username)
		c.Next()
	}
}

func extractToken(c *gin.Context) string {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		return ""
	}
	return strings.TrimPrefix(authHeader, "Bearer ")
}
