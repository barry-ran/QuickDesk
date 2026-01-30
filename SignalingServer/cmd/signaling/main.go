package main

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"

	"quickdesk/signaling/internal/config"
	"quickdesk/signaling/internal/database"
	"quickdesk/signaling/internal/handler"
	"quickdesk/signaling/internal/middleware"
	"quickdesk/signaling/internal/models"
	"quickdesk/signaling/internal/repository"
	"quickdesk/signaling/internal/service"
)

func main() {
	log.Println("Starting QuickDesk Signaling Server...")

	// Load configuration
	cfg := config.Load()

	// Initialize databases
	log.Println("Connecting to databases...")
	db := database.InitPostgreSQL(cfg)
	redisClient := database.InitRedis(cfg)

	// Auto-migrate models
	log.Println("Running database migrations...")
	if err := db.AutoMigrate(&models.Device{}); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Initialize repositories
	deviceRepo := repository.NewDeviceRepository(db)

	// Initialize services
	deviceService := service.NewDeviceService(deviceRepo, redisClient)
	authService := service.NewAuthService(redisClient)

	// Initialize handlers
	apiHandler := handler.NewAPIHandler(deviceService, authService, cfg)
	wsHandler := handler.NewWSHandler(deviceService, authService)
	
	// Set WSHandler reference for API handler (needed for online status checks)
	apiHandler.SetWSHandler(wsHandler)

	// Create Gin router
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.LoggerMiddleware())

	// Health check endpoint
	router.GET("/health", apiHandler.HealthCheck)

	// API routes
	v1 := router.Group("/api/v1")
	{
		// Device management
		v1.POST("/devices/register", apiHandler.RegisterDevice)
		v1.GET("/devices/:device_id", apiHandler.GetDevice)
		v1.GET("/devices/:device_id/status", apiHandler.GetDeviceStatus)
		
		// Authentication
		v1.POST("/auth/verify", apiHandler.VerifyPassword)
	}

	// WebSocket routes
	router.GET("/signal/:device_id", wsHandler.HandleWebSocket)

	// Legacy route for backward compatibility with existing tests
	router.GET("/host/:device_id", wsHandler.HandleWebSocket)
	router.GET("/client/:device_id/:access_code", func(c *gin.Context) {
		// Extract access_code from path
		accessCode := c.Param("access_code")

		// Set access_code as query parameter
		c.Request.URL.RawQuery = fmt.Sprintf("access_code=%s", accessCode)

		// Forward to standard WebSocket handler
		wsHandler.HandleWebSocket(c)
	})

	// Start server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("Server starting on %s", addr)
	log.Printf("API: http://%s/api/v1", addr)
	log.Printf("WebSocket: ws://%s/signal/{device_id}?access_code={code}", addr)
	log.Println("Ready to accept connections.")

	if err := router.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
