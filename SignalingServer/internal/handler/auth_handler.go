package handler

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"quickdesk/signaling/internal/models"
	"quickdesk/signaling/internal/observability"
	"quickdesk/signaling/internal/service"

	"github.com/gin-gonic/gin"
)

// AuthHandler implements public auth endpoints under /v1/auth/*:
//
//	POST /v1/auth/register
//	POST /v1/auth/sessions
//	POST /v1/auth/sessions:sms
//	POST /v1/auth/tokens:refresh
//	POST /v1/auth/password-resets
//	POST /v1/auth/password-resets:confirm
//
// None of these require an existing session.
type AuthHandler struct {
	users  *service.UserService
	tokens *service.TokenService
	sms    *service.SmsService
	bus    *service.EventBus
}

func NewAuthHandler(users *service.UserService, tokens *service.TokenService, sms *service.SmsService, bus *service.EventBus) *AuthHandler {
	return &AuthHandler{users: users, tokens: tokens, sms: sms, bus: bus}
}

// -----------------------------------------------------------------------
// Wire types
// -----------------------------------------------------------------------

type authRegisterReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Phone    string `json:"phone"`
	Email    string `json:"email"`
	SmsCode  string `json:"sms_code"`
}

type authSessionReq struct {
	Identifier string `json:"identifier" binding:"required"`
	Password   string `json:"password"   binding:"required"`
}

type authSessionSmsReq struct {
	Phone   string `json:"phone"    binding:"required"`
	SmsCode string `json:"sms_code" binding:"required"`
}

type authRefreshReq struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type authResetReq struct {
	Phone string `json:"phone" binding:"required"`
}

type authResetConfirmReq struct {
	Phone       string `json:"phone"        binding:"required"`
	SmsCode     string `json:"sms_code"     binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

// authSessionResponse mirrors the docs §2.2 wire contract.
type authSessionResponse struct {
	User             gin.H  `json:"user"`
	AccessToken      string `json:"access_token"`
	AccessExpiresAt  string `json:"access_expires_at"`
	RefreshToken     string `json:"refresh_token"`
	RefreshExpiresAt string `json:"refresh_expires_at"`
}

// -----------------------------------------------------------------------
// Handlers
// -----------------------------------------------------------------------

func (h *AuthHandler) Register(c *gin.Context) {
	var req authRegisterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}

	// When the caller supplies a phone AND SMS is configured, require a
	// verified code. If SMS is disabled server-wide the phone is accepted
	// unverified (self-hosted convenience per §0).
	if req.Phone != "" && h.sms.IsEnabled() {
		if !service.ValidatePhone(req.Phone) {
			ProblemBadRequest(c, "PHONE_INVALID", "Invalid phone format")
			return
		}
		if req.SmsCode == "" {
			ProblemBadRequest(c, ProblemCodeInvalidRequest, "sms_code is required with phone")
			return
		}
		if err := h.sms.VerifyCode(c.Request.Context(), req.Phone, service.SmsSceneRegister, req.SmsCode); err != nil {
			writeSmsProblem(c, err)
			return
		}
	}

	user, err := h.users.Register(c.Request.Context(), service.RegisterInput{
		Username: req.Username,
		Password: req.Password,
		Phone:    req.Phone,
		Email:    req.Email,
	})
	if err != nil {
		writeUserErrorProblem(c, err)
		return
	}
	h.issueSession(c, user, "register")
}

func (h *AuthHandler) CreateSession(c *gin.Context) {
	var req authSessionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	user, err := h.users.LoginByIdentifier(c.Request.Context(), strings.TrimSpace(req.Identifier), req.Password)
	if err != nil {
		writeUserErrorProblem(c, err)
		return
	}
	h.issueSession(c, user, "password_login")
}

func (h *AuthHandler) CreateSessionSms(c *gin.Context) {
	var req authSessionSmsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	if !service.ValidatePhone(req.Phone) {
		ProblemBadRequest(c, "PHONE_INVALID", "Invalid phone format")
		return
	}
	if err := h.sms.VerifyCode(c.Request.Context(), req.Phone, service.SmsSceneLogin, req.SmsCode); err != nil {
		writeSmsProblem(c, err)
		return
	}
	user, err := h.users.LoginByPhone(c.Request.Context(), req.Phone)
	if err != nil {
		writeUserErrorProblem(c, err)
		return
	}
	h.issueSession(c, user, "sms_login")
}

func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req authRefreshReq
	if err := c.ShouldBindJSON(&req); err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	tokens, err := h.tokens.RotateRefreshToken(c.Request.Context(), service.ScopeUser, req.RefreshToken)
	if err != nil {
		var fb *service.FamilyBreakInfo
		if errors.As(err, &fb) {
			if fb.SubjectID != 0 {
				// A refresh-token reuse revokes only the compromised family.
				// Keep the family_id on the event so RealtimeHandler does not
				// mistake this for an account-wide administrative revocation.
				log.Printf("[auth] refresh family revoked user_id=%d family_id=%s reason=family_break request_id=%s",
					fb.SubjectID, fb.FamilyID, c.GetHeader("X-Request-ID"))
				h.bus.Publish(c.Request.Context(), service.Event{
					Type:   service.EventSessionRevoked,
					UserID: fb.SubjectID,
					Data: map[string]interface{}{
						"family_id": fb.FamilyID,
						"reason":    "family_break",
					},
				})
			}
			ProblemUnauthorized(c, ProblemCodeRefreshInvalid, "Refresh token reuse detected; session family revoked")
			return
		}
		ProblemUnauthorized(c, ProblemCodeRefreshInvalid, "Refresh token invalid or rotated")
		return
	}
	observability.Event("auth", "refresh_succeeded", map[string]interface{}{
		"family_id":  tokens.FamilyID,
		"request_id": c.GetString("request_id"),
	})
	c.JSON(http.StatusOK, gin.H{
		"access_token":       tokens.AccessToken,
		"access_expires_at":  tokens.AccessExpiresAt.UTC().Format("2006-01-02T15:04:05Z"),
		"refresh_token":      tokens.RefreshToken,
		"refresh_expires_at": tokens.RefreshExpiresAt.UTC().Format("2006-01-02T15:04:05Z"),
	})
}

func (h *AuthHandler) RequestPasswordReset(c *gin.Context) {
	var req authResetReq
	if err := c.ShouldBindJSON(&req); err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	if !service.ValidatePhone(req.Phone) {
		ProblemBadRequest(c, "PHONE_INVALID", "Invalid phone format")
		return
	}
	if !h.sms.IsEnabled() {
		ProblemConflict(c, ProblemCodeSmsDisabled, "SMS service not configured")
		return
	}
	// Don't leak whether the phone is registered; just silently return OK
	// when no user exists.
	if _, err := h.users.GetByPhone(c.Request.Context(), req.Phone); err != nil {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
		return
	}
	if err := h.sms.SendCode(c.Request.Context(), req.Phone, service.SmsSceneResetPassword); err != nil {
		writeSmsProblem(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *AuthHandler) ConfirmPasswordReset(c *gin.Context) {
	var req authResetConfirmReq
	if err := c.ShouldBindJSON(&req); err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	if !service.ValidatePhone(req.Phone) {
		ProblemBadRequest(c, "PHONE_INVALID", "Invalid phone format")
		return
	}
	if err := h.sms.VerifyCode(c.Request.Context(), req.Phone, service.SmsSceneResetPassword, req.SmsCode); err != nil {
		writeSmsProblem(c, err)
		return
	}
	if err := h.users.ResetPasswordByPhone(c.Request.Context(), req.Phone, req.NewPassword); err != nil {
		writeUserErrorProblem(c, err)
		return
	}
	// §2.2 / R32: password changes must revoke every existing session,
	// otherwise a device that still holds the user's pre-reset refresh
	// token can silently mint a new access_token. Look up the user so
	// we can target the reverse-family index, then publish
	// session.revoked to kick any connected events WebSocket.
	if u, err := h.users.GetByPhone(c.Request.Context(), req.Phone); err == nil && u != nil {
		h.tokens.RevokeAllForSubject(c.Request.Context(), service.ScopeUser, u.ID)
		observability.Event("auth", "sessions_revoked", map[string]interface{}{
			"reason":     "password_reset",
			"request_id": c.GetString("request_id"),
			"user_id":    u.ID,
		})
		h.bus.Publish(c.Request.Context(), service.Event{
			Type:   service.EventSessionRevoked,
			UserID: u.ID,
			Data:   map[string]interface{}{"reason": "password_reset"},
		})
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// -----------------------------------------------------------------------
// helpers
// -----------------------------------------------------------------------

// issueSession mints the access+refresh pair and writes the §2.2 envelope.
func (h *AuthHandler) issueSession(c *gin.Context, user *models.User, source string) {
	tokens, err := h.tokens.IssueSession(c.Request.Context(), service.ScopeUser, user.ID, service.SessionMetadata{
		UserAgent: c.Request.UserAgent(),
		IP:        c.ClientIP(),
	})
	if err != nil {
		ProblemInternal(c, "Failed to mint session tokens")
		return
	}
	observability.Event("auth", "session_created", map[string]interface{}{
		"family_id":  tokens.FamilyID,
		"ip":         c.ClientIP(),
		"request_id": c.GetString("request_id"),
		"source":     source,
		"user_id":    user.ID,
	})
	c.JSON(http.StatusOK, authSessionResponse{
		User:             userJSON(user),
		AccessToken:      tokens.AccessToken,
		AccessExpiresAt:  tokens.AccessExpiresAt.UTC().Format("2006-01-02T15:04:05Z"),
		RefreshToken:     tokens.RefreshToken,
		RefreshExpiresAt: tokens.RefreshExpiresAt.UTC().Format("2006-01-02T15:04:05Z"),
	})
}
