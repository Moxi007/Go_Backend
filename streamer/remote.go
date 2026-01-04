// remote.go
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
    // 🔥 记录请求到达的精确时间
    arriveTime := time.Now()
    requestId := arriveTime.UnixNano() % 100000

    logger.Debug("🟢 Request ARRIVED",
        "id", requestId,
        "time", arriveTime.Format("15:04:05.000"),
        "client", c.ClientIP(),
        "path", c.Query("path"))

    defer func() {
        logger.Debug("🔴 Request END", "id", requestId, "duration_ms", time.Since(arriveTime).Milliseconds())
    }()

    // 1. 获取 URL 参数
    pathFromUrl := c.Query("path")
    signature := c.Query("signature")

    if pathFromUrl == "" || signature == "" {
        c.String(http.StatusBadRequest, "Missing path or signature")
        return
    }

    // 2. 验证签名
    t1 := time.Now()
    if !verifySignature(signature, config.GlobalConfig.Encipher) {
        logger.Error("Access Denied", "ip", c.ClientIP(), "reason", "Invalid Signature")
        c.String(http.StatusForbidden, "Invalid or expired signature")
        return
    }
    logger.Debug("Signature verified", "id", requestId, "took_ms", time.Since(t1).Milliseconds())

    // 3. 验证通过，移交推流逻辑
    ServeFile(c, pathFromUrl)
}

// verifySignature 保持不变...
func verifySignature(signatureStr string, secret string) bool {
    payloadJson, err := base64.StdEncoding.DecodeString(signatureStr)
    if err != nil {
        return false
    }

    var payload map[string]string
    if err := json.Unmarshal(payloadJson, &payload); err != nil {
        return false
    }

    dataB64, ok1 := payload["data"]
    sigB64, ok2 := payload["signature"]
    if !ok1 || !ok2 {
        return false
    }

    dataBytes, err := base64.StdEncoding.DecodeString(dataB64)
    if err != nil {
        return false
    }

    sigBytes, err := base64.StdEncoding.DecodeString(sigB64)
    if err != nil {
        return false
    }

    h := hmac.New(sha256.New, []byte(secret))
    h.Write(dataBytes)
    computedSig := h.Sum(nil)

    if !hmac.Equal(sigBytes, computedSig) {
        return false
    }

    var dataMap map[string]interface{}
    if err := json.Unmarshal(dataBytes, &dataMap); err != nil {
        return false
    }

    expireAtVal, ok := dataMap["expireAt"].(float64)
    if !ok {
        return false
    }

    if time.Now().Unix() > int64(expireAtVal) {
        return false
    }

    return true
}