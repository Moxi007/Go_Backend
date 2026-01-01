package middleware

import (
	"Go_Backend/config"
	"Go_Backend/logger"
	"bytes"
	"github.com/gin-gonic/gin"
	"io"
)

// CorsMiddleware 处理 CORS 头并记录请求日志
func CorsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 记录基本请求信息
		logger.Info("Incoming request: %s %s", c.Request.Method, c.Request.URL.Path)
		
		// 优化：仅在 DEBUG 模式下且为 POST/PUT 时读取 Body
		// 避免生产环境无意义的内存拷贝
		cfg := config.GetConfig()
		if cfg.LogLevel == "DEBUG" && (c.Request.Method == "POST" || c.Request.Method == "PUT") {
			body, err := io.ReadAll(c.Request.Body)
			if err != nil {
				logger.Error("Error reading request body: %v", err)
			} else {
				// 读完后需要重写回 Body，否则后续 Handler 读不到
				c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
				logger.Debug("Request Body: %s", string(body))
			}
		}

		// 设置 CORS 头
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// 处理预检请求
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
