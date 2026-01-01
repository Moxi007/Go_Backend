package logger

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
)

// 日志级别常量
const (
	DEBUG = iota
	INFO
	WARN
	ERROR
)

type Logger struct {
	level int
	mu    sync.Mutex
}

var instance *Logger
var once sync.Once

// InitializeLogger 初始化日志系统
func InitializeLogger(levelStr string) {
	once.Do(func() {
		lvl := INFO // 默认 INFO
		switch strings.ToUpper(levelStr) {
		case "DEBUG":
			lvl = DEBUG
		case "WARN":
			lvl = WARN
		case "ERROR":
			lvl = ERROR
		}

		instance = &Logger{
			level: lvl,
		}
		// 设置标准 log 的输出格式 (日期+时间)
		log.SetFlags(log.Ldate | log.Ltime)
		log.SetOutput(os.Stdout)
		
		fmt.Printf("Logger initialized with level: %s\n", strings.ToUpper(levelStr))
	})
}

// 内部输出函数
func (l *Logger) print(level int, prefix string, msg string, keysAndValues ...interface{}) {
	if l == nil || level < l.level {
		return
	}

	// 拼接键值对
	var kvBuilder strings.Builder
	for i := 0; i < len(keysAndValues); i += 2 {
		if i+1 < len(keysAndValues) {
			kvBuilder.WriteString(fmt.Sprintf(" %v=%v", keysAndValues[i], keysAndValues[i+1]))
		}
	}

	// 颜色控制 (可选，这里用简单的符号区分)
	log.Printf("[%s] %s%s\n", prefix, msg, kvBuilder.String())
}

// Info 打印信息日志
func Info(msg string, keysAndValues ...interface{}) {
	if instance == nil { InitializeLogger("INFO") } // 防止未初始化崩溃
	instance.print(INFO, "INFO", msg, keysAndValues...)
}

// Error 打印错误日志
func Error(msg string, keysAndValues ...interface{}) {
	if instance == nil { InitializeLogger("INFO") }
	instance.print(ERROR, "ERRO", msg, keysAndValues...)
}

// Debug 打印调试日志
func Debug(msg string, keysAndValues ...interface{}) {
	if instance == nil { InitializeLogger("INFO") }
	instance.print(DEBUG, "DBUG", msg, keysAndValues...)
}
