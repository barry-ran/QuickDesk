package main

import (
	"context"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"

	signaling "quickdesk/signaling"
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
	if err := db.AutoMigrate(&models.Device{}, &models.Preset{}, &models.AdminUser{}); err != nil {
		log.Printf("Warning: migration error (continuing anyway): %v", err)
	}

	// Initialize repositories
	deviceRepo := repository.NewDeviceRepository(db)
	presetRepo := repository.NewPresetRepository(db)
	adminUserRepo := repository.NewAdminUserRepository(db)

	// Initialize services
	deviceService := service.NewDeviceService(deviceRepo, redisClient)
	authService := service.NewAuthService(redisClient)
	presetService := service.NewPresetService(presetRepo)
	adminUserService := service.NewAdminUserService(adminUserRepo)

	// Create initial admin user if not exists (use ADMIN_USER/ADMIN_PASSWORD from config)
	ctx := context.Background()
	if _, err := adminUserRepo.GetByUsername(ctx, cfg.Admin.User); err != nil {
		log.Printf("Creating initial admin user '%s'...", cfg.Admin.User)
		hashedPassword, err := service.HashPassword(cfg.Admin.Password)
		if err != nil {
			log.Fatalf("Failed to hash initial admin password: %v", err)
		}
		initialAdmin := &models.AdminUser{
			Username: cfg.Admin.User,
			Password: hashedPassword,
			Email:    "",
			Role:     "super_admin",
			Status:   true,
		}
		if err := adminUserRepo.Create(ctx, initialAdmin); err != nil {
			log.Fatalf("Failed to create initial admin user: %v", err)
		}
		log.Println("Initial admin user created successfully")
	}

	// Initialize handlers
	apiHandler := handler.NewAPIHandler(deviceService, authService, presetService, cfg)
	wsHandler := handler.NewWSHandler(deviceService, authService)
	
	// Set WSHandler reference for API handler (needed for online status checks)
	apiHandler.SetWSHandler(wsHandler)

	// Create Gin router
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.LoggerMiddleware())
	router.Use(middleware.CORSMiddleware(cfg.Security.AllowedOrigins))

	// API Key authentication middleware
	apiKeyAuth := middleware.NewAPIKeyAuth(cfg.Security.APIKey, cfg.Security.AllowedOrigins)

	// Health check endpoint (no API key required)
	router.GET("/health", apiHandler.HealthCheck)

	// API routes
	v1 := router.Group("/api/v1")
	{
		// Preset is public so old clients (without API key) can still
		// fetch min_version and show the force-upgrade prompt.
		v1.GET("/preset", apiHandler.GetPreset)

		// Client-facing APIs require API key
		clientAPI := v1.Group("")
		clientAPI.Use(apiKeyAuth.Required())
		{
			clientAPI.POST("/devices/register", apiHandler.RegisterDevice)
			clientAPI.GET("/devices/:device_id", apiHandler.GetDevice)
			clientAPI.GET("/devices/:device_id/status", apiHandler.GetDeviceStatus)
			clientAPI.POST("/auth/verify", apiHandler.VerifyPassword)
			clientAPI.GET("/ice-config", apiHandler.GetIceConfig)
		}

		// Admin authentication
		adminAuth := middleware.NewAdminAuth(adminUserService)
		v1.POST("/admin/login", adminAuth.Login)

		// Admin API (requires admin token, no API key needed)
		admin := v1.Group("/admin")
		admin.Use(adminAuth.AuthRequired())
		{
			admin.GET("/preset", apiHandler.GetAdminPreset)
			admin.PUT("/preset", apiHandler.UpdateAdminPreset)

			// Admin user management
			adminUserHandler := handler.NewAdminUserHandler(adminUserService)
			admin.GET("/users", adminUserHandler.GetAdminUsers)
			admin.POST("/users", adminUserHandler.CreateAdminUser)
			admin.PUT("/users/:id", adminUserHandler.UpdateAdminUser)
			admin.DELETE("/users/:id", adminUserHandler.DeleteAdminUser)
		}
	}

	// WebSocket routes (API key checked inside handler before upgrade)
	wsHandler.SetAPIKeyAuth(apiKeyAuth)
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

	// Admin UI (embedded Vue frontend)
	handler.RegisterAdminUI(router, signaling.WebDistFS)

	// Start server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("Server starting on %s", addr)
	log.Printf("API: http://%s/api/v1", addr)
	log.Printf("Admin: http://%s/admin/", addr)
	log.Printf("WebSocket: ws://%s/signal/{device_id}?access_code={code}", addr)
	log.Println("Ready to accept connections.")

	if err := router.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
