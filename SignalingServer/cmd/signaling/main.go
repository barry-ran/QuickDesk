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
	"time"
	// Embed the IANA timezone database: device access windows are evaluated in
	// a named zone, and time.LoadLocation needs tzdata on hosts that do not
	// ship it (Windows, scratch containers).
	_ "time/tzdata"

	signaling "quickdesk/signaling"
	"quickdesk/signaling/internal/config"
	"quickdesk/signaling/internal/database"
	"quickdesk/signaling/internal/handler"
	"quickdesk/signaling/internal/holiday"
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
		// Migrated one model at a time on purpose. AutoMigrate returns on the
		// first failure, so a single table that cannot be altered (a new
		// NOT NULL column over rows that are still NULL is the usual cause)
		// would silently skip every model after it in the list. Per-model
		// calls keep one bad table from blocking the rest, and the log names
		// the table that needs attention.
		for _, model := range []interface{}{
			&models.User{},
			&models.Device{},
			&models.UserDevice{},
			&models.ConnectionHistory{},
			&models.UserFavorite{},
			&models.SessionRecording{},
			&models.RemoteSession{},
			&models.AdminUser{},
			&models.AuditLog{},
			&models.Settings{},
			&models.DeviceGroup{},
			&models.DeviceGroupMember{},
			&models.DeviceAccessLog{},
			&models.DeviceAccessPolicy{},
			&models.DeviceAccessGrant{},
			&models.DeviceAccessApproval{},
			&models.Webhook{},
			&models.UserNotificationChannel{},
			&models.NotificationDelivery{},
			&models.NotificationQuietHours{},
			&models.DeviceOfflineAlertRule{},
			&models.ScheduledTask{},
			&models.InstanceLogEntry{},
			&models.UpdateRelease{},
			&models.Preset{},
		} {
			if err := db.AutoMigrate(model); err != nil {
				log.Printf("Warning: AutoMigrate %T failed (continuing with the remaining tables): %v", model, err)
			}
		}
		// AutoMigrate manages columns, not expression indexes; the log search
		// indexes are ensured separately and may legitimately be unavailable.
		database.EnsureLogSearchIndexes(context.Background(), db)
	} else {
		log.Println("AutoMigrate disabled (set RUN_AUTO_MIGRATE=true to enable).")
	}

	// -------------------------------------------------------------------
	// Services
	// -------------------------------------------------------------------
	// Statutory holiday calendar for device access policy exception dates.
	// Loading it here (rather than lazily at first evaluate) means a broken
	// override file fails at startup, not on someone's first connection.
	//
	// When HOLIDAY_CALENDAR_URL is set the year files are fetched before that
	// first load, so a deployment gets next year's notice without anybody
	// copying it in. A failed fetch is logged and never fatal: the embedded
	// year and whatever is already on disk stay in use.
	holidayURL := strings.TrimSpace(cfg.Holiday.URL)
	var holidayFetcher *holiday.Fetcher
	if holidayURL != "" {
		if strings.TrimSpace(cfg.Holiday.Dir) == "" {
			log.Fatalf("HOLIDAY_CALENDAR_URL is set but HOLIDAY_CALENDAR_DIR is empty; the fetcher needs a cache directory")
		}
		holidayFetcher = holiday.NewFetcher(holidayURL, cfg.Holiday.Dir)
		changed, err := holidayFetcher.Refresh(context.Background())
		if err != nil {
			log.Printf("Holiday calendar fetch from %s failed (keeping the embedded and cached files): %v",
				holidayURL, err)
		}
		if len(changed) > 0 {
			log.Printf("Holiday calendar: fetched year files %v into %q", changed, cfg.Holiday.Dir)
		}
	}
	holidayCalendar, err := holiday.Load(cfg.Holiday.Dir)
	if err != nil {
		if holidayFetcher == nil {
			log.Fatalf("load holiday calendar (HOLIDAY_CALENDAR_DIR=%q): %v", cfg.Holiday.Dir, err)
		}
		// A deployment that fetches its calendars can legitimately have an empty
		// cache on the first boot - the directory exists, the network did not.
		// The embedded year is a usable calendar, so start with it and let the
		// next refresh fill the gap; refusing to boot over a flaky first fetch
		// would be the worse failure.
		log.Printf("Holiday calendar: no cached files yet (%v); using the embedded year until a fetch succeeds", err)
		holidayCalendar, err = holiday.Load("")
		if err != nil {
			log.Fatalf("load embedded holiday calendar: %v", err)
		}
	}
	service.SetHolidayCalendar(holidayCalendar)
	log.Printf("Holiday calendar: %d dates across years %v (override dir=%q)",
		holidayCalendar.Len(), holidayCalendar.Years(), cfg.Holiday.Dir)
	if holidayFetcher != nil && cfg.Holiday.RefreshMinutes > 0 {
		go holidayFetcher.Run(context.Background(),
			time.Duration(cfg.Holiday.RefreshMinutes)*time.Minute,
			func(changed []int) {
				reloaded, err := holiday.Load(cfg.Holiday.Dir)
				if err != nil {
					log.Printf("[holiday] reloading the calendar after a fetch failed: %v", err)
					return
				}
				service.SetHolidayCalendar(reloaded)
				log.Printf("Holiday calendar refreshed (years %v): %d dates across years %v",
					changed, reloaded.Len(), reloaded.Years())
			})
		log.Printf("Holiday calendar refresh: every %dm from %s", cfg.Holiday.RefreshMinutes, holidayURL)
	}

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
	deviceAccessService := service.NewDeviceAccessService(db)
	deviceAccessPolicyService := service.NewDeviceAccessPolicyService(db)
	deviceAccessGrantService := service.NewDeviceAccessGrantService(db)
	deviceAccessApprovalService := service.NewDeviceAccessApprovalService(db)
	if cfg.Approval.DenialCooldownSeconds > 0 {
		deviceAccessApprovalService.WithDenialCooldown(
			time.Duration(cfg.Approval.DenialCooldownSeconds) * time.Second)
		deviceAccessApprovalService.WithDenialCooldownEscalation(
			cfg.Approval.DenialCooldownMultiplier, cfg.Approval.DenialCooldownMaxSeconds)
		if multiplier := deviceAccessApprovalService.DenialCooldownMultiplier(); multiplier > 1 {
			log.Printf(
				"Approval denial cooldown: %ds per denial, x%d per consecutive denial, capped at %ds",
				cfg.Approval.DenialCooldownSeconds, multiplier, cfg.Approval.DenialCooldownMaxSeconds)
		} else {
			log.Printf("Approval denial cooldown: %ds", cfg.Approval.DenialCooldownSeconds)
		}
	}
	notificationDeliveryService := service.NewNotificationDeliveryService(db)
	notificationQuietHoursService := service.NewNotificationQuietHoursService(db)
	// The worker asks the setting before every post, so a snooze or a
	// maintenance window holds alerts instead of delivering them late.
	notificationDeliveryService.WithQuietHours(notificationQuietHoursService)
	if cfg.Notification.ArchiveDir != "" {
		notificationDeliveryService.WithArchiveDir(cfg.Notification.ArchiveDir)
	}
	if cfg.Notification.ArchiveColdDir != "" {
		notificationDeliveryService.WithArchiveColdDir(cfg.Notification.ArchiveColdDir)
	}
	if cfg.Notification.AggregateSeconds > 0 {
		notificationDeliveryService.WithAggregateWindow(
			time.Duration(cfg.Notification.AggregateSeconds) * time.Second)
		log.Printf("Notification aggregation: window=%ds (delivery is held up to this long)",
			cfg.Notification.AggregateSeconds)
	}
	notificationChannelService := service.NewNotificationChannelService(db, notificationDeliveryService)
	sessionRecordingService := service.NewSessionRecordingService(db)
	remoteSessionService := service.NewRemoteSessionService(db)
	// A signaling socket cannot outlive the process that held it, so anything
	// still open at boot ended with the previous process, without the handler's
	// deferred close ever running. Stamping those rows is what keeps "who is in
	// this machine right now" honest across a restart.
	if closed, err := remoteSessionService.CloseOpenSessions(context.Background(), models.RemoteSessionEndServerRestart); err != nil {
		log.Printf("Warning: closing remote sessions left open by the previous process failed: %v", err)
	} else if closed > 0 {
		log.Printf("Remote sessions: closed %d left open by the previous process", closed)
	}
	recordingLifecycle := service.NewRecordingLifecycle(
		sessionRecordingService,
		cfg.Recording.Dir,
		time.Duration(cfg.Recording.StaleMinutes)*time.Minute,
		time.Duration(cfg.Recording.RetentionDays)*24*time.Hour,
		time.Duration(cfg.Recording.SweepIntervalMin)*time.Minute,
	)
	if cfg.Recording.SweepIntervalMin > 0 {
		go recordingLifecycle.Run(context.Background())
		log.Printf("Recording lifecycle: stale_after=%dm retention_days=%d sweep_interval=%dm",
			cfg.Recording.StaleMinutes, cfg.Recording.RetentionDays, cfg.Recording.SweepIntervalMin)
	}
	go notificationDeliveryService.Run(context.Background())
	if cfg.Notification.SweepIntervalMin > 0 &&
		(cfg.Notification.RetentionDays > 0 || cfg.Notification.ArchiveRetentionDays > 0) {
		go notificationDeliveryService.RunRetention(
			context.Background(),
			time.Duration(cfg.Notification.SweepIntervalMin)*time.Minute,
			cfg.Notification.RetentionDays,
			cfg.Notification.ArchiveRetentionDays,
		)
		archive := cfg.Notification.ArchiveDir
		if archive == "" {
			archive = "(delete, no archive)"
		}
		log.Printf("Notification retention: retention_days=%d archive_retention_days=%d sweep_interval=%dm archive=%s cold_archive=%s",
			cfg.Notification.RetentionDays, cfg.Notification.ArchiveRetentionDays,
			cfg.Notification.SweepIntervalMin, archive, cfg.Notification.ArchiveColdDir)
	}
	tokenService := service.NewTokenService(redisClient)
	rateLimitService := service.NewRateLimitService(redisClient)
	metricsService := service.NewMetricsService()
	bus := service.NewEventBus(redisClient)
	instanceID := uuid.NewString()
	presenceService := service.NewPresenceService(redisClient, instanceID)
	presenceService.Start(context.Background())
	// Cross-instance log shipping. Attaching the sink after the database is open
	// (rather than inside observability.Configure) is what makes it possible at
	// all: the store is the database. Lines logged before this point - the
	// startup banner included - are only in the local file, which the docs say.
	var instanceLogSink *service.InstanceLogSink
	if cfg.InstanceLog.Sink {
		instanceLogSink = service.NewInstanceLogSink(db, instanceID)
		observability.AttachLogSink(instanceLogSink)
		go instanceLogSink.Run(context.Background())
		go instanceLogSink.RunRetention(context.Background(),
			time.Duration(cfg.InstanceLog.SweepIntervalMin)*time.Minute,
			cfg.InstanceLog.RetentionDays)
		log.Printf("Instance log aggregation: sink=on retention_days=%d sweep_interval=%dm buffer=%d instance=%s",
			cfg.InstanceLog.RetentionDays, cfg.InstanceLog.SweepIntervalMin,
			service.DefaultInstanceLogBufferLines, instanceID)
		// The application log ships itself through the attached writer; the
		// PostgreSQL and Redis logs are files nobody writes through this process,
		// so they are tailed. Without this, `scope=all` covers the application
		// log of every instance and the database logs of none.
		for _, followed := range []struct{ source, path string }{
			{service.PostgresLogSourceName, cfg.Runtime.PostgresLogFile},
			{service.RedisLogSourceName, cfg.Runtime.RedisLogFile},
		} {
			if strings.TrimSpace(followed.path) == "" {
				continue
			}
			tailer := service.NewFileLogTailer(instanceLogSink, followed.path, followed.source)
			go tailer.Run(context.Background())
			log.Printf("Instance log aggregation: tailing %s (source=%s) from the end of the file",
				followed.path, followed.source)
		}
	}
	smsService := service.NewSmsService(redisClient, settingsService)
	auditService := service.NewAuditService(db).
		WithArchiveDir(cfg.Audit.ArchiveDir).
		WithArchiveColdDir(cfg.Audit.ArchiveColdDir)
	if cfg.Audit.SweepIntervalMin > 0 &&
		(cfg.Audit.RetentionDays > 0 || cfg.Audit.MaxRows > 0 ||
			cfg.Audit.MaxRowsPerActor > 0 || cfg.Audit.MaxRowsPerDevice > 0 ||
			cfg.Audit.MaxRowsPerAction > 0 || cfg.Audit.ArchiveRetentionDays > 0) {
		go auditService.RunRetention(
			context.Background(),
			time.Duration(cfg.Audit.SweepIntervalMin)*time.Minute,
			cfg.Audit.RetentionDays,
			cfg.Audit.MaxRows,
			cfg.Audit.MaxRowsPerActor,
			cfg.Audit.MaxRowsPerDevice,
			cfg.Audit.MaxRowsPerAction,
			cfg.Audit.ArchiveRetentionDays,
		)
		archive := cfg.Audit.ArchiveDir
		if archive == "" {
			archive = "(delete, no archive)"
		}
		log.Printf("Audit retention: retention_days=%d max_rows=%d max_rows_per_actor=%d max_rows_per_device=%d max_rows_per_action=%d archive_retention_days=%d sweep_interval=%dm archive=%s cold_archive=%s",
			cfg.Audit.RetentionDays, cfg.Audit.MaxRows, cfg.Audit.MaxRowsPerActor,
			cfg.Audit.MaxRowsPerDevice, cfg.Audit.MaxRowsPerAction, cfg.Audit.ArchiveRetentionDays,
			cfg.Audit.SweepIntervalMin, archive, cfg.Audit.ArchiveColdDir)
	}
	webhookService := service.NewWebhookService(db)
	groupService := service.NewDeviceGroupService(groupRepo)

	// Owner-scheduled control-plane actions. Only kinds whose whole effect
	// lives in the control plane are registered: a task that needed a client to
	// cooperate could be scheduled but never actually run.
	scheduledTaskService := service.NewScheduledTaskService(db).
		WithAudit(auditService).
		WithNotifier(bus)
	scheduledTaskService.RegisterAction(
		models.ScheduledTaskSignOutAllSessions,
		service.NewSignOutAllSessionsAction(tokenService, bus),
	)
	if cfg.ScheduledTask.IntervalSeconds > 0 {
		go scheduledTaskService.Run(
			context.Background(),
			time.Duration(cfg.ScheduledTask.IntervalSeconds)*time.Second,
		)
		log.Printf("Scheduled tasks: interval=%ds kinds=%v",
			cfg.ScheduledTask.IntervalSeconds, scheduledTaskService.SupportedKinds())
	}

	// Subscribe system-scope event fanouts (realtime subscribes inside its
	// own constructor so it can receive user-scope events too).
	bus.Subscribe(service.NewWebhookSubscriber(webhookService))
	bus.Subscribe(service.NewAuditSubscriber(auditService))
	bus.Subscribe(service.NewUserNotificationSubscriber(notificationChannelService))

	// Background worker that replays publish-failed events from the
	// outbox retry list (§2.17). Lives for the full process lifetime.
	bus.StartRetryWorker(context.Background())

	// Watch Redis keyspace notifications so heartbeat-TTL expiries
	// surface as device.online.changed events (§2.17). Requires Redis
	// server to be configured with `notify-keyspace-events Ex`.
	service.NewPresenceWatcher(redisClient, bus, presenceService, deviceRepo).
		Start(context.Background())

	// Per-device offline alerts. Presence says "the heartbeat stopped"; this
	// evaluator adds the part an owner actually asked for - how long - and
	// publishes through the user notification outbox, so the alert inherits
	// durable delivery, retry and dead-letter replay.
	deviceOfflineAlertService := service.NewDeviceOfflineAlertService(db, presenceService, bus)
	if cfg.OfflineAlert.IntervalSeconds > 0 {
		go deviceOfflineAlertService.Run(
			context.Background(),
			time.Duration(cfg.OfflineAlert.IntervalSeconds)*time.Second,
		)
		log.Printf("Offline alerts: enabled evaluate_interval=%ds", cfg.OfflineAlert.IntervalSeconds)
	}

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
	deviceHandler := handler.NewDeviceHandler(deviceService, favoriteService, connectionService, presenceService, deviceAccessService, bus, db)
	notificationChannelHandler := handler.NewNotificationChannelHandler(notificationChannelService, notificationDeliveryService)
	notificationQuietHoursHandler := handler.NewNotificationQuietHoursHandler(
		notificationQuietHoursService, notificationDeliveryService)
	deviceAccessPolicyHandler := handler.NewDeviceAccessPolicyHandler(deviceService, deviceAccessPolicyService, bus)
	deviceOfflineAlertHandler := handler.NewDeviceOfflineAlertHandler(deviceService, deviceOfflineAlertService)
	updateReleaseService := service.NewUpdateReleaseService(db)
	updateReleaseHandler := handler.NewUpdateReleaseHandler(updateReleaseService)
	updateArtifactLifecycle := service.NewUpdateArtifactLifecycle(
		updateReleaseService,
		cfg.Update.ArtifactDir,
		time.Duration(cfg.Update.StaleUploadMinutes)*time.Minute,
		time.Duration(cfg.Update.SweepIntervalMin)*time.Minute,
	)
	if cfg.Update.SweepIntervalMin > 0 {
		go updateArtifactLifecycle.Run(context.Background())
		log.Printf("Update artifacts: stale_upload_after=%dm sweep_interval=%dm dir=%s",
			cfg.Update.StaleUploadMinutes, cfg.Update.SweepIntervalMin, cfg.Update.ArtifactDir)
	}
	adminUpdateReleaseHandler := handler.NewAdminUpdateReleaseHandler(updateReleaseService, auditService,
		handler.UpdateArtifactStore{
			Dir:           cfg.Update.ArtifactDir,
			MaxBytes:      int64(cfg.Update.MaxUploadMB) * 1024 * 1024,
			PublicBaseURL: cfg.Update.PublicBaseURL,
		}).WithLifecycle(updateArtifactLifecycle)
	scheduledTaskHandler := handler.NewScheduledTaskHandler(scheduledTaskService)
	deviceAccessGrantHandler := handler.NewDeviceAccessGrantHandler(deviceService, deviceAccessGrantService, bus)
	deviceAccessApprovalHandler := handler.NewDeviceAccessApprovalHandler(
		deviceService, deviceAccessApprovalService, deviceAccessGrantService,
		tokenService, presenceService, deviceAccessService, bus)
	sessionRecordingHandler := handler.NewSessionRecordingHandler(sessionRecordingService, bus, handler.SessionRecordingStore{
		Dir:       cfg.Recording.Dir,
		MaxBytes:  int64(cfg.Recording.MaxUploadMB) * 1024 * 1024,
		URLSecret: cfg.Recording.URLSecret,
		TicketTTL: time.Duration(cfg.Recording.TicketTTLSeconds) * time.Second,
	}, auditService)
	hostHandler := handler.NewHostHandler(deviceService, tokenService, presenceService, settingsService, rateLimitService, deviceAccessService, deviceAccessPolicyService, deviceAccessGrantService, deviceAccessApprovalService, bus, cfg)
	realtimeHandler := handler.NewRealtimeHandler(tokenService, bus, presenceService, db, deviceService, favoriteService, metricsService, deviceAccessService, deviceAccessPolicyService, remoteSessionService, redisClient)
	// realtimeHandler is built first because the session surfaces use it to close
	// a live socket: a revoke has to reach the connection, not just the row.
	remoteSessionHandler := handler.NewRemoteSessionHandler(remoteSessionService, sessionRecordingService, deviceAccessService, deviceService, realtimeHandler, auditService)

	adminAuthHandler := handler.NewAdminAuthHandler(adminUserService, tokenService, auditService)
	adminTOTPHandler := handler.NewAdminTOTPHandler(adminUserService, db)
	adminAdminsHandler := handler.NewAdminAdminsHandler(adminUserService, tokenService, auditService)
	adminUsersHandler := handler.NewAdminUsersHandler(userService, tokenService, bus, auditService, presenceService, db)
	adminDevicesHandler := handler.NewAdminDevicesHandler(deviceService, presenceService, bus, auditService, db)
	adminSettingsHandler := handler.NewAdminSettingsHandler(settingsService, bus, auditService)
	adminPresetHandler := handler.NewAdminPresetHandler(presetService, auditService)
	adminAuditHandler := handler.NewAdminAuditHandler(
		auditService, cfg.Audit.RetentionDays, cfg.Audit.MaxRows, cfg.Audit.MaxRowsPerActor,
		cfg.Audit.MaxRowsPerDevice, cfg.Audit.MaxRowsPerAction, cfg.Audit.ArchiveRetentionDays)
	adminSessionRecordingHandler := handler.NewAdminSessionRecordingHandler(
		sessionRecordingService, auditService, recordingLifecycle,
		cfg.Recording.RetentionDays, cfg.Recording.StaleMinutes)
	adminRemoteSessionHandler := handler.NewAdminRemoteSessionHandler(
		remoteSessionService, sessionRecordingService, deviceAccessService, realtimeHandler, auditService)
	adminNotificationDeliveryHandler := handler.NewAdminNotificationDeliveryHandler(
		notificationDeliveryService, auditService,
		cfg.Notification.RetentionDays, cfg.Notification.ArchiveRetentionDays)
	// Cross-instance search is only offered when this instance is shipping into
	// the shared store; otherwise `scope=all` would read a table nobody fills.
	var aggregatedLogs *service.AggregatedLogQueryService
	if instanceLogSink != nil {
		aggregatedLogs = service.NewAggregatedLogQueryService(db).WithSink(instanceLogSink)
	}
	adminLogsHandler := handler.NewAdminLogsHandler(
		auditService,
		db,
		cfg.Runtime.LogDir,
		cfg.Runtime.PostgresLogFile,
		cfg.Runtime.RedisLogFile,
		service.NewLogQueryService(cfg.Runtime.LogDir),
		aggregatedLogs,
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
	metricsHandler := handler.NewMetricsHandler(metricsService, Version, cfg.Metrics.Token)
	router := gin.New()
	// Only peers on this list may set X-Forwarded-For; every other request is
	// attributed to its socket address. Without the call gin believes the
	// header from any peer, which hands every IP-based gate - allow_ips,
	// approval_trusted_ips, the per-IP rate limits, the access log - to the
	// client. An invalid entry is fatal rather than ignored: falling back
	// silently would leave the gates spoofable while looking configured.
	if err := router.SetTrustedProxies(cfg.Security.TrustedProxies); err != nil {
		log.Fatalf("TRUSTED_PROXIES is invalid: %v", err)
	}
	if len(cfg.Security.TrustedProxies) == 0 {
		log.Println("Trusted proxies: none (X-Forwarded-For is ignored everywhere)")
	} else {
		log.Printf("Trusted proxies: %s", strings.Join(cfg.Security.TrustedProxies, ", "))
	}
	router.Use(gin.Recovery())
	router.Use(middleware.RequestID())
	router.Use(middleware.LoggerMiddleware(metricsService))
	router.Use(middleware.CORSMiddleware(settingsService))

	// Health + public surface.
	router.GET("/health", publicHandler.Health)
	if cfg.Metrics.Token != "" {
		router.GET("/metrics", metricsHandler.Prometheus)
		log.Println("Prometheus metrics: enabled at /metrics (bearer token required)")
	}

	// Every v1 endpoint — including public read-only ones — sits behind
	// APIKeyAuth.Required() (§2.1 "所有调用官方服务器的接口带 header").
	// When QUICKDESK_API_KEY isn't configured server-side, apikey.go's
	// Enabled() returns false and the middleware becomes a no-op, so a
	// self-hosted deployment without the gate still works.
	//
	// The only routes outside this group are /health (unauthenticated by
	// design for k8s readiness probes), the opt-in /metrics endpoint
	// (bearer-token protected) and the two /v1/realtime/* WS upgrades —
	// browsers can't attach custom headers to a WebSocket handshake, so
	// those authenticate via first-frame auth instead (§2.13).
	v1 := router.Group("/v1")
	v1.Use(apiKeyAuth.Required())
	{
		v1.GET("/preset", publicHandler.Preset)
		v1.GET("/settings/public", publicHandler.PublicSettings)
		v1.GET("/features", publicHandler.Features)
		v1.POST("/verification-codes", publicHandler.SendVerificationCode)
		// Self-hosted update channel: the index the installed clients poll.
		// API-key gated like every other v1 route, but not user-authenticated -
		// a controlled machine checks for updates before anyone logs in.
		v1.GET("/update:check", updateReleaseHandler.Check)
		// Installers this server hosts itself. Same gate as the check: the
		// client already holds the API key by the time it downloads a build,
		// so no ticket or browser navigation is involved.
		v1.GET("/update-releases/:id/artifact", updateReleaseHandler.DownloadArtifact)

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
			me.GET("/devices/:device_id/access-logs", deviceHandler.ListAccessLogs)
			// The same history with the session lifecycle attached: each entry is
			// a session plus the recordings and access-log rows that share its id.
			me.GET("/devices/:device_id/sessions", remoteSessionHandler.ListForDevice)
			me.GET("/devices/:device_id/access-policy", deviceAccessPolicyHandler.Get)
			me.PUT("/devices/:device_id/access-policy", deviceAccessPolicyHandler.Put)
			me.GET("/devices/:device_id/offline-alert", deviceOfflineAlertHandler.Get)
			me.PUT("/devices/:device_id/offline-alert", deviceOfflineAlertHandler.Put)
			me.DELETE("/devices/:device_id/offline-alert", deviceOfflineAlertHandler.Delete)
			me.GET("/devices/:device_id/access-grants", deviceAccessGrantHandler.List)
			me.POST("/devices/:device_id/access-grants", deviceAccessGrantHandler.Create)
			me.DELETE("/devices/:device_id/access-grants/:grant_id", deviceAccessGrantHandler.Delete)
			me.GET("/devices/:device_id/access-approvals", deviceAccessApprovalHandler.List)
			me.GET("/devices/:device_id/access-approvals:summary", deviceAccessApprovalHandler.Summary)
			me.POST("/devices/:device_id/access-approvals/:approval_id/approve", deviceAccessApprovalHandler.Approve)
			me.POST("/devices/:device_id/access-approvals/:approval_id/deny", deviceAccessApprovalHandler.Deny)
			me.PATCH("/devices/:device_id", deviceHandler.Patch)
			me.DELETE("/devices/:device_id", deviceHandler.Unbind)
			me.DELETE("/devices/:device_id/session", deviceHandler.ClearSession)

			me.GET("/connections", deviceHandler.ListConnections)
			me.POST("/connections", deviceHandler.RecordConnection)

			// Who is in the caller's machines right now (status=active) and who
			// has been. Deliberately not /sessions, which is the caller's own
			// login sessions.
			me.GET("/remote-sessions", remoteSessionHandler.List)
			me.GET("/remote-sessions/:session_id", remoteSessionHandler.Get)
			// The owner's "throw them out": closes the live socket, which is
			// what ends the session and frees the device's controller slot.
			me.POST("/remote-sessions/:session_id/revoke", remoteSessionHandler.Revoke)

			me.GET("/notification-channels", notificationChannelHandler.List)
			me.POST("/notification-channels", notificationChannelHandler.Create)
			me.PATCH("/notification-channels/:id", notificationChannelHandler.Update)
			me.DELETE("/notification-channels/:id", notificationChannelHandler.Delete)
			me.POST("/notification-channels/:id/test", notificationChannelHandler.Test)
			me.GET("/notification-channels/:id/deliveries", notificationChannelHandler.Deliveries)
			me.POST("/notification-channels/:id/deliveries/:deliveryId/retry", notificationChannelHandler.RetryDelivery)
			// `ack` closes the loop the webhook POST cannot: a delivery the
			// endpoint accepted is not the same fact as one a person saw.
			me.POST("/notification-channels/:id/deliveries/:deliveryId/ack", notificationChannelHandler.AcknowledgeDelivery)
			me.POST("/notification-channels/:id/deliveries:retry-failed", notificationChannelHandler.RetryFailedDeliveries)
			me.POST("/notification-channels:retry-failed", notificationChannelHandler.RetryFailedDeliveriesForUser)
			// Per-user "do not page me": the delivery worker holds alerts while
			// this is active instead of sending them into the quiet period.
			me.GET("/notification-quiet-hours", notificationQuietHoursHandler.Get)
			me.PUT("/notification-quiet-hours", notificationQuietHoursHandler.Put)

			me.GET("/scheduled-tasks", scheduledTaskHandler.List)
			me.POST("/scheduled-tasks", scheduledTaskHandler.Create)
			me.DELETE("/scheduled-tasks/:id", scheduledTaskHandler.Delete)

			me.GET("/recordings", sessionRecordingHandler.List)
			me.POST("/recordings", sessionRecordingHandler.Create)
			me.GET("/recordings/:id", sessionRecordingHandler.Get)
			me.PATCH("/recordings/:id", sessionRecordingHandler.Update)
			me.DELETE("/recordings/:id", sessionRecordingHandler.Delete)
			me.POST("/recordings/:id/artifact", sessionRecordingHandler.UploadArtifact)
			// PATCH is the resumable half: one chunk at a time, with the
			// authoritative offset coming back in `Upload-Offset`.
			me.PATCH("/recordings/:id/artifact", sessionRecordingHandler.AppendArtifact)
			me.GET("/recordings/:id/artifact", sessionRecordingHandler.DownloadArtifact)
			me.POST("/recordings/:id/artifact-ticket", sessionRecordingHandler.CreateArtifactTicket)
			me.POST("/recordings/:id/thumbnail", sessionRecordingHandler.UploadThumbnail)
			me.GET("/recordings/:id/thumbnail", sessionRecordingHandler.DownloadThumbnail)
			me.POST("/recordings/:id/thumbnail-ticket", sessionRecordingHandler.CreateThumbnailTicket)

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

		// Controller challenge: the same gate as verify, because it also
		// presents the access code. It exists for grants bound to a public key,
		// where the controller has to sign something the server minted before
		// verify will accept it.
		//
		// The path is *not* `access-code:challenge`, even though that would
		// read better: gin's router treats the `:` in `access-code:verify` as
		// the start of a wildcard segment (the handler is reachable as
		// `/access-code<param>`), so a sibling sharing that static prefix
		// panics at route-registration time. It is the same trap the
		// `webhooks/:id/test` note in docs/user-api-docs.md describes.
		v1.POST("/devices/:device_id/controller-challenges", hostHandler.IssueControllerChallenge)

		// claim: the requesting controller exchanges a granted approval
		// (approval_id + one-time secret from access-code:verify) for a
		// signal token. No device secret or user token: the secret is the
		// credential, which is why it is compared in constant time.
		v1.POST("/devices/:device_id/access-approvals:claim", deviceAccessApprovalHandler.Claim)

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
		adminGuarded.POST("/audit-logs:sweep", adminAuditHandler.Sweep)
		// A separate segment, not `audit-logs:archive`: gin keys routes by
		// method and cannot hold two different wildcard names (`:export` and
		// `:archive`) under the same GET prefix.
		adminGuarded.GET("/audit-archive:search", adminAuditHandler.Archive)

		// Session recordings (cross-user index).
		adminGuarded.GET("/recordings", adminSessionRecordingHandler.List)
		adminGuarded.GET("/recordings/:id", adminSessionRecordingHandler.Get)
		adminGuarded.DELETE("/recordings/:id", adminSessionRecordingHandler.Delete)
		adminGuarded.GET("/recordings/:id/artifact", adminSessionRecordingHandler.DownloadArtifactAdmin)
		adminGuarded.GET("/recordings/:id/thumbnail", adminSessionRecordingHandler.DownloadThumbnailAdmin)
		// Action-style paths keep gin's router from having to disambiguate a
		// literal segment against the ":id" wildcard above.
		adminGuarded.GET("/recordings:storage", adminSessionRecordingHandler.Storage)
		adminGuarded.POST("/recordings:sweep", adminSessionRecordingHandler.Sweep)

		// Cross-user control-session console: the admin counterpart of
		// /v1/me/remote-sessions, with the same joined shape as the device list.
		adminGuarded.GET("/remote-sessions", adminRemoteSessionHandler.List)
		adminGuarded.GET("/remote-sessions/:session_id", adminRemoteSessionHandler.Get)
		adminGuarded.POST("/remote-sessions/:session_id/revoke", adminRemoteSessionHandler.Revoke)

		// User-notification outbox (delivery states + on-demand retention).
		adminGuarded.GET("/notification-deliveries:stats", adminNotificationDeliveryHandler.Stats)
		adminGuarded.POST("/notification-deliveries:sweep", adminNotificationDeliveryHandler.Sweep)

		// Raw operational log downloads are intentionally restricted to
		// super administrators and a fixed server-side file allow-list.
		adminGuarded.GET("/system/logs", adminLogsHandler.List)
		adminGuarded.GET("/system/logs:search", adminLogsHandler.Search)
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

		// Self-hosted update channel index.
		adminGuarded.GET("/update-releases", adminUpdateReleaseHandler.List)
		adminGuarded.POST("/update-releases", adminUpdateReleaseHandler.Publish)
		adminGuarded.POST("/update-releases:sweep", adminUpdateReleaseHandler.Sweep)
		adminGuarded.POST("/update-releases/:id/artifact", adminUpdateReleaseHandler.UploadArtifact)
		adminGuarded.PATCH("/update-releases/:id/artifact", adminUpdateReleaseHandler.AppendArtifact)
		adminGuarded.DELETE("/update-releases/:id", adminUpdateReleaseHandler.Delete)

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
	// Recording playback/download — also outside the v1 group. A top-level
	// browser navigation (<video src>, window.open, <a download>) carries
	// neither an X-API-Key nor an Origin header, so APIKeyAuth would reject it
	// before any handler ran. The signed, short-lived ticket in ?t= is the
	// credential instead; the Bearer-authenticated route under /v1/me stays
	// available for API clients.
	// -------------------------------------------------------------------
	router.GET("/v1/recordings/:id/artifact", sessionRecordingHandler.DownloadArtifactByTicket)
	router.GET("/v1/recordings/:id/thumbnail", sessionRecordingHandler.DownloadThumbnailByTicket)

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
