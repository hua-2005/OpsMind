// Package main 是 OpsMind 后端服务的入口。
//
// 负责初始化配置、数据库连接、路由注册和 HTTP 服务启动。
// MVP 阶段采用单体分层架构，所有模块在同一进程内运行。
package main

import (
	"fmt"
	"log/slog"
	"os"

	"opsmind/internal/config"
	"opsmind/internal/database"
	"opsmind/internal/handler"
	"opsmind/internal/repository"
	"opsmind/internal/router"
	"opsmind/internal/service"
)

func main() {
	slog.Info("OpsMind 服务启动中...")

	// 1. 加载配置
	cfg, err := config.Load("")
	if err != nil {
		slog.Error("加载配置失败", "error", err)
		os.Exit(1)
	}

	// 生产模式下 JWT 密钥必须非空，否则拒绝启动。
	if cfg.JWT.Secret == "" {
		if cfg.Server.Mode == "release" {
			slog.Error("JWT 密钥为空，生产模式不允许启动，请设置环境变量 OPSMIND_JWT_SECRET")
			os.Exit(1)
		}
		slog.Warn("JWT 密钥为空，JWT 认证功能不可用（仅调试模式允许）")
	}

	// 2. 初始化数据库连接
	db, err := database.Init(cfg.Database)
	if err != nil {
		slog.Error("数据库连接失败", "error", err)
		os.Exit(1)
	}
	slog.Info("数据库连接成功")

	// 3. 自动迁移（开发/测试阶段自动建表，生产环境建议用独立迁移脚本）
	if err := database.AutoMigrate(db); err != nil {
		slog.Error("数据库迁移失败", "error", err)
		os.Exit(1)
	}
	slog.Info("数据库迁移完成")

	// 4. 初始化 Repository 层
	userRepo := repository.NewUserRepo(db)
	roleRepo := repository.NewRoleRepo(db)
	// 后续里程碑补充：configRepo, ticketRepo, knowledgeRepo, chatRepo, auditRepo, messageRepo

	// 5. 初始化 Service 层
	authService := service.NewAuthService(userRepo, db)
	userService := service.NewUserService(userRepo, db)
	roleService := service.NewRoleService(roleRepo, db)
	// 后续里程碑补充：ticketService, knowledgeService, chatService, dashboardService, configService, messageService

	// 6. 初始化 Handler 层
	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler(userService)
	roleHandler := handler.NewRoleHandler(roleService)
	// 后续里程碑补充：ticketHandler, knowledgeHandler, chatHandler, dashboardHandler, configHandler, messageHandler, auditHandler

	// 7. 设置路由
	r := router.Setup(cfg, &router.Handlers{
		Auth: authHandler,
		User: userHandler,
		Role: roleHandler,
	})

	// 8. 启动 HTTP 服务
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	slog.Info("HTTP 服务已启动", "addr", addr)

	if err := r.Run(addr); err != nil {
		slog.Error("HTTP 服务启动失败", "error", err)
		os.Exit(1)
	}
}
