package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"runtime/debug"
	"time"

	"quickdesk/signaling/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// PublicHandler serves the unauthenticated v1 surface that's neither auth
// (see AuthHandler) nor device-side (see HostHandler):
//
//   GET  /health
//   GET  /v1/preset
//   GET  /v1/settings/public
//   GET  /v1/features
//   POST /v1/verification-codes
type PublicHandler struct {
	db       *gorm.DB
	rdb      *redis.Client
	preset   *service.PresetService
	settings *service.SettingsService
	sms      *service.SmsService

	startedAt time.Time
	version   string
}

func NewPublicHandler(
	db *gorm.DB,
	rdb *redis.Client,
	preset *service.PresetService,
	settings *service.SettingsService,
	sms *service.SmsService,
	version string,
) *PublicHandler {
	if version == "" {
		if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" {
			version = info.Main.Version
		} else {
			version = "dev"
		}
	}
	return &PublicHandler{
		db:        db,
		rdb:       rdb,
		preset:    preset,
		settings:  settings,
		sms:       sms,
		startedAt: time.Now().UTC(),
		version:   version,
	}
}

// Health implements GET /health. Per §2.20 we return 200 only when every
// component is healthy — any non-"ok" component flips the HTTP status so
// a k8s readiness probe fails cleanly.
type componentStatus = map[string]string

func (h *PublicHandler) Health(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	components := componentStatus{}
	overall := http.StatusOK

	// Postgres: simple SELECT 1.
	sqlDB, err := h.db.DB()
	switch {
	case err != nil:
		components["postgres"] = "down"
		overall = http.StatusServiceUnavailable
	default:
		if err := sqlDB.PingContext(ctx); err != nil {
			components["postgres"] = "down"
			overall = http.StatusServiceUnavailable
		} else {
			components["postgres"] = "ok"
		}
	}

	// Redis: PING.
	if err := h.rdb.Ping(ctx).Err(); err != nil {
		components["redis"] = "down"
		overall = http.StatusServiceUnavailable
	} else {
		components["redis"] = "ok"
	}

	payload := gin.H{
		"status":     statusFromCode(overall),
		"version":    h.version,
		"started_at": h.startedAt.Format(time.RFC3339),
		"components": components,
	}
	c.JSON(overall, payload)
}

func statusFromCode(code int) string {
	if code == http.StatusOK {
		return "ok"
	}
	return "degraded"
}

// Preset implements GET /v1/preset (§2.2 — same wire shape the Qt
// PresetManager expects today).
func (h *PublicHandler) Preset(c *gin.Context) {
	p, err := h.preset.GetPreset()
	if err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"announcement":  parseI18NJSON(p.Notice),
		"links":         parseI18NJSON(p.Links),
		"webclient_url": p.WebclientURL,
		"min_version":   p.MinVersion,
	})
}

// parseI18NJSON decodes a JSON blob into a gin.H, returning an empty
// object on failure. Keeps preset endpoint robust against bad rows.
func parseI18NJSON(raw string) interface{} {
	if raw == "" {
		return gin.H{}
	}
	var out interface{}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return gin.H{}
	}
	return out
}

// PublicSettings implements GET /v1/settings/public — the whitelist of
// site/branding fields safe for anonymous callers.
//
// Response uses snake_case per the v1 convention.
func (h *PublicHandler) PublicSettings(c *gin.Context) {
	s := h.settings.Get()
	c.JSON(http.StatusOK, gin.H{
		"site_enabled": s.SiteEnabled,
		"site_name":    s.SiteName,
		"login_logo":   s.LoginLogo,
		"small_logo":   s.SmallLogo,
		"favicon":      s.Favicon,
	})
}

// Features implements GET /v1/features. Per §2.2 the payload is a flat
// set of booleans the Qt / Web frontends use to decide which UI to show.
func (h *PublicHandler) Features(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"sms_enabled":      h.sms.IsEnabled(),
		"register_enabled": true,
	})
}

// -----------------------------------------------------------------------
// POST /v1/verification-codes
// -----------------------------------------------------------------------

type verificationCodeReq struct {
	Phone string `json:"phone" binding:"required"`
	Scene string `json:"scene" binding:"required"`
}

// SendVerificationCode implements POST /v1/verification-codes (§2.2): the
// single entry point for all SMS scenes.
func (h *PublicHandler) SendVerificationCode(c *gin.Context) {
	var req verificationCodeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	if !service.ValidatePhone(req.Phone) {
		ProblemBadRequest(c, "PHONE_INVALID", "Invalid phone format")
		return
	}
	if !service.ValidScene(req.Scene) {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "unknown scene")
		return
	}
	if !h.sms.IsEnabled() {
		ProblemConflict(c, ProblemCodeSmsDisabled, "SMS service is not configured")
		return
	}
	if err := h.sms.SendCode(c.Request.Context(), req.Phone, service.SmsScene(req.Scene)); err != nil {
		// Map sentinel errors onto RFC7807 problems.
		switch {
		case errors.Is(err, service.ErrSmsDisabled):
			ProblemConflict(c, ProblemCodeSmsDisabled, err.Error())
		case errors.Is(err, service.ErrSmsRateLimit):
			ProblemTooManyRequests(c, ProblemCodeSmsRateLimit, "SMS send rate exceeded", 60)
		case errors.Is(err, service.ErrSmsDaily):
			ProblemTooManyRequests(c, ProblemCodeSmsRateLimit, "Daily SMS quota exceeded", 0)
		default:
			ProblemInternal(c, err.Error())
		}
		return
	}
	// `request_id` mirrors the X-Request-ID header so clients can cite a
	// specific verification attempt when filing support tickets (§2.2).
	c.JSON(http.StatusOK, gin.H{
		"request_id": c.GetString("request_id"),
		"expires_at": time.Now().UTC().Add(5 * time.Minute).Format(time.RFC3339),
	})
}
