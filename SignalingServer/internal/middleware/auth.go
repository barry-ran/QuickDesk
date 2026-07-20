package middleware

import (
	"context"
	"net/http"
	"strings"

	"quickdesk/signaling/internal/httpx"
	"quickdesk/signaling/internal/observability"
	"quickdesk/signaling/internal/service"

	"github.com/gin-gonic/gin"
)

// Gin context keys set by the auth middlewares. Handlers read these via
// MustUserID / MustAdminID / MustDeviceID rather than touching the raw key
// strings.
const (
	ctxKeyUserID   = "auth.user_id"
	ctxKeyAdminID  = "auth.admin_id"
	ctxKeyAccessTk = "auth.access_token"
	ctxKeyFamilyID = "auth.family_id"
	ctxKeyDeviceID = "auth.device_id"
)

// =====================================================================
// User access_token (Bearer)
// =====================================================================

type UserAuth struct {
	tokens *service.TokenService
}

func NewUserAuth(tokens *service.TokenService) *UserAuth {
	return &UserAuth{tokens: tokens}
}

// Required returns a middleware that aborts with RFC7807 401 if no valid
// user access_token is present. On success it sets ctxKeyUserID and
// ctxKeyAccessTk.
func (a *UserAuth) Required() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := bearerOrQuery(c)
		if token == "" {
			observability.Event("auth", "rejected", map[string]interface{}{"path": c.Request.URL.Path, "reason": "missing_user_token", "request_id": c.GetString("request_id")})
			httpx.Unauthorized(c, httpx.CodeUnauthorized, "Missing access token")
			return
		}
		family, uid, err := a.tokens.LookupAccessToken(c.Request.Context(), service.ScopeUser, token)
		if err != nil {
			observability.Event("auth", "rejected", map[string]interface{}{"path": c.Request.URL.Path, "reason": "invalid_user_token", "request_id": c.GetString("request_id")})
			httpx.Unauthorized(c, httpx.CodeTokenExpired, "Access token invalid or expired")
			return
		}
		c.Set(ctxKeyUserID, uid)
		c.Set(ctxKeyAccessTk, token)
		c.Set(ctxKeyFamilyID, family)
		// Bump LastSeen on the session family so /v1/me/sessions stays
		// fresh. Fire-and-forget; no point in failing the request on a
		// Redis hiccup. Use background context because the request context
		// is cancelled after the handler returns.
		go a.tokens.TouchSession(context.Background(), service.ScopeUser, family)
		c.Next()
	}
}

// MustUserID returns the user ID set by UserAuth.Required(); panics only if
// the middleware was forgotten — handlers should never see 0.
func MustUserID(c *gin.Context) uint {
	if v, ok := c.Get(ctxKeyUserID); ok {
		if uid, ok := v.(uint); ok {
			return uid
		}
	}
	return 0
}

// CurrentAccessToken returns the bearer token used for this request, or
// empty if none. Used by DELETE /v1/me/sessions/current.
func CurrentAccessToken(c *gin.Context) string {
	if v, ok := c.Get(ctxKeyAccessTk); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// CurrentFamilyID returns the session family_id associated with the
// current access token (set by UserAuth.Required()/AdminAuth.Required()).
// Used by ListSessions to flag the "current" entry and by
// DELETE /v1/me/sessions/current to revoke the full family.
func CurrentFamilyID(c *gin.Context) string {
	if v, ok := c.Get(ctxKeyFamilyID); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// =====================================================================
// Admin access_token
// =====================================================================

type AdminAuth struct {
	tokens *service.TokenService
}

func NewAdminAuth(tokens *service.TokenService) *AdminAuth {
	return &AdminAuth{tokens: tokens}
}

// Required is the admin counterpart of UserAuth.Required().
func (a *AdminAuth) Required() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := bearerOrQuery(c)
		if token == "" {
			observability.Event("auth", "rejected", map[string]interface{}{"path": c.Request.URL.Path, "reason": "missing_admin_token", "request_id": c.GetString("request_id")})
			httpx.Unauthorized(c, httpx.CodeUnauthorized, "Missing admin access token")
			return
		}
		family, aid, err := a.tokens.LookupAccessToken(c.Request.Context(), service.ScopeAdmin, token)
		if err != nil {
			observability.Event("auth", "rejected", map[string]interface{}{"path": c.Request.URL.Path, "reason": "invalid_admin_token", "request_id": c.GetString("request_id")})
			httpx.Unauthorized(c, httpx.CodeTokenExpired, "Admin token invalid or expired")
			return
		}
		c.Set(ctxKeyAdminID, aid)
		c.Set(ctxKeyAccessTk, token)
		c.Set(ctxKeyFamilyID, family)
		go a.tokens.TouchSession(context.Background(), service.ScopeAdmin, family)
		c.Next()
	}
}

func MustAdminID(c *gin.Context) uint {
	if v, ok := c.Get(ctxKeyAdminID); ok {
		if id, ok := v.(uint); ok {
			return id
		}
	}
	return 0
}

// =====================================================================
// device_secret Bearer
// =====================================================================

type DeviceAuth struct {
	devices *service.DeviceService
}

func NewDeviceAuth(devices *service.DeviceService) *DeviceAuth {
	return &DeviceAuth{devices: devices}
}

// Required validates `Authorization: Bearer <device_secret>` against the
// device referenced in the URL parameter `device_id`. Sets ctxKeyDeviceID
// on success.
func (a *DeviceAuth) Required() gin.HandlerFunc {
	return func(c *gin.Context) {
		deviceID := c.Param("device_id")
		if deviceID == "" {
			httpx.BadRequest(c, httpx.CodeInvalidRequest, "device_id is required in path")
			return
		}
		secret := bearerOrQuery(c)
		if secret == "" {
			observability.Event("device_auth", "rejected", map[string]interface{}{"device_id": deviceID, "reason": "missing_secret", "request_id": c.GetString("request_id")})
			httpx.Unauthorized(c, httpx.CodeUnauthorized, "Missing device secret")
			return
		}
		ok, err := a.devices.VerifyDeviceSecret(c.Request.Context(), deviceID, secret)
		if err != nil {
			observability.Event("device_auth", "rejected", map[string]interface{}{"device_id": deviceID, "reason": "device_not_found", "request_id": c.GetString("request_id")})
			// Device row not found �?treat as bad credentials (don't leak
			// existence) �?but signal NOT_FOUND so a host that's been
			// rotated can detect it and re-provision (§2.21 / scenario 26).
			httpx.NotFound(c, httpx.CodeDeviceNotFound, "Device not registered")
			return
		}
		if !ok {
			observability.Event("device_auth", "rejected", map[string]interface{}{"device_id": deviceID, "reason": "secret_mismatch", "request_id": c.GetString("request_id")})
			httpx.Unauthorized(c, httpx.CodeDeviceSecretBad, "Device secret mismatch")
			return
		}
		c.Set(ctxKeyDeviceID, deviceID)
		c.Next()
	}
}

func MustDeviceID(c *gin.Context) string {
	if v, ok := c.Get(ctxKeyDeviceID); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// =====================================================================
// shared helpers
// =====================================================================

// bearerOrQuery extracts a token from the Authorization header
// (`Bearer <token>`) or, as a last resort, the `?token=` query string.
// Query support is intentional for things like sync WS clients that can't
// set headers easily; production paths should always use the header.
func bearerOrQuery(c *gin.Context) string {
	auth := c.GetHeader("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		t := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
		if t != "" {
			return t
		}
	}
	return strings.TrimSpace(c.Query("token"))
}

// _ keeps net/http imported for future helpers (status constants used
// elsewhere in this package).
var _ = http.StatusOK
