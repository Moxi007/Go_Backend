package streamer

import (
	"Go_Backend/config"
	"Go_Backend/logger"
	"context" // 新增
	"net/http"
	"os"
	"path/filepath"
	"sync" // 新增

	"github.com/gin-gonic/gin"
)

// getLocalPath 并发版：同时搜索所有挂载点，返回最快找到的那个
func getLocalPath(relativePath string) string {
	cleanRelPath := filepath.Clean(relativePath)
	cfg := config.GlobalConfig
	if cfg == nil || len(cfg.Mounts) == 0 {
		return ""
	}

	// 1. 创建通道接收结果 (Buffer设为1即可，因为我们只要第1个结果)
	resultCh := make(chan string, 1)
	
	// 2. 创建 WaitGroup 等待所有协程结束
	var wg sync.WaitGroup
	
	// 3. 创建 Context 用于取消操作 (一旦找到，通知其他人不用找了)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // 确保退出时释放资源

	// 4. 并发启动搜索任务
	for i := range cfg.Mounts {
		wg.Add(1)
		
		// ⚠️ 注意：将 mount 变量作为参数传递给协程，防止闭包捕获问题
		go func(m config.Mount) {
			defer wg.Done()

			// 优化：如果已经有人找到了，我就不费劲去 Stat 了
			select {
			case <-ctx.Done():
				return
			default:
			}

			fullPath := filepath.Join(m.Root, cleanRelPath)

			// 执行耗时的 IO 操作 (os.Stat)
			if _, err := os.Stat(fullPath); err == nil {
				// 找到了！
				select {
				case resultCh <- fullPath: // 尝试把结果发出去
					logger.Info("File found (Concurrent)", "backend", m.Name, "path", fullPath)
					cancel() // 广播：找到了，大家可以停了！
				case <-ctx.Done(): // 如果发不出去，说明已经有人先发了
				}
			}
		}(cfg.Mounts[i])
	}

	// 5. 启动一个守护协程，等所有人都干完活(或都失败)后关闭通道
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// 6. 等待结果
	// 如果找到文件，resultCh 会吐出路径
	// 如果所有盘都找不到，resultCh 会被 close，返回空字符串
	return <-resultCh
}

// ServeFile 保持不变 (引用上面的 getLocalPath 即可)
func ServeFile(c *gin.Context, relativePath string) {
	localAbsPath := getLocalPath(relativePath)
	
	if localAbsPath == "" {
		logger.Error("File not found in any mount point", "path", relativePath)
		c.String(http.StatusNotFound, "File not found")
		return
	}

	// 缓存控制
	c.Header("Cache-Control", "public, max-age=31536000, immutable")
	
	// 零拷贝传输
	http.ServeFile(c.Writer, c.Request, localAbsPath)
}
