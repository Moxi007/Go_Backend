package main

import (
	"Go_Backend/config"
	"Go_Backend/logger"  // 引用上面的 logger 包
	"Go_Backend/streamer"
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: ./go_backend <config_file_path>")
		return
	}

	// 1. 加载配置
	config.LoadConfig(os.Args[1])

	// 2. ✅✅✅ [关键] 初始化日志系统 ✅✅✅
	// 从配置中读取 LogLevel，如果没有配则默认 INFO
	logger.InitializeLogger(config.GlobalConfig.LogLevel)
	
	logger.Info("Backend starting...", "version", "2.0-Concurrent", "port", config.GlobalConfig.Server.Port)

	// 3. 设置 Gin 模式
	if config.GlobalConfig.LogLevel == "DEBUG" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode) // 生产模式，性能更好
	}
	
	r := gin.New()
	r.Use(gin.Recovery()) // 崩溃恢复

	// 4. 注册路由
	r.GET("/stream", streamer.HandleStreamRequest)

	// 5. 启动服务
	addr := fmt.Sprintf(":%d", config.GlobalConfig.Server.Port)
	logger.Info("Server listening", "address", addr)
	
	if err := r.Run(addr); err != nil {
		logger.Error("Startup failed", "error", err)
	}
}
