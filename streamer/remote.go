package streamer

import (
	"Go_Backend/config"
	"Go_Backend/logger"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// HandleStreamRequest 处理流媒体请求入口
func HandleStreamRequest(c *gin.Context) {
	// 1. 获取 URL 参数
	// path: 相对路径 (例如 "2023/Movie.mp4")
	// signature: 包含过期时间等信息的加密串
	pathFromUrl := c.Query("path")
	signature := c.Query("signature")

	if pathFromUrl == "" || signature == "" {
		c.String(http.StatusBadRequest, "Missing path or signature")
		return
	}

	// 2. 验证签名 (直接在此处处理，无需 cipher 包)
	if !verifySignature(signature, config.GlobalConfig.Encipher) {
		logger.Error("Access Denied", "ip", c.ClientIP(), "reason", "Invalid Signature")
		c.String(http.StatusForbidden, "Invalid or expired signature")
		return
	}

	// 3. 验证通过，移交推流逻辑
	ServeFile(c, pathFromUrl)
}

// verifySignature 验证前端生成的签名 (对应前端 stream/signature.go 的逻辑)
func verifySignature(signatureStr string, secret string) bool {
	// A. 解码最外层的 Base64
	payloadJson, err := base64.StdEncoding.DecodeString(signatureStr)
	if err != nil {
		return false
	}

	// B. 解析 JSON {"data": "...", "signature": "..."}
	var payload map[string]string
	if err := json.Unmarshal(payloadJson, &payload); err != nil {
		return false
	}

	dataB64, ok1 := payload["data"]
	sigB64, ok2 := payload["signature"]
	if !ok1 || !ok2 {
		return false
	}

	// C. 解码内部数据
	dataBytes, err := base64.StdEncoding.DecodeString(dataB64)
	if err != nil { return false }

	sigBytes, err := base64.StdEncoding.DecodeString(sigB64)
	if err != nil { return false }

	// D. 验证 HMAC-SHA256 签名 (防篡改)
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(dataBytes)
	computedSig := h.Sum(nil)

	if !hmac.Equal(sigBytes, computedSig) {
		return false
	}

	// E. 验证过期时间 (防盗链)
	var dataMap map[string]interface{}
	if err := json.Unmarshal(dataBytes, &dataMap); err != nil {
		return false
	}

	expireAtVal, ok := dataMap["expireAt"].(float64) // JSON数字默认为float64
	if !ok {
		return false
	}

	if time.Now().Unix() > int64(expireAtVal) {
		return false // 已过期
	}

	return true
}
