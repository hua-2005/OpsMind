// RBAC 鉴权中间件：校验用户角色启停状态 + 权限码
package middleware

import (
	"opsmind/server/internal/pkg/errors"
	"opsmind/server/internal/pkg/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RequirePermission 校验权限码，须在 AuthRequired 之后使用
// 会校验：角色是否启用、权限是否启用
func RequirePermission(db *gorm.DB, permCode string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("user_id")
		if userID == 0 {
			response.Error(c, errors.New(errors.CodeUnauthorized, "未登录"))
			return
		}

		// 系统管理员拥有所有权限，跳过数据库校验
		if isAdmin(db, userID) {
			c.Next()
			return
		}

		var count int64
		db.Raw(`
			SELECT COUNT(*) FROM sys_permission p
			INNER JOIN sys_role_permission rp ON p.id = rp.permission_id
			INNER JOIN sys_role r ON r.id = rp.role_id AND r.status = 1
			INNER JOIN sys_user_role ur ON ur.role_id = r.id
			WHERE ur.user_id = ? AND p.code = ? AND p.status = 1
		`, userID, permCode).Scan(&count)

		if count == 0 {
			response.Error(c, errors.New(errors.CodeForbidden, "无操作权限"))
			return
		}

		c.Next()
	}
}

func isAdmin(db *gorm.DB, userID int64) bool {
	var count int64
	db.Raw(`
		SELECT COUNT(*) FROM sys_user_role ur
		INNER JOIN sys_role r ON r.id = ur.role_id
		WHERE ur.user_id = ? AND r.code = 'admin' AND r.status = 1
	`, userID).Scan(&count)
	return count > 0
}
