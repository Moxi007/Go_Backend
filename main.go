package main

import (
	"Go_Backend/config"
	"Go_Backend/logger"
	"Go_Backend/streamer"
	"fmt"
	"os"

	"github.com/gin-gonic/gin"

	// ✅ 优化: 自动适配容器 CPU 配额，防止在 Docker 中发生 CPU 节流
	// 需先运行: go get go.uber.org/automaxprocs
	_ "go.uber.org/automaxprocs"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: ./go_backend <config_file_path>")
		return
	}

	// 1. 加载配置
	config.LoadConfig(os.Args[1])

	// 2. 初始化日志
	logger.InitializeLogger(config.GlobalConfig.LogLevel)
	
	logger.Info("Backend starting...", "version", "3.0-TTL-Optimized", "port", config.GlobalConfig.Server.Port)

	// 3. 设置 Gin 模式
	if config.GlobalConfig.LogLevel == "DEBUG" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
	
	r := gin.New()
	r.Use(gin.Recovery())

	// 4. 注册路由
	r.GET("/stream", streamer.HandleStreamRequest)

	// 5. 启动服务
	addr := fmt.Sprintf(":%d", config.GlobalConfig.Server.Port)
	logger.Info("Server listening", "address", addr)
	
	if err := r.Run(addr); err != nil {
		logger.Error("Startup failed", "error", err)
	}
}
