// Gin 路由注册：全局中间件 + 公开路由 + 后台管理路由
package router

import (
	"opsmind/server/internal/api/v1"
	"opsmind/server/internal/config"
	"opsmind/server/internal/domain/audit"
	"opsmind/server/internal/domain/auth"
	"opsmind/server/internal/domain/user"
	"opsmind/server/internal/middleware"
	"opsmind/server/internal/pkg/errors"
	"opsmind/server/internal/pkg/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func Setup(db *gorm.DB, cfg *config.Config) *gin.Engine {
	r := gin.New()

	// 全局中间件
	r.Use(middleware.Recovery())
	r.Use(middleware.RequestID())

	// 统一 404/405
	r.NoRoute(func(c *gin.Context) {
		response.ErrorRaw(c, 404, errors.CodeNotFound, "接口不存在")
	})
	r.NoMethod(func(c *gin.Context) {
		response.ErrorRaw(c, 405, errors.CodeValidationError, "不支持的请求方法")
	})

	auditSvc := audit.NewService(db)
	authSvc := auth.NewService(db, cfg)
	userSvc := user.NewService(db)

	authH := v1.NewAuthHandler(authSvc, auditSvc)
	accountH := v1.NewAccountHandler(userSvc, auditSvc)
	roleH := v1.NewRoleHandler(userSvc)

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "opsmind-server"})
	})

	api := r.Group("/api/v1")
	{
		// 公开路由 — 认证
		authGroup := api.Group("/auth")
		{
			authGroup.POST("/login", authH.Login)
			authGroup.POST("/refresh", middleware.AuthRequired(authSvc), authH.Refresh)
			authGroup.POST("/logout", middleware.AuthRequired(authSvc), authH.Logout)
			authGroup.GET("/profile", middleware.AuthRequired(authSvc), authH.Profile)
			authGroup.GET("/permissions", middleware.AuthRequired(authSvc), authH.Permissions)
		}

		// 后台管理路由 — 需认证+鉴权
		admin := api.Group("/admin")
		admin.Use(middleware.AuthRequired(authSvc))
		{
			// 账号管理
			accounts := admin.Group("/accounts")
			{
				accounts.GET("", middleware.RequirePermission(db, "account:list"), accountH.List)
				accounts.POST("", middleware.RequirePermission(db, "account:create"), accountH.Create)
				accounts.GET("/:id", middleware.RequirePermission(db, "account:detail"), accountH.Detail)
				accounts.PATCH("/:id", middleware.RequirePermission(db, "account:update"), accountH.Update)
				accounts.POST("/:id/freeze", middleware.RequirePermission(db, "account:freeze"), accountH.Freeze)
				accounts.POST("/:id/restore", middleware.RequirePermission(db, "account:restore"), accountH.Restore)
			}

			// 角色权限
			admin.GET("/roles", middleware.RequirePermission(db, "role:list"), roleH.ListRoles)
			admin.GET("/permissions", middleware.RequirePermission(db, "permission:list"), roleH.GetPermissions)
			admin.PATCH("/roles/:id/permissions", middleware.RequirePermission(db, "role:permission:update"), roleH.UpdateRolePermissions)
		}
	}

	return r
}
