package main

import (
	"Go_Backend/config"
	"Go_Backend/logger"
	"Go_Backend/middleware"
	"Go_Backend/streamer"
	"fmt"
	"github.com/gin-gonic/gin"
	"log"
	"os"
	"strconv"
)

func initializeConfig(configFile string) error {
	logger.Info("Initializing config...")

	err := config.Initialize(configFile, "")
	if err != nil {
		log.Printf("Error initializing config: %v", err)
		return err
	}

	logger.Info("Configuration initialized successfully")

	// 使用 config.GetConfig() (返回指针)
	loglevel := config.GetConfig().LogLevel
	logger.InitializeLogger(loglevel)

	encipher := config.GetConfig().Encipher
	if err := streamer.InitializeSignature(encipher); err != nil {
		logger.Error("Failed to initialize Signature", "error", err)
		return err
	}
	logger.Info("Signature initialized successfully")

	return nil
}

func initializeGinEngine() *gin.Engine {
	logger.Info("Initializing Gin engine...")

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.Use(middleware.CorsMiddleware())
	// 注册路由
	r.GET("/stream", streamer.Remote)

	logger.Info("Gin engine initialized successfully")
	return r
}

func startServer(r *gin.Engine) error {
	logger.Info("Starting the server...")

	port := config.GetConfig().Port
	if port == 0 {
		port = 60002
	}
	err := r.Run("0.0.0.0:" + strconv.Itoa(port))
	if err != nil {
		logger.Error("Error starting server: %v", err)
		return err
	}

	logger.Info("Server started successfully on port %d", port)
	return nil
}

func handleRequest(configFile string) error {
	logger.SetDefaultLogger()
	logger.Info("\n-----------------------------------------------\n")
	logger.Info("Start request handle.")

	if err := initializeConfig(configFile); err != nil {
		return err
	}
	r := initializeGinEngine()
	if err := startServer(r); err != nil {
		return err
	}

	return nil
}

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Println("Please provide the configuration file as an argument.")
		return
	}
	configFile := args[0]

	if err := handleRequest(configFile); err != nil {
		log.Fatalf("Request handling failed: %v", err)
	}
}
