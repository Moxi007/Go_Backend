package streamer

import (
    "Go_Backend/config"
    "Go_Backend/logger"
    "context"
    "errors"
    "net/http"
    "os"
    "path/filepath"
    "sync"
    "time"

    "github.com/gin-gonic/gin"
)

// ---------------------------------------------------------
// TTL 缓存
// ---------------------------------------------------------

type PathCacheItem struct {
    FullPath  string
    ExpiresAt int64
}

type TTLCache struct {
    items sync.Map
}

func (c *TTLCache) Store(key, value string) {
    c.items.Store(key, PathCacheItem{
        FullPath:  value,
        ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
    })
}

func (c *TTLCache) Load(key string) (string, bool) {
    val, ok := c.items.Load(key)
    if !ok {
        return "", false
    }
    item := val.(PathCacheItem)
    if time.Now().Unix() > item.ExpiresAt {
        c.items.Delete(key)
        return "", false
    }
    return item.FullPath, true
}

func (c *TTLCache) StartCleanup() {
    go func() {
        ticker := time.NewTicker(10 * time.Minute)
        for range ticker.C {
            now := time.Now().Unix()
            c.items.Range(func(key, value interface{}) bool {
                item := value.(PathCacheItem)
                if now > item.ExpiresAt {
                    c.items.Delete(key)
                }
                return true
            })
        }
    }()
}

var pathCache = &TTLCache{}

func init() {
    pathCache.StartCleanup()
}

// ---------------------------------------------------------
// 并发文件搜索
// ---------------------------------------------------------

func openFileConcurrently(relativePath string) (*os.File, string, error) {
    cleanRelPath := filepath.Clean(relativePath)

    if cachedFullPath, ok := pathCache.Load(cleanRelPath); ok {
        f, err := os.Open(cachedFullPath)
        if err == nil {
            logger.Debug("Cache hit", "path", cachedFullPath)
            return f, cachedFullPath, nil
        }
        pathCache.items.Delete(cleanRelPath)
    }

    cfg := config.GlobalConfig
    if cfg == nil || len(cfg.Mounts) == 0 {
        return nil, "", errors.New("no mounts configured")
    }

    type searchResult struct {
        file *os.File
        path string
        err  error
    }

    resultCh := make(chan searchResult, len(cfg.Mounts))
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    for i := range cfg.Mounts {
        go func(m config.Mount) {
            select {
            case <-ctx.Done():
                return
            default:
            }

            fullPath := filepath.Join(m.Root, cleanRelPath)
            file, err := os.Open(fullPath)

            select {
            case resultCh <- searchResult{file: file, path: fullPath, err: err}:
            case <-ctx.Done():
                if file != nil {
                    file.Close()
                }
            }
        }(cfg.Mounts[i])
    }

    var lastErr error
    for i := 0; i < len(cfg.Mounts); i++ {
        select {
        case res := <-resultCh:
            if res.err == nil {
                cancel()
                go func() {
                    for j := i + 1; j < len(cfg.Mounts); j++ {
                        select {
                        case extra := <-resultCh:
                            if extra.file != nil {
                                extra.file.Close()
                            }
                        case <-time.After(100 * time.Millisecond):
                            return
                        }
                    }
                }()
                logger.Info("File opened (Search)", "path", res.path)
                pathCache.Store(cleanRelPath, res.path)
                return res.file, res.path, nil
            }
            lastErr = res.err

        case <-ctx.Done():
            return nil, "", errors.New("search timeout")
        }
    }

    if lastErr != nil {
        return nil, "", lastErr
    }
    return nil, "", errors.New("file not found in any mount")
}

// ---------------------------------------------------------
// 推流逻辑
// ---------------------------------------------------------

func ServeFile(c *gin.Context, relativePath string) {
    startTime := time.Now()
    logger.Info("Request received", "path", relativePath, "client", c.ClientIP())

    // 🔥 Keep-Alive 头
    c.Header("Connection", "keep-alive")
    c.Header("Keep-Alive", "timeout=60, max=1000")

    file, fullPath, err := openFileConcurrently(relativePath)
    if err != nil {
        logger.Error("File open failed", "err", err)
        c.String(http.StatusNotFound, "File not found")
        return
    }
    defer file.Close()

    logger.Info("File opened", "path", fullPath, "duration_ms", time.Since(startTime).Milliseconds())

    fileInfo, err := file.Stat()
    if err != nil {
        c.String(http.StatusInternalServerError, "File stat failed")
        return
    }

    c.Header("Cache-Control", "public, max-age=31536000, immutable")
    c.Header("Accept-Ranges", "bytes")
    c.Header("X-Content-Type-Options", "nosniff")

    logger.Info("Starting stream", "path", fullPath, "size_mb", fileInfo.Size()/1024/1024)

    ctx := c.Request.Context()
    http.ServeContent(c.Writer, c.Request, filepath.Base(fullPath), fileInfo.ModTime(), file)

    if ctx.Err() != nil {
        logger.Info("Stream cancelled by client", "path", fullPath, "reason", ctx.Err())
    } else {
        logger.Info("Stream completed", "path", fullPath, "duration_ms", time.Since(startTime).Milliseconds())
    }
}