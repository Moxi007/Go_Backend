package main

import (
    "Go_Backend/config"
    "Go_Backend/logger"
    "Go_Backend/streamer"
    "fmt"
    "net/http"
    "os"
    "time"

    "github.com/gin-gonic/gin"
    _ "go.uber.org/automaxprocs"
)

func main() {
    if len(os.Args) < 2 {
        fmt.Println("Usage: ./go_backend <config_file_path>")
        return
    }

    config.LoadConfig(os.Args[1])
    logger.InitializeLogger(config.GlobalConfig.LogLevel)

    logger.Info("Backend starting...", "version", "3.5-KeepAlive", "port", config.GlobalConfig.Server.Port)

    if config.GlobalConfig.LogLevel == "DEBUG" {
        gin.SetMode(gin.DebugMode)
    } else {
        gin.SetMode(gin.ReleaseMode)
    }

    r := gin.New()
    r.Use(gin.Recovery())

    // 🔥 全局 Keep-Alive 中间件
    r.Use(func(c *gin.Context) {
        c.Header("Connection", "keep-alive")
        c.Next()
    })

    r.GET("/stream", streamer.HandleStreamRequest)

    // 🔥 优化 HTTP Server 配置
    srv := &http.Server{
        Addr:              fmt.Sprintf(":%d", config.GlobalConfig.Server.Port),
        Handler:           r,
        ReadTimeout:       0,                // 视频流不限读超时
        WriteTimeout:      0,                // 视频流不限写超时
        IdleTimeout:       120 * time.Second, // Keep-Alive 空闲超时
        ReadHeaderTimeout: 10 * time.Second,
        MaxHeaderBytes:    1 << 20,
    }

    // 🔥 启用 TCP Keep-Alive
    srv.SetKeepAlivesEnabled(true)

    logger.Info("Server listening", "address", srv.Addr, "keepalive", "enabled")
    if err := srv.ListenAndServe(); err != nil {
        logger.Error("Startup failed", "error", err)
    }
}