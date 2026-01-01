package config

import (
	"github.com/spf13/viper"
	"strings"
)

// Config 保存所有配置值
type Config struct {
	Encipher        string // 加密密钥
	StorageBasePath string // 本地存储基路径
	Port            int    // 监听端口
	LogLevel        string // 日志级别
}

var globalConfig Config

// Initialize 从配置文件加载配置
func Initialize(configFile string, loglevel string) error {
	viper.SetConfigType("yaml")

	if configFile != "" {
		viper.SetConfigFile(configFile)
	}

	// 读取配置或使用默认值
	if err := viper.ReadInConfig(); err != nil {
		globalConfig = Config{
			Encipher:        "",
			StorageBasePath: "",
			Port:            60002,
			LogLevel:        defaultLogLevel(loglevel),
		}
	} else {
		globalConfig = Config{
			Encipher:        viper.GetString("Encipher"),
			StorageBasePath: viper.GetString("StorageBasePath"),
			Port:            viper.GetInt("Server.port"),
			LogLevel:        getLogLevel(loglevel),
		}
	}
	return nil
}

// GetConfig 返回全局配置 (优化：返回指针)
func GetConfig() *Config {
	return &globalConfig
}

func defaultLogLevel(loglevel string) string {
	if loglevel != "" {
		return loglevel
	}
	return "INFO"
}

func getLogLevel(loglevel string) string {
	if loglevel != "" {
		return loglevel
	}
	// 统一转大写，规范化
	return strings.ToUpper(viper.GetString("LogLevel"))
}
