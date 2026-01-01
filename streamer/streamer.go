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
// ✅ 优化模块: 自定义 TTL 缓存
// 解决问题: 替代 sync.Map + Sleep 模式，消除协程泄漏隐患
// ---------------------------------------------------------

type PathCacheItem struct {
	FullPath  string
	ExpiresAt int64
}

type TTLCache struct {
	items sync.Map
}

// Store 存入缓存，固定 1 小时有效期
func (c *TTLCache) Store(key, value string) {
	c.items.Store(key, PathCacheItem{
		FullPath:  value,
		ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
	})
}

// Load 读取缓存，懒惰删除过期项
func (c *TTLCache) Load(key string) (string, bool) {
	val, ok := c.items.Load(key)
	if !ok {
		return "", false
	}
	item := val.(PathCacheItem)
	// 检查是否过期
	if time.Now().Unix() > item.ExpiresAt {
		c.items.Delete(key)
		return "", false
	}
	return item.FullPath, true
}

// StartCleanup 启动单例守护协程，每 10 分钟清理一次所有过期项
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

// 全局缓存实例
var pathCache = &TTLCache{}

func init() {
	pathCache.StartCleanup() // 程序启动时开启清理任务
}

// ---------------------------------------------------------
// ✅ 核心逻辑: 并发搜索与打开
// ---------------------------------------------------------

// FileResult 封装搜索结果
type FileResult struct {
	File *os.File
	Path string
}

// openFileConcurrently 并发尝试打开文件
// 优势：将 (Stat + Open) 合并为一次 Open 操作，减少 50% 的云盘网络交互
func openFileConcurrently(relativePath string) (*os.File, string, error) {
	cleanRelPath := filepath.Clean(relativePath)

	// 1. 快速通道：查缓存
	if cachedFullPath, ok := pathCache.Load(cleanRelPath); ok {
		// 尝试直接打开缓存的路径
		f, err := os.Open(cachedFullPath)
		if err == nil {
			return f, cachedFullPath, nil
		}
		// 打开失败说明文件可能被移动或删除，移除缓存
		pathCache.items.Delete(cleanRelPath)
	}

	// 2. 慢速通道：并发搜索
	cfg := config.GlobalConfig
	if cfg == nil || len(cfg.Mounts) == 0 {
		return nil, "", errors.New("no mounts configured")
	}

	// 缓冲区设为1，只要有一个赢家即可
	successCh := make(chan FileResult, 1)
	
	// 上下文控制，一旦找到，通知其他协程停止
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	for i := range cfg.Mounts {
		wg.Add(1)
		go func(m config.Mount) {
			defer wg.Done()

			// 快速失败检查
			select {
			case <-ctx.Done():
				return
			default:
			}

			fullPath := filepath.Join(m.Root, cleanRelPath)

			// 🔥 核心优化：直接 Open，而不是先 Stat
			file, err := os.Open(fullPath)
			if err == nil {
				// 尝试提交结果
				select {
				case successCh <- FileResult{File: file, Path: fullPath}:
					cancel() // 我赢了，其他人可以停了
				default:
					// 通道已满（已经有人赢了），或者超时
					// 必须关闭我刚刚打开的文件，防止句柄泄露
					file.Close()
				}
			}
		}(cfg.Mounts[i])
	}

	// 守护协程：所有人都找完了还没找到，就关闭通道
	go func() {
		wg.Wait()
		close(successCh)
	}()

	// 等待结果
	select {
	case res, ok := <-successCh:
		if !ok {
			return nil, "", errors.New("file not found in any mount")
		}
		
		logger.Info("File opened (Search)", "path", res.Path)

		// 写入 TTL 缓存
		pathCache.Store(cleanRelPath, res.Path)

		return res.File, res.Path, nil

	case <-time.After(10 * time.Second): // 防止极端卡死
		return nil, "", errors.New("search timeout")
	}
}

// ServeFile 优化后的推流入口
func ServeFile(c *gin.Context, relativePath string) {
	// 1. 获取已打开的文件句柄 (Zero-Copy 准备)
	file, fullPath, err := openFileConcurrently(relativePath)
	if err != nil {
		logger.Error("File open failed", "err", err, "path", relativePath)
		c.String(http.StatusNotFound, "File not found")
		return
	}
	// ⚠️ 关键：请求结束时关闭文件句柄
	defer file.Close()

	// 2. 获取文件元数据 (用于 Content-Length 和 Last-Modified)
	// 因为文件已经打开，Stat() 通常是内存操作，极快
	fileInfo, err := file.Stat()
	if err != nil {
		c.String(http.StatusInternalServerError, "File stat failed")
		return
	}

	// 3. 设置缓存头 (配合 CDN/网盘)
	c.Header("Cache-Control", "public, max-age=31536000, immutable")
	
	// 4. 使用 ServeContent 代替 ServeFile
	// 它接受 *os.File 并自动处理 Range 请求，同时利用底层系统调用优化传输
	http.ServeContent(c.Writer, c.Request, filepath.Base(fullPath), fileInfo.ModTime(), file)
}
