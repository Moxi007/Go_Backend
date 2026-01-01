package streamer

import (
	"PiliPili_Backend/logger"
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
)

// Stream 直接使用 http.ServeFile 进行零拷贝传输
// 标准库会自动处理 Range 头、Content-Type 和 Last-Modified
func Stream(c *gin.Context, filePath string) {
	logger.Info("Starting file streaming", "filePath", filePath)

	// 检查文件是否存在
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Error("File not found", "filePath", filePath)
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
		logger.Error("Error accessing file", "error", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	// 如果是目录，禁止访问
	if fileInfo.IsDir() {
		logger.Error("Path is a directory, denying access", "filePath", filePath)
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	logger.Info("Serving file via sendfile (zero-copy)", "size", fileInfo.Size())

	// http.ServeFile 会处理 Range 请求、ETag、Last-Modified 等
	// 并且在支持的系统上使用 sendfile 系统调用
	http.ServeFile(c.Writer, c.Request, filePath)
}
