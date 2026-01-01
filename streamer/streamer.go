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

// pathCache 缓存 "相对路径 -> 绝对路径" 的映射
// 作用：加速后续的分片请求 (Chunk Requests)
var pathCache sync.Map

// FileResult 封装搜索结果
type FileResult struct {
	File *os.File
	Path string
}

// openFileConcurrently 并发尝试打开文件
// 优势：将 (Stat + Open) 合并为一次 Open 操作，减少 50% 的云盘交互耗时
func openFileConcurrently(relativePath string) (*os.File, string, error) {
	cleanRelPath := filepath.Clean(relativePath)

	// ----------------------
	// 1. 快速通道：查缓存
	// ----------------------
	if val, ok := pathCache.Load(cleanRelPath); ok {
		cachedFullPath := val.(string)
		// 尝试直接打开缓存的路径
		f, err := os.Open(cachedFullPath)
		if err == nil {
			// logger.Debug("Cache hit", "path", cachedFullPath) // 调试可开启
			return f, cachedFullPath, nil
		}
		// 如果打开失败（文件被删或移动），清除缓存，回退到搜索模式
		pathCache.Delete(cleanRelPath)
	}

	// ----------------------
	// 2. 慢速通道：并发搜索
	// ----------------------
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
			// 如果成功，我们直接拿到了文件句柄，后续不用再 Open 了
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
		
		// 记录到日志 (仅首次搜索时)
		logger.Info("File opened (Search)", "path", res.Path)

		// 写入缓存，方便下次直接命中
		pathCache.Store(cleanRelPath, res.Path)
		
		// 简单的缓存过期策略（可选）：1小时后清理
		// 避免长期运行内存占用过大，虽说存字符串也占不了多少
		go func(k string) {
			time.Sleep(1 * time.Hour)
			pathCache.Delete(k)
		}(cleanRelPath)

		return res.File, res.Path, nil

	case <-time.After(10 * time.Second): // 防止极端卡死
		return nil, "", errors.New("search timeout")
	}
}

// ServeFile 优化后的推流入口
func ServeFile(c *gin.Context, relativePath string) {
	// 1. 获取已打开的文件句柄 (0-Copy 准备)
	file, fullPath, err := openFileConcurrently(relativePath)
	if err != nil {
		logger.Error("File open failed", "err", err, "path", relativePath)
		c.String(http.StatusNotFound, "File not found")
		return
	}
	// ⚠️ 关键：请求结束时关闭文件句柄
	defer file.Close()

	// 2. 获取文件元数据 (用于 Content-Length 和 Last-Modified)
	// 因为文件已经打开，f.Stat() 通常是内存操作，极快
	fileInfo, err := file.Stat()
	if err != nil {
		c.String(http.StatusInternalServerError, "File stat failed")
		return
	}

	// 3. 设置缓存头 (配合 CDN/网盘)
	c.Header("Cache-Control", "public, max-age=31536000, immutable")
	
	// 4. 使用 ServeContent 代替 ServeFile
	// http.ServeContent 接受 io.ReadSeeker。
	// 当传入 *os.File 时，Go 标准库底层仍会尝试优化 (如 sendfile 或高效 copy)
	// 且它自动处理 Range 请求 (断点续传)
	http.ServeContent(c.Writer, c.Request, filepath.Base(fullPath), fileInfo.ModTime(), file)
}
