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

// Remote handles streaming a file and checking for valid Range requests.
func Remote(c *gin.Context) {
	// 获取参数
	signature := c.Query("signature")
	rawPath := c.Query("path")

	// 鉴权
	itemId, mediaId, expireAt, err := authenticate(c, signature)
	if err != nil {
		logger.Error("Authentication failed: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// 打印调试信息
	if config.GetConfig().LogLevel == "DEBUG" {
		beijingTime := expireAt.In(time.FixedZone("CST", 8*3600))
		logger.Debug(
			"Auth success | Path: %s | ItemID: %s | MediaID: %s | Expire: %s",
			rawPath, itemId, mediaId, beijingTime.Format("2006-01-02 15:04:05"),
		)
	}

	// --- 路径安全处理 ---
	basePath := config.GetConfig().StorageBasePath
	
	// 清洗路径，解析 ".." 等符号
	cleanPath := filepath.Clean(rawPath)
	
	// 防止路径穿越攻击 (虽然有签名校验，但做一层防御更好)
	if strings.Contains(cleanPath, "..") {
		logger.Error("Invalid path (directory traversal attempt)", "path", rawPath)
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	// 拼接完整路径
	// 注意：这里要确保 basePath 结尾和 cleanPath 开头的斜杠处理正确
	// filepath.Join 会自动处理分隔符
	fullFilePath := filepath.Join(basePath, cleanPath)

	// 调用优化后的 Stream
	Stream(c, fullFilePath)
}

// authenticate 保持不变，但建议检查一下 json 引入是否优化
func authenticate(c *gin.Context, signature string) (itemId, mediaId string, expireAt time.Time, err error) {
	sigInstance, initErr := GetSignatureInstance()
	if initErr != nil {
		return "", "", time.Time{}, initErr
	}

	data, decryptErr := sigInstance.Decrypt(signature)
	if decryptErr != nil {
		return "", "", time.Time{}, decryptErr
	}

	// ... (其余校验逻辑保持不变) ...
	itemIdValue, _ := data["itemId"].(string)
	mediaIdValue, _ := data["mediaId"].(string)
	expireAtValue, _ := data["expireAt"].(float64)

	if itemIdValue == "" || mediaIdValue == "" {
		return "", "", time.Time{}, errors.New("invalid payload")
	}

	expireAt = time.Unix(int64(expireAtValue), 0)
	if expireAt.Before(time.Now().UTC()) {
		return "", "", time.Time{}, errors.New("signature expired")
	}

	return itemIdValue, mediaIdValue, expireAt, nil
}
