// QuickDesk signaling server 鈥?v1 refactor entrypoint.
//
// See docs/dev/淇′护鏈嶅姟鍣ˋPI閲嶆瀯鏂规.md 搂2.2 for the canonical route table;
// this file is the wiring that implements it. Keep them in lock-step 鈥?
// when adding a route here, also update the doc.
package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"strings"

	signaling "quickdesk/signaling"
	"quickdesk/signaling/internal/config"
	"quickdesk/signaling/internal/database"
	"quickdesk/signaling/internal/handler"
	"quickdesk/signaling/internal/httpx"
	"quickdesk/signaling/internal/middleware"
	"quickdesk/signaling/internal/models"
	"quickdesk/signaling/internal/observability"
	"quickdesk/signaling/internal/repository"
	"quickdesk/signaling/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Version is the semver / git-describe tag injected at build time via
//
//	go build -ldflags "-X main.Version=$(git describe --tags --always)"
//
// Defaults to "dev" when unset (搂2.20).
var Version string

func main() {
	cfg := config.Load()
	logPath, err := observability.Configure(observability.Config{
		Dir:        cfg.Runtime.LogDir,
		MaxSizeMB:  cfg.Runtime.LogMaxSizeMB,
		MaxBackups: cfg.Runtime.LogMaxBackups,
	})
	if err != nil {
		log.Fatalf("configure application logging: %v", err)
	}
	log.Printf("Starting QuickDesk Signaling Server; persistent_log=%s", logPath)
	if cfg.Runtime.PprofAddr != "" {
		if !isLoopbackAddr(cfg.Runtime.PprofAddr) {
			log.Printf("pprof disabled: PPROF_ADDR must bind to loopback, got %q", cfg.Runtime.PprofAddr)
		} else {
			go func() {
				log.Printf("pprof listening on %s", cfg.Runtime.PprofAddr)
				if err := http.ListenAndServe(cfg.Runtime.PprofAddr, nil); err != nil {
					log.Printf("pprof server stopped: %v", err)
				}
			}()
		}
	}

	log.Println("Connecting to databases...")
	db := database.InitPostgreSQL(cfg)
	redisClient := database.InitRedis(cfg)

	if cfg.Runtime.RunAutoMigrate {
		log.Println("Running AutoMigrate (schema source of truth is migrations/001_init.sql)...")
		if err := db.AutoMigrate(
			&models.User{},
			&models.Device{},
			&models.UserDevice{},
			&models.ConnectionHistory{},
			&models.UserFavorite{},
			&models.AdminUser{},
			&models.AuditLog{},
			&models.Settings{},
			&models.DeviceGroup{},
			&models.DeviceGroupMember{},
			&models.Webhook{},
			&models.Preset{},
		); err != nil {
			log.Printf("Warning: AutoMigrate error (continuing anyway): %v", err)
		}
	} else {
		log.Println("AutoMigrate disabled (set RUN_AUTO_MIGRATE=true to enable).")
	}

	// -------------------------------------------------------------------
	// Services
	// -------------------------------------------------------------------
	settingsService := service.NewSettingsService(db)
	settingsService.SeedFromEnv(service.EnvSeed{
		TurnURLs:        strings.Join(cfg.Ice.TurnURLs, "\n"),
		TurnAuthSecret:  cfg.Ice.AuthSecret,
		TurnTTL:         cfg.Ice.CredentialTTL,
		StunURLs:        strings.Join(cfg.Ice.StunURLs, "\n"),
		APIKey:          cfg.Security.APIKey,
		AllowedOrigins:  strings.Join(cfg.Security.AllowedOrigins, "\n"),
		SmsKeyID:        cfg.Sms.AccessKeyID,
		SmsKeySecret:    cfg.Sms.AccessKeySecret,
		SmsSignName:     cfg.Sms.SignName,
		SmsTemplateCode: cfg.Sms.TemplateCode,
	})

	deviceRepo := repository.NewDeviceRepository(db)
	presetRepo := repository.NewPresetRepository(db)
	adminUserRepo := repository.NewAdminUserRepository(db)
	groupRepo := repository.NewDeviceGroupRepository(db)

	secrets := service.NewDeviceSecretService()
	deviceService := service.NewDeviceService(deviceRepo, secrets)
	presetService := service.NewPresetService(presetRepo)
	adminUserService := service.NewAdminUserService(adminUserRepo)
	userService := service.NewUserService(db, service.UserServiceDeps{DeviceUnbinder: deviceRepo})
	favoriteService := service.NewFavoriteService(db)
	connectionService := service.NewConnectionService(db)
	tokenService := service.NewTokenService(redisClient)
	rateLimitService := service.NewRateLimitService(redisClient)
	metricsService := service.NewMetricsService()
	bus := service.NewEventBus(redisClient)
	instanceID := uuid.NewString()
	presenceService := service.NewPresenceService(redisClient, instanceID)
	presenceService.Start(context.Background())
	smsService := service.NewSmsService(redisClient, settingsService)
	auditService := service.NewAuditService(db)
	webhookService := service.NewWebhookService(db)
	groupService := service.NewDeviceGroupService(groupRepo)

	// Subscribe system-scope event fanouts (realtime subscribes inside its
	// own constructor so it can receive user-scope events too).
	bus.Subscribe(service.NewWebhookSubscriber(webhookService))
	bus.Subscribe(service.NewAuditSubscriber(auditService))

	// Background worker that replays publish-failed events from the
	// outbox retry list (§2.17). Lives for the full process lifetime.
	bus.StartRetryWorker(context.Background())

	// Watch Redis keyspace notifications so heartbeat-TTL expiries
	// surface as device.online.changed events (§2.17). Requires Redis
	// server to be configured with `notify-keyspace-events Ex`.
	service.NewPresenceWatcher(redisClient, bus, presenceService, deviceRepo).
		Start(context.Background())

	// Bootstrap the initial admin user if needed.
	ctx := context.Background()
	if _, err := adminUserRepo.GetByUsername(ctx, cfg.Admin.User); err != nil {
		log.Printf("Creating initial admin user '%s'...", cfg.Admin.User)
		hashed, err := service.HashPassword(cfg.Admin.Password)
		if err != nil {
			log.Fatalf("hash initial admin password: %v", err)
		}
		initial := &models.AdminUser{
			Username: cfg.Admin.User,
			Password: hashed,
			Role:     "super_admin",
			Status:   true,
		}
		if err := adminUserRepo.Create(ctx, initial); err != nil {
			log.Fatalf("create initial admin user: %v", err)
		}
		log.Println("Initial admin user created successfully")
	}

	// -------------------------------------------------------------------
	// Handlers
	// -------------------------------------------------------------------
	publicHandler := handler.NewPublicHandler(db, redisClient, presetService, settingsService, smsService, Version)
	authHandler := handler.NewAuthHandler(userService, tokenService, smsService, bus)
	meHandler := handler.NewMeHandler(userService, tokenService, smsService, bus)
	deviceHandler := handler.NewDeviceHandler(deviceService, favoriteService, connectionService, presenceService, bus, db)
	hostHandler := handler.NewHostHandler(deviceService, tokenService, presenceService, settingsService, rateLimitService, bus, cfg)
	realtimeHandler := handler.NewRealtimeHandler(tokenService, bus, presenceService, db, deviceService, favoriteService, metricsService, redisClient)

	adminAuthHandler := handler.NewAdminAuthHandler(adminUserService, tokenService, auditService)
	adminTOTPHandler := handler.NewAdminTOTPHandler(adminUserService, db)
	adminAdminsHandler := handler.NewAdminAdminsHandler(adminUserService, tokenService, auditService)
	adminUsersHandler := handler.NewAdminUsersHandler(userService, tokenService, bus, auditService, presenceService, db)
	adminDevicesHandler := handler.NewAdminDevicesHandler(deviceService, presenceService, bus, auditService, db)
	adminSettingsHandler := handler.NewAdminSettingsHandler(settingsService, bus, auditService)
	adminPresetHandler := handler.NewAdminPresetHandler(presetService, auditService)
	adminAuditHandler := handler.NewAdminAuditHandler(auditService)
	adminLogsHandler := handler.NewAdminLogsHandler(
		auditService,
		db,
		cfg.Runtime.LogDir,
		cfg.Runtime.PostgresLogFile,
		cfg.Runtime.RedisLogFile,
	)
	adminWebhooksHandler := handler.NewAdminWebhooksHandler(webhookService, auditService)
	adminGroupsHandler := handler.NewAdminGroupsHandler(groupService, auditService)
	adminStatsHandler := handler.NewAdminStatsHandler(deviceService, presenceService, metricsService, db)

	// -------------------------------------------------------------------
	// Middleware
	// -------------------------------------------------------------------
	apiKeyAuth := middleware.NewAPIKeyAuth(settingsService)
	userAuth := middleware.NewUserAuth(tokenService)
	adminAuth := middleware.NewAdminAuth(tokenService)
	deviceAuth := middleware.NewDeviceAuth(deviceService)

	// -------------------------------------------------------------------
	// Router
	// -------------------------------------------------------------------
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.RequestID())
	router.Use(middleware.LoggerMiddleware(metricsService))
	router.Use(middleware.CORSMiddleware(settingsService))

	// Health + public surface.
	router.GET("/health", publicHandler.Health)

	// Every v1 endpoint — including public read-only ones — sits behind
	// APIKeyAuth.Required() (§2.1 "所有调用官方服务器的接口带 header").
	// When QUICKDESK_API_KEY isn't configured server-side, apikey.go's
	// Enabled() returns false and the middleware becomes a no-op, so a
	// self-hosted deployment without the gate still works.
	//
	// The only routes outside this group are /health (unauthenticated by
	// design for k8s readiness probes) and the two /v1/realtime/* WS
	// upgrades — browsers can't attach custom headers to a WebSocket
	// handshake, so those authenticate via first-frame auth instead
	// (§2.13).
	v1 := router.Group("/v1")
	v1.Use(apiKeyAuth.Required())
	{
		v1.GET("/preset", publicHandler.Preset)
		v1.GET("/settings/public", publicHandler.PublicSettings)
		v1.GET("/features", publicHandler.Features)
		v1.POST("/verification-codes", publicHandler.SendVerificationCode)

		// Auth (public; no user token yet, but API key still required
		// when configured — see note above).
		auth := v1.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/sessions", authHandler.CreateSession)
			auth.POST("/sessions:sms", authHandler.CreateSessionSms)
			auth.POST("/tokens:refresh", authHandler.RefreshToken)
			auth.POST("/password-resets", authHandler.RequestPasswordReset)
			auth.POST("/password-resets:confirm", authHandler.ConfirmPasswordReset)
		}

		// Current user (requires access_token in addition to api key).
		me := v1.Group("/me")
		me.Use(userAuth.Required())
		{
			me.GET("", meHandler.Get)
			me.PUT("/password", meHandler.ChangePassword)
			me.PUT("/username", meHandler.ChangeUsername)
			me.PUT("/phone", meHandler.ChangePhone)
			me.PUT("/email", meHandler.ChangeEmail)

			me.GET("/sessions", meHandler.ListSessions)
			me.DELETE("/sessions/current", meHandler.DeleteCurrentSession)
			me.DELETE("/sessions/:session_id", meHandler.DeleteSessionByID)

			me.GET("/devices", deviceHandler.ListMine)
			me.POST("/devices", deviceHandler.Bind)
			me.GET("/devices/:device_id", deviceHandler.GetOne)
			me.PATCH("/devices/:device_id", deviceHandler.Patch)
			me.DELETE("/devices/:device_id", deviceHandler.Unbind)
			me.DELETE("/devices/:device_id/session", deviceHandler.ClearSession)

			me.GET("/connections", deviceHandler.ListConnections)
			me.POST("/connections", deviceHandler.RecordConnection)

			me.GET("/favorites", deviceHandler.ListFavorites)
			me.POST("/favorites", deviceHandler.AddFavorite)
			me.PATCH("/favorites/:device_id", deviceHandler.UpdateFavorite)
			me.DELETE("/favorites/:device_id", deviceHandler.DeleteFavorite)
		}

		// -----------------------------------------------------------------
		// Device-side surface.
		// -----------------------------------------------------------------

		// provision only needs X-API-Key (no device_secret yet).
		v1.POST("/devices:provision", hostHandler.Provision)

		// Device-secret-protected endpoints (heartbeat, signal-tokens,
		// access-code PUT). device_secret auth composes on top of the
		// v1-wide X-API-Key gate per §2.2.
		dev := v1.Group("/devices/:device_id")
		dev.Use(deviceAuth.Required())
		{
			dev.POST("/heartbeat", hostHandler.Heartbeat)
			dev.POST("/signal-tokens", hostHandler.IssueHostSignalToken)
			dev.PUT("/access-code", hostHandler.SetAccessCode)
		}

		// verify: X-API-Key OR Origin whitelist — apikey.Required()
		// already implements the OR logic (§2.2 H1).
		v1.POST("/devices/:device_id/access-code:verify", hostHandler.VerifyAccessCode)

		// ice-config: X-API-Key (hosts additionally present Bearer
		// device_secret, but apikey.Required() already covers both paths
		// via the Origin-whitelist fallback).
		v1.GET("/ice-config", hostHandler.GetICEConfig)

	}

	// -------------------------------------------------------------------
	// Admin surface — registered outside the v1 API-key-protected group.
	// Admin endpoints authenticate via admin JWT tokens (adminAuth), not
	// via X-API-Key. This avoids a chicken-and-egg problem: the admin
	// web UI is where API keys/allowed-origins are configured, so it
	// cannot itself require an API key to be accessible.
	// -------------------------------------------------------------------
	adminAuthGroup := router.Group("/v1/admin/auth")
	{
		adminAuthGroup.POST("/sessions", adminAuthHandler.CreateSession)
		adminAuthGroup.POST("/sessions:totp", adminAuthHandler.CreateSessionFromTOTP)
		adminAuthGroup.POST("/tokens:refresh", adminAuthHandler.RefreshToken)
		// Logout requires an admin token.
		adminAuthGroup.DELETE("/sessions/current", adminAuth.Required(), adminAuthHandler.DeleteCurrentSession)
	}

	admin := router.Group("/v1/admin")
	admin.Use(adminAuth.Required())
	admin.Use(middleware.IPWhitelistMiddleware(settingsService))
	{
		// 2FA enrollment endpoints only require a valid admin session. 2FA
		// is optional for editing; when enabled, it is enforced at login.
		admin.POST("/admins/me/2fa/setup", adminTOTPHandler.Setup)
		admin.POST("/admins/me/2fa/verify", adminTOTPHandler.Verify)
		admin.DELETE("/admins/me/2fa", adminTOTPHandler.Delete)
	}

	// All other admin endpoints require admin auth. 2FA is optional: if an
	// admin enables it, login requires TOTP; writes are not blocked just
	// because 2FA is disabled.
	adminGuarded := router.Group("/v1/admin")
	adminGuarded.Use(adminAuth.Required())
	adminGuarded.Use(middleware.IPWhitelistMiddleware(settingsService))
	{
		// Admin accounts (CRUD over the admin users themselves).
		adminGuarded.GET("/admins", adminAdminsHandler.List)
		adminGuarded.POST("/admins", adminAdminsHandler.Create)
		adminGuarded.GET("/admins/:id", adminAdminsHandler.Get)
		adminGuarded.PATCH("/admins/:id", adminAdminsHandler.Patch)
		adminGuarded.DELETE("/admins/:id", adminAdminsHandler.Delete)

		// Business users.
		adminGuarded.GET("/users", adminUsersHandler.List)
		adminGuarded.POST("/users", adminUsersHandler.Create)
		adminGuarded.POST("/users:batch", adminUsersHandler.Batch)
		adminGuarded.GET("/users/:id", adminUsersHandler.Get)
		adminGuarded.GET("/users/:id/details", adminUsersHandler.GetDetails)
		adminGuarded.PATCH("/users/:id", adminUsersHandler.Patch)
		adminGuarded.DELETE("/users/:id", adminUsersHandler.Delete)
		adminGuarded.POST("/users/:id/sessions:revoke", adminUsersHandler.RevokeSessions)
		adminGuarded.PATCH("/users/:id/device-count", adminUsersHandler.PatchDeviceCount)

		// Devices.
		adminGuarded.GET("/devices", adminDevicesHandler.List)
		adminGuarded.GET("/devices/:device_id", adminDevicesHandler.Get)
		adminGuarded.DELETE("/devices/:device_id", adminDevicesHandler.Delete)
		adminGuarded.POST("/devices/:device_id/unbind", adminDevicesHandler.ForceUnbind)
		adminGuarded.POST("/devices/:device_id/secret:rotate", adminDevicesHandler.RotateSecret)
		adminGuarded.POST("/devices:batch", func(c *gin.Context) {
			adminDevicesHandler.Batch(c, groupService)
		})

		// User → device bindings.
		adminGuarded.GET("/device-bindings", adminDevicesHandler.ListBindings)

		// Stats / observability.
		adminGuarded.GET("/stats", adminStatsHandler.GetStats)
		adminGuarded.GET("/system/status", adminStatsHandler.GetSystemStatus)
		adminGuarded.GET("/connections", adminStatsHandler.GetConnections)
		adminGuarded.GET("/activity", adminStatsHandler.GetActivity)
		adminGuarded.GET("/trends", adminStatsHandler.GetTrends)

		// Audit logs.
		adminGuarded.GET("/audit-logs", adminAuditHandler.List)
		adminGuarded.GET("/audit-logs:export", adminLogsHandler.ExportAudit)

		// Raw operational log downloads are intentionally restricted to
		// super administrators and a fixed server-side file allow-list.
		adminGuarded.GET("/system/logs", adminLogsHandler.List)
		adminGuarded.GET("/system/logs/:name", adminLogsHandler.Download)

		// Preset.
		adminGuarded.GET("/preset", adminPresetHandler.Get)
		adminGuarded.PUT("/preset", adminPresetHandler.Update)

		// Settings.
		adminGuarded.GET("/settings", adminSettingsHandler.Get)
		adminGuarded.PUT("/settings", adminSettingsHandler.Update)

		// Webhooks.
		adminGuarded.GET("/webhooks", adminWebhooksHandler.List)
		adminGuarded.POST("/webhooks", adminWebhooksHandler.Create)
		adminGuarded.GET("/webhooks/:id", adminWebhooksHandler.Get)
		adminGuarded.PATCH("/webhooks/:id", adminWebhooksHandler.Patch)
		adminGuarded.DELETE("/webhooks/:id", adminWebhooksHandler.Delete)
		// §2.2: webhook delivery test. We avoid the AIP-136
		// "{name}:test" custom-method form here because gin's
		// httprouter cannot register a literal suffix on a
		// wildcard segment ("only one wildcard per path segment is
		// allowed"). Treating the test delivery as a sub-resource
		// is RESTful, gin-friendly and clients still POST against
		// /webhooks/:id/test which reads naturally.
		adminGuarded.POST("/webhooks/:id/test", adminWebhooksHandler.Test)

		// Device groups.
		adminGuarded.GET("/groups", adminGroupsHandler.List)
		adminGuarded.POST("/groups", adminGroupsHandler.Create)
		adminGuarded.PATCH("/groups/:id", adminGroupsHandler.Patch)
		adminGuarded.DELETE("/groups/:id", adminGroupsHandler.Delete)
		adminGuarded.POST("/groups/:id/devices", adminGroupsHandler.AddDevices)
		adminGuarded.DELETE("/groups/:id/devices", adminGroupsHandler.RemoveDevices)
		adminGuarded.GET("/groups/:id/devices", adminGroupsHandler.ListDevices)
	}

	// -------------------------------------------------------------------
	// Realtime WebSockets — registered outside the v1 group so they
	// don't inherit APIKeyAuth.Required() (browsers can't attach custom
	// headers to a WS upgrade). First-frame auth authenticates them
	// instead (§2.13).
	// -------------------------------------------------------------------
	router.GET("/v1/realtime/events", realtimeHandler.HandleEvents)
	router.GET("/v1/realtime/signal", realtimeHandler.HandleSignal)

	// -------------------------------------------------------------------
	// Admin web single-page app (served from embedded FS).
	// -------------------------------------------------------------------
	handler.RegisterAdminUI(router, signaling.WebDistFS)

	router.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/admin/")
	})

	// -------------------------------------------------------------------
	// Go!
	// -------------------------------------------------------------------
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("Server starting on %s (version=%s, instance=%s)", addr, Version, instanceID)
	log.Printf("API: http://%s/v1 (admin UI at /admin/)", addr)
	log.Printf("Realtime WebSockets: ws://%s/v1/realtime/{events,signal}", addr)
	log.Println("Ready to accept connections.")
	if err := router.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func isLoopbackAddr(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// ensure httpx is imported 鈥?used transitively via middleware + handler.
var _ = httpx.CodeUnauthorized
