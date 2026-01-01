package config

import (
	"github.com/spf13/viper"
	"strings"
)

type Config struct {
	Encipher        string
	StorageBasePath string
	Port            int
	LogLevel        string
}

var globalConfig Config

func Initialize(configFile string, loglevel string) error {
	viper.SetConfigType("yaml")

	if configFile != "" {
		viper.SetConfigFile(configFile)
	}

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

// GetConfig 返回指针
func GetConfig() *Config {
	return &globalConfig
}

func defaultLogLevel(loglevel string) string {
	if loglevel != "" { return loglevel }
	return "INFO"
}

func getLogLevel(loglevel string) string {
	if loglevel != "" { return loglevel }
	return strings.ToUpper(viper.GetString("LogLevel"))
}
