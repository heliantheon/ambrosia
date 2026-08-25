package main

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/heliantheon/aegis-go/guard"
	ambrosiaconfig "github.com/heliantheon/ambrosia/config"
	ambrosia "github.com/heliantheon/ambrosia/internal"
	"github.com/heliantheon/common/config"
	"github.com/heliantheon/common/logger"
	"github.com/heliantheon/common/observability"
)

// @title Helios API
// @version 1.0
// @description Helios 统一后端 API - 提供认证、业务和身份与访问管理服务
// @host localhost:18000
// @BasePath /api
// @securityDefinitions.apikey Bearer
// @in header
// @name Authorization
// @description 输入 "Bearer {token}"
func main() {
	config.LoadAmbrosia()
	logger.InitWithConfig(logger.Config{
		Format: config.GetLogFormat(),
		Level:  config.GetLogLevel(),
		Debug:  config.IsDebug(),
	})
	defer logger.Sync()
	shutdownTracing, err := observability.Init(context.Background(), observability.Config{ServiceName: "ambrosia"})
	if err != nil {
		logger.Warnf("初始化 OpenTelemetry 失败，链路追踪保持降级: %v", err)
	}
	defer shutdownTracing()
	if err := ambrosiaconfig.Validate(); err != nil {
		logger.Fatalf("Ambrosia 配置校验失败: %v", err)
	}
	initTokenManager()

	db := ambrosiaconfig.InitDB()
	app, err := ambrosia.New(db)
	if err != nil {
		logger.Fatalf("初始化 Ambrosia 失败: %v", err)
	}

	if !config.IsDebug() {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(observability.GinMiddleware("ambrosia")...)
	r.Use(gin.Recovery())
	r.RedirectTrailingSlash = false

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	app.RegisterRoutes(r)

	addr := fmt.Sprintf(":%d", config.GetServerPort())
	logger.Infof("ambrosia 服务启动: %s", addr)
	if err := r.Run(addr); err != nil {
		logger.Fatalf("服务启动失败: %v", err)
	}
}

func initTokenManager() {
	seed, err := ambrosiaconfig.GetAegisSecretKeyBytes()
	if err != nil {
		logger.Fatalf("初始化 Ambrosia token manager 失败: %v", err)
	}
	if err := guard.NewServiceTokenManager(ambrosiaconfig.GetAegisIssuer(), ambrosiaconfig.GetAegisAudience(), seed); err != nil {
		logger.Fatalf("初始化 Ambrosia token manager 失败: %v", err)
	}
}
