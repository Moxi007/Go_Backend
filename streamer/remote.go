package streamer

import (
	"PiliPili_Backend/config"
	"PiliPili_Backend/logger"
	"errors"
	"github.com/gin-gonic/gin"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

// Remote 处理远程流请求，进行鉴权和路径解析
func Remote(c *gin.Context) {
	// 获取 URL 参数
	signature := c.Query("signature")
	rawPath := c.Query("path")

	// 1. 签名鉴权
	itemId, mediaId, expireAt, err := authenticate(c, signature)
	if err != nil {
		logger.Error("Authentication failed: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// 仅在 DEBUG 模式下打印详细鉴权信息
	if config.GetConfig().LogLevel == "DEBUG" {
		beijingTime := expireAt.In(time.FixedZone("CST", 8*3600))
		logger.Debug(
			"Auth success | Path: %s | ItemID: %s | MediaID: %s | Expire: %s",
			rawPath, itemId, mediaId, beijingTime.Format("2006-01-02 15:04:05"),
		)
	}

	// 2. 路径安全处理 (关键安全优化)
	basePath := config.GetConfig().StorageBasePath
	
	// 清洗路径，解析 "." 和 ".."
	cleanPath := filepath.Clean(rawPath)
	
	// 防御性编程：再次检查是否包含 ".."，防止清洗后仍有逃逸风险
	if strings.Contains(cleanPath, "..") {
		logger.Error("Potential directory traversal attack detected", "path", rawPath)
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	// 安全拼接绝对路径
	fullFilePath := filepath.Join(basePath, cleanPath)

	// 3. 调用优化后的流传输方法
	Stream(c, fullFilePath)
}

// authenticate 验证签名并解密内容
func authenticate(c *gin.Context, signature string) (itemId, mediaId string, expireAt time.Time, err error) {
	sigInstance, initErr := GetSignatureInstance()
	if initErr != nil {
		logger.Error("Signature instance is not initialized", "error", initErr)
		return "", "", time.Time{}, initErr
	}

	// 解密签名
	data, decryptErr := sigInstance.Decrypt(signature)
	if decryptErr != nil {
		// 签名无效或解密失败
		return "", "", time.Time{}, decryptErr
	}

	// 类型断言获取字段
	itemIdValue, _ := data["itemId"].(string)
	mediaIdValue, _ := data["mediaId"].(string)
	expireAtValue, _ := data["expireAt"].(float64)

	// 校验字段完整性
	if itemIdValue == "" || mediaIdValue == "" {
		return "", "", time.Time{}, errors.New("invalid signature payload")
	}

	// 校验过期时间
	expireAt = time.Unix(int64(expireAtValue), 0)
	if expireAt.Before(time.Now().UTC()) {
		return "", "", time.Time{}, errors.New("signature has expired")
	}

	return itemIdValue, mediaIdValue, expireAt, nil
}
