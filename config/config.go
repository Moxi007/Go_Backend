package config

import (
	"log"

	"github.com/spf13/viper"
)

// Mount 定义挂载点结构
type Mount struct {
	Name string `mapstructure:"name"`
	Path string `mapstructure:"path"`
	Root string `mapstructure:"root"`
}

// Config 全局配置结构
type Config struct {
	LogLevel string
	Encipher string
	Mounts   []Mount `mapstructure:"Mounts"` // 多挂载点支持
	Server   struct {
		Port int
	}
}

// 直接暴露全局变量，读取速度最快
var GlobalConfig *Config

func LoadConfig(path string) {
	viper.SetConfigFile(path)
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file: %s", err)
	}

	if err := viper.Unmarshal(&GlobalConfig); err != nil {
		log.Fatalf("Unable to decode into struct: %v", err)
	}
}
