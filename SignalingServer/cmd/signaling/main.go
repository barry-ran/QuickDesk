package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

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

	cfg := config.Load()

	log.Println("Connecting to databases...")
	db := database.InitPostgreSQL(cfg)
	redisClient := database.InitRedis(cfg)

	log.Println("Running database migrations...")
	if err := db.AutoMigrate(
		&models.Device{}, &models.Preset{}, &models.AdminUser{}, &models.User{},
		&models.UserDevice{}, &models.ConnectionHistory{}, &models.Settings{},
		&models.UserFavorite{},
	); err != nil {
		log.Printf("Warning: migration error (continuing anyway): %v", err)
	}

	// Initialize settings service and seed from .env on first run
	settingsService := service.NewSettingsService(db)
	settingsService.SeedFromEnv(service.EnvSeed{
		TurnURLs:       strings.Join(cfg.Ice.TurnURLs, "\n"),
		TurnAuthSecret: cfg.Ice.AuthSecret,
		TurnTTL:        cfg.Ice.CredentialTTL,
		StunURLs:       strings.Join(cfg.Ice.StunURLs, "\n"),
		APIKey:         cfg.Security.APIKey,
		AllowedOrigins: strings.Join(cfg.Security.AllowedOrigins, "\n"),
		SmsKeyID:       cfg.Sms.AccessKeyID,
		SmsKeySecret:   cfg.Sms.AccessKeySecret,
		SmsSignName:    cfg.Sms.SignName,
		SmsTemplateCode: cfg.Sms.TemplateCode,
	})

	deviceRepo := repository.NewDeviceRepository(db)
	presetRepo := repository.NewPresetRepository(db)
	adminUserRepo := repository.NewAdminUserRepository(db)

	deviceService := service.NewDeviceService(deviceRepo, redisClient)
	authService := service.NewAuthService(redisClient)
	presetService := service.NewPresetService(presetRepo)
	adminUserService := service.NewAdminUserService(adminUserRepo)

	// Create initial admin user if not exists
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
	apiHandler := handler.NewAPIHandler(deviceService, authService, presetService, settingsService, cfg)
	wsHandler := handler.NewWSHandler(deviceService, authService, db, redisClient)
	
	apiHandler.SetWSHandler(wsHandler)

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.LoggerMiddleware())
	router.Use(middleware.CORSMiddleware(settingsService))

	apiKeyAuth := middleware.NewAPIKeyAuth(settingsService)

	router.GET("/health", apiHandler.HealthCheck)

	v1 := router.Group("/api/v1")
	{
		v1.GET("/preset", apiHandler.GetPreset)

		settingsHandler := handler.NewSettingsHandler(settingsService)
		v1.GET("/settings", settingsHandler.GetPublicSettings)

		// Public feature flags (clients use this to decide which UI to show)
		v1.GET("/features", apiHandler.GetFeatures)

		// SMS service – reads credentials dynamically from settingsService
		smsService := service.NewSmsService(redisClient, settingsService)

		// SMS verification code endpoint
		smsHandler := handler.NewSmsHandler(smsService, db)
		v1.POST("/sms/send", smsHandler.SendCode)

		// User authentication (public, no API key required)
		userAuth := handler.NewUserAuth(db, redisClient)
		userAuth.SetSmsService(smsService)
		v1.POST("/user/register", userAuth.Register)
		v1.POST("/user/login", userAuth.Login)
		v1.POST("/user/login-sms", userAuth.LoginWithSms)
		v1.POST("/user/logout", userAuth.Logout)
		v1.POST("/user/reset-password", userAuth.SendResetPasswordCode)
		v1.PUT("/user/reset-password", userAuth.ResetPassword)

		userDeviceHandler := handler.NewUserDeviceHandler(db)
		userDeviceHandler.SetSyncNotifier(func(userID uint, msg interface{}) {
			wsHandler.NotifyUserSync(userID, msg)
		})
		userAPI := v1.Group("/user")
		userAPI.Use(userAuth.AuthRequired())
		{
			userAPI.GET("/me", userAuth.GetMe)
			userAPI.PUT("/password", userAuth.ChangePassword)
			userAPI.PUT("/username", userAuth.ChangeUsername)
			userAPI.PUT("/phone", userAuth.ChangePhone)
			userAPI.PUT("/email", userAuth.ChangeEmail)
			userAPI.GET("/devices", userDeviceHandler.GetUserDevices)
			userAPI.POST("/devices/unbind", userDeviceHandler.UnbindDevice)
			userAPI.POST("/devices/auto-bind", userDeviceHandler.AutoBindDevice)
			userAPI.POST("/devices/record", userDeviceHandler.RecordConnection)
			userAPI.GET("/devices/logs", userDeviceHandler.GetUserDeviceLogs)
			userAPI.PUT("/devices/:device_id/access-code", userDeviceHandler.UpdateAccessCode)
			userAPI.PUT("/devices/:device_id/remark", userDeviceHandler.UpdateDeviceRemark)
			userAPI.POST("/devices/:device_id/login", userDeviceHandler.DeviceLogin)
			userAPI.POST("/devices/:device_id/logout", userDeviceHandler.DeviceLogout)
			userAPI.GET("/favorites", userDeviceHandler.GetFavorites)
			userAPI.POST("/favorites", userDeviceHandler.AddFavorite)
			userAPI.PUT("/favorites/:device_id", userDeviceHandler.UpdateFavorite)
			userAPI.DELETE("/favorites/:device_id", userDeviceHandler.RemoveFavorite)
		}

		clientAPI := v1.Group("")
		clientAPI.Use(apiKeyAuth.Required())
		{
			clientAPI.POST("/devices/register", apiHandler.RegisterDevice)
			clientAPI.GET("/devices/:device_id", apiHandler.GetDevice)
			clientAPI.GET("/devices/:device_id/status", apiHandler.GetDeviceStatus)
			clientAPI.POST("/auth/verify", apiHandler.VerifyPassword)
			clientAPI.GET("/ice-config", apiHandler.GetIceConfig)
		}

		adminAuth := middleware.NewAdminAuth(adminUserService, redisClient)
		v1.POST("/admin/login", adminAuth.Login)

		admin := v1.Group("/admin")
		admin.Use(adminAuth.AuthRequired())
		{
			admin.GET("/preset", apiHandler.GetAdminPreset)
			admin.PUT("/preset", apiHandler.UpdateAdminPreset)

			adminUserHandler := handler.NewAdminUserHandler(adminUserService)
			admin.GET("/users", adminUserHandler.GetAdminUsers)
			admin.POST("/users", adminUserHandler.CreateAdminUser)
			admin.PUT("/users/:id", adminUserHandler.UpdateAdminUser)
			admin.DELETE("/users/:id", adminUserHandler.DeleteAdminUser)

			admin.GET("/stats", apiHandler.GetAdminStats)
			admin.GET("/system/status", apiHandler.GetSystemStatus)
			admin.GET("/connections", apiHandler.GetConnectionStatus)
			admin.GET("/activity", apiHandler.GetActivity)
			admin.GET("/devices", apiHandler.GetAdminDevices)

			userHandler := handler.NewUserHandler(db)
			admin.GET("/user-list", userHandler.GetUsers)
			admin.GET("/user-list/:id", userHandler.GetUser)
			admin.POST("/user-list", userHandler.CreateUser)
			admin.PUT("/user-list/:id", userHandler.UpdateUser)
			admin.DELETE("/user-list/:id", userHandler.DeleteUser)
			admin.PUT("/user-list/:id/device-count", userHandler.UpdateUserDeviceCount)

			admin.GET("/device-bindings", userDeviceHandler.GetAllBindings)

			admin.GET("/settings", settingsHandler.GetSettings)
			admin.POST("/settings", settingsHandler.UpdateSettings)
		}
	}

	wsHandler.SetAPIKeyAuth(apiKeyAuth)
	router.GET("/signal/:device_id", wsHandler.HandleWebSocket)

	// User sync WebSocket (token-authenticated, no API key)
	router.GET("/api/v1/user/sync", wsHandler.HandleUserSync)

	// Legacy route for backward compatibility with existing tests
	router.GET("/host/:device_id", wsHandler.HandleWebSocket)
	router.GET("/client/:device_id/:access_code", func(c *gin.Context) {
		accessCode := c.Param("access_code")
		c.Request.URL.RawQuery = fmt.Sprintf("access_code=%s", accessCode)
		wsHandler.HandleWebSocket(c)
	})

	handler.RegisterAdminUI(router, signaling.WebDistFS)

	router.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/admin/")
	})

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
