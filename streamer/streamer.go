package streamer

import (
	"PiliPili_Backend/logger"
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
)

// Stream 处理文件流传输
// 优化：使用 http.ServeFile 代替手动读取，利用内核级零拷贝 (sendfile) 技术提升性能
func Stream(c *gin.Context, filePath string) {
	logger.Info("Starting file streaming", "filePath", filePath)

	// 1. 获取文件状态，检查文件是否存在
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Error("File not found", "filePath", filePath)
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
		// 其他错误（如权限不足）
		logger.Error("Error accessing file", "error", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	// 2. 安全检查：禁止直接访问目录
	if fileInfo.IsDir() {
		logger.Error("Path is a directory, denying access", "filePath", filePath)
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	logger.Info("Serving file via sendfile (zero-copy)", "size", fileInfo.Size())

	// 设置强缓存头
    // public: 允许中间人(CDN/Nginx)缓存
    // max-age: 缓存时间(秒)，这里设为 1 年，因为网盘文件通常不会变
    // immutable: 告诉客户端文件内容永远不会变
    c.Header("Cache-Control", "public, max-age=31536000, immutable")

	// 3. 核心传输逻辑
	// http.ServeFile 会自动处理：
	// - Content-Type (MIME 类型)
	// - Content-Length
	// - Range (断点续传/多线程下载)
	// - If-Modified-Since (缓存控制)
	http.ServeFile(c.Writer, c.Request, filePath)
}
