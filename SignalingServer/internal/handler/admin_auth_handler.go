package handler

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"quickdesk/signaling/internal/middleware"
	"quickdesk/signaling/internal/models"
	"quickdesk/signaling/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/pquerna/otp/totp"
	"gorm.io/gorm"
)

// AdminAuthHandler serves /v1/admin/auth/* — admin login, two-step TOTP,
// token refresh, current-session logout.
type AdminAuthHandler struct {
	admins *service.AdminUserService
	tokens *service.TokenService
	audit  *service.AuditService
}

func NewAdminAuthHandler(admins *service.AdminUserService, tokens *service.TokenService, audit *service.AuditService) *AdminAuthHandler {
	return &AdminAuthHandler{admins: admins, tokens: tokens, audit: audit}
}

type adminLoginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	TOTPCode string `json:"totp_code"`
}

type adminLoginResp struct {
	Admin            gin.H  `json:"admin,omitempty"`
	AccessToken      string `json:"access_token,omitempty"`
	AccessExpiresAt  string `json:"access_expires_at,omitempty"`
	RefreshToken     string `json:"refresh_token,omitempty"`
	RefreshExpiresAt string `json:"refresh_expires_at,omitempty"`
	// When TOTP is required but not provided, the server returns a small
	// pre_token the client uses to complete the two-step login.
	PreToken        string `json:"pre_token,omitempty"`
	TwoFactorNeeded bool   `json:"two_factor_needed,omitempty"`
}

// CreateSession handles POST /v1/admin/auth/sessions.
func (h *AdminAuthHandler) CreateSession(c *gin.Context) {
	var req adminLoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	u, err := h.admins.ValidateCredentials(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		if strings.Contains(err.Error(), "disabled") {
			ProblemForbidden(c, ProblemCodeAccountDisabled, err.Error())
			return
		}
		ProblemUnauthorized(c, ProblemCodeInvalidCredentials, "Invalid credentials")
		return
	}
	if u.TOTPEnabled {
		if req.TOTPCode == "" {
			// Issue a short-lived pre_token so the :totp endpoint can
			// pick up the same admin without re-entering the password.
			preTok, exp, err := h.tokens.IssueSignalToken(c.Request.Context(), service.SignalTokenPayload{
				DeviceID: "admin_pretoken",
				Role:     service.SignalRole("admin"),
				ClientID: formatUint(u.ID),
			})
			if err != nil {
				ProblemInternal(c, err.Error())
				return
			}
			_ = exp
			WriteProblemJSON(c, http.StatusUnauthorized, "TOTP_REQUIRED", "Two-factor authentication required", "Admin login needs second factor", gin.H{"pre_token": preTok})
			return
		}
		if !totp.Validate(req.TOTPCode, u.TOTPSecret) {
			ProblemUnauthorized(c, ProblemCodeInvalidCredentials, "Invalid TOTP code")
			return
		}
	}
	h.writeAdminSession(c, u)
	h.audit.Log(c.Request.Context(), u.ID, u.Username, "admin.login", "admin_user", formatUint(u.ID), "", c.ClientIP())
}

// CreateSessionFromTOTP handles POST /v1/admin/auth/sessions:totp — second
// step of two-factor login.
func (h *AdminAuthHandler) CreateSessionFromTOTP(c *gin.Context) {
	var req struct {
		PreToken string `json:"pre_token" binding:"required"`
		TOTPCode string `json:"totp_code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	payload, err := h.tokens.ConsumeSignalToken(c.Request.Context(), req.PreToken)
	if err != nil || payload.DeviceID != "admin_pretoken" {
		ProblemUnauthorized(c, ProblemCodeTokenInvalid, "pre_token invalid")
		return
	}
	adminID := parseUint(payload.ClientID)
	u, err := h.admins.GetAdminUserByID(c.Request.Context(), adminID)
	if err != nil {
		ProblemUnauthorized(c, ProblemCodeInvalidCredentials, "admin not found")
		return
	}
	if !u.TOTPEnabled {
		ProblemForbidden(c, ProblemCodeForbidden, "2FA not enabled for this account")
		return
	}
	if !totp.Validate(req.TOTPCode, u.TOTPSecret) {
		ProblemUnauthorized(c, ProblemCodeInvalidCredentials, "Invalid TOTP code")
		return
	}
	h.writeAdminSession(c, u)
	h.audit.Log(c.Request.Context(), u.ID, u.Username, "admin.login.totp", "admin_user", formatUint(u.ID), "", c.ClientIP())
}

func (h *AdminAuthHandler) writeAdminSession(c *gin.Context, u *models.AdminUser) {
	tokens, err := h.tokens.IssueSession(c.Request.Context(), service.ScopeAdmin, u.ID, service.SessionMetadata{
		UserAgent: c.Request.UserAgent(),
		IP:        c.ClientIP(),
	})
	if err != nil {
		ProblemInternal(c, "Failed to mint admin session tokens")
		return
	}
	c.JSON(http.StatusOK, adminLoginResp{
		Admin: gin.H{
			"id":           u.ID,
			"username":     u.Username,
			"email":        u.Email,
			"role":         u.Role,
			"status":       u.Status,
			"totp_enabled": u.TOTPEnabled,
			"last_login":   u.LastLogin,
		},
		AccessToken:      tokens.AccessToken,
		AccessExpiresAt:  tokens.AccessExpiresAt.UTC().Format("2006-01-02T15:04:05Z"),
		RefreshToken:     tokens.RefreshToken,
		RefreshExpiresAt: tokens.RefreshExpiresAt.UTC().Format("2006-01-02T15:04:05Z"),
	})
}

// RefreshToken handles POST /v1/admin/auth/tokens:refresh.
func (h *AdminAuthHandler) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	tokens, err := h.tokens.RotateRefreshToken(c.Request.Context(), service.ScopeAdmin, req.RefreshToken)
	if err != nil {
		var fb *service.FamilyBreakInfo
		if errors.As(err, &fb) {
			// Admin family break: audit it; we don't push events out
			// because admin web uses polling, not the realtime stream.
			if fb.SubjectID != 0 {
				h.audit.Log(c.Request.Context(), fb.SubjectID, "", "admin.refresh.family_break", "admin_user", formatUint(fb.SubjectID), "", c.ClientIP())
			}
			ProblemUnauthorized(c, ProblemCodeRefreshInvalid, "Refresh token reuse detected; session family revoked")
			return
		}
		ProblemUnauthorized(c, ProblemCodeRefreshInvalid, "Refresh token invalid or rotated")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"access_token":       tokens.AccessToken,
		"access_expires_at":  tokens.AccessExpiresAt.UTC().Format("2006-01-02T15:04:05Z"),
		"refresh_token":      tokens.RefreshToken,
		"refresh_expires_at": tokens.RefreshExpiresAt.UTC().Format("2006-01-02T15:04:05Z"),
	})
}

// DeleteCurrentSession handles DELETE /v1/admin/auth/sessions/current.
// Mirrors the user-side semantics: revoke the current access_token AND
// the refresh family it belongs to so the operator is fully signed out.
func (h *AdminAuthHandler) DeleteCurrentSession(c *gin.Context) {
	aid := middleware.MustAdminID(c)
	family := middleware.CurrentFamilyID(c)
	if at := middleware.CurrentAccessToken(c); at != "" {
		_ = h.tokens.RevokeAccessToken(c.Request.Context(), service.ScopeAdmin, at)
	}
	if family != "" {
		_ = h.tokens.RevokeFamilyForSubject(c.Request.Context(), service.ScopeAdmin, aid, family)
	}
	h.audit.Log(c.Request.Context(), aid, "", "admin.logout", "admin_user", formatUint(aid), "", c.ClientIP())
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// -----------------------------------------------------------------------
// 2FA setup / verify / delete for the current admin.
// -----------------------------------------------------------------------

// AdminTOTPHandler implements /v1/admin/admins/me/2fa{:setup,:verify,""}.
type AdminTOTPHandler struct {
	admins *service.AdminUserService
	db     *gorm.DB
}

func NewAdminTOTPHandler(admins *service.AdminUserService, db *gorm.DB) *AdminTOTPHandler {
	return &AdminTOTPHandler{admins: admins, db: db}
}

// Setup generates a new TOTP secret and returns the provisioning URI. The
// secret isn't marked `enabled` until the admin successfully verifies it.
func (h *AdminTOTPHandler) Setup(c *gin.Context) {
	id := middleware.MustAdminID(c)
	u, err := h.admins.GetAdminUserByID(c.Request.Context(), id)
	if err != nil {
		ProblemNotFound(c, ProblemCodeNotFound, "Admin not found")
		return
	}
	key, err := totp.Generate(totp.GenerateOpts{Issuer: "QuickDesk", AccountName: u.Username})
	if err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	if err := h.db.WithContext(c.Request.Context()).
		Model(u).Update("totp_secret", key.Secret()).Error; err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"secret": key.Secret(), "qr_uri": key.URL()})
}

// Verify completes TOTP setup — flips totp_enabled=true once the admin
// can produce a valid code.
func (h *AdminTOTPHandler) Verify(c *gin.Context) {
	id := middleware.MustAdminID(c)
	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	u, err := h.admins.GetAdminUserByID(c.Request.Context(), id)
	if err != nil {
		ProblemNotFound(c, ProblemCodeNotFound, "Admin not found")
		return
	}
	if u.TOTPSecret == "" {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "2FA not setup")
		return
	}
	if !totp.Validate(req.Code, u.TOTPSecret) {
		ProblemBadRequest(c, "TOTP_INVALID", "Invalid TOTP code")
		return
	}
	if err := h.db.WithContext(c.Request.Context()).
		Model(u).Update("totp_enabled", true).Error; err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Delete disables 2FA for the current admin. Guarded by a fresh code so a
// stolen session token alone can't strip the second factor.
func (h *AdminTOTPHandler) Delete(c *gin.Context) {
	id := middleware.MustAdminID(c)
	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	u, err := h.admins.GetAdminUserByID(c.Request.Context(), id)
	if err != nil {
		ProblemNotFound(c, ProblemCodeNotFound, "Admin not found")
		return
	}
	if !u.TOTPEnabled {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "2FA not enabled")
		return
	}
	if !totp.Validate(req.Code, u.TOTPSecret) {
		ProblemBadRequest(c, "TOTP_INVALID", "Invalid TOTP code")
		return
	}
	if err := h.db.WithContext(c.Request.Context()).Model(u).Updates(map[string]interface{}{
		"totp_enabled": false,
		"totp_secret":  "",
	}).Error; err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// parseUint / formatUint are tiny helpers used across admin handlers.
func parseUint(s string) uint {
	var v uint
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0
		}
		v = v*10 + uint(r-'0')
	}
	return v
}

func formatUint(v uint) string {
	if v == 0 {
		return "0"
	}
	buf := [20]byte{}
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[i:])
}

var _ = time.Now // keep time import if unused in this file
