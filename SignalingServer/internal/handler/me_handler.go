package handler

import (
	"net/http"

	"quickdesk/signaling/internal/middleware"
	"quickdesk/signaling/internal/service"

	"github.com/gin-gonic/gin"
)

// MeHandler implements endpoints under /v1/me/* that act on the
// currently-authenticated user. All routes behind this handler must be
// wrapped with middleware.UserAuth.Required().
type MeHandler struct {
	users  *service.UserService
	tokens *service.TokenService
	sms    *service.SmsService
	bus    *service.EventBus
}

func NewMeHandler(users *service.UserService, tokens *service.TokenService, sms *service.SmsService, bus *service.EventBus) *MeHandler {
	return &MeHandler{users: users, tokens: tokens, sms: sms, bus: bus}
}

// Get handles GET /v1/me.
func (h *MeHandler) Get(c *gin.Context) {
	uid := middleware.MustUserID(c)
	u, err := h.users.GetByID(c.Request.Context(), uid)
	if err != nil {
		writeUserErrorProblem(c, err)
		return
	}
	c.JSON(http.StatusOK, userJSON(u))
}

// -----------------------------------------------------------------------
// PUT /v1/me/password  — change password (revokes this session too)
// -----------------------------------------------------------------------

type meChangePasswordReq struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

func (h *MeHandler) ChangePassword(c *gin.Context) {
	uid := middleware.MustUserID(c)
	var req meChangePasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	if err := h.users.ChangePassword(c.Request.Context(), uid, req.OldPassword, req.NewPassword); err != nil {
		writeUserErrorProblem(c, err)
		return
	}
	// §2.2: "改密码后全部 session revoke". Revoking only the current
	// access_token leaves refresh tokens alive on every other device;
	// the next refresh round would silently hand the caller a new
	// session. Kill every family belonging to this user so other
	// devices must log in again with the new password (R32). The bus
	// event below fans out to all connected events WebSockets too.
	h.tokens.RevokeAllForSubject(c.Request.Context(), service.ScopeUser, uid)
	h.bus.Publish(c.Request.Context(), service.Event{
		Type:   service.EventSessionRevoked,
		UserID: uid,
		Data:   map[string]interface{}{"reason": "password_changed"},
	})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// -----------------------------------------------------------------------
// PUT /v1/me/username
// -----------------------------------------------------------------------

type meChangeUsernameReq struct {
	Username string `json:"username" binding:"required"`
}

func (h *MeHandler) ChangeUsername(c *gin.Context) {
	uid := middleware.MustUserID(c)
	var req meChangeUsernameReq
	if err := c.ShouldBindJSON(&req); err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	if err := h.users.ChangeUsername(c.Request.Context(), uid, req.Username); err != nil {
		writeUserErrorProblem(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// -----------------------------------------------------------------------
// PUT /v1/me/phone — changing the phone number requires SMS verification
// of the *new* phone (bind_phone scene).
// -----------------------------------------------------------------------

type meChangePhoneReq struct {
	Phone   string `json:"phone"`
	SmsCode string `json:"sms_code"`
}

func (h *MeHandler) ChangePhone(c *gin.Context) {
	uid := middleware.MustUserID(c)
	var req meChangePhoneReq
	if err := c.ShouldBindJSON(&req); err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	// Clearing the phone is allowed (no SMS check) as long as SMS
	// verification isn't mandatory for the account's other flows. Real
	// policy can be revisited later.
	if req.Phone != "" && h.sms.IsEnabled() {
		if !service.ValidatePhone(req.Phone) {
			ProblemBadRequest(c, "PHONE_INVALID", "Invalid phone format")
			return
		}
		if req.SmsCode == "" {
			ProblemBadRequest(c, ProblemCodeInvalidRequest, "sms_code is required")
			return
		}
		if err := h.sms.VerifyCode(c.Request.Context(), req.Phone, service.SmsSceneBindPhone, req.SmsCode); err != nil {
			writeSmsProblem(c, err)
			return
		}
	}
	if err := h.users.ChangePhone(c.Request.Context(), uid, req.Phone); err != nil {
		writeUserErrorProblem(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// -----------------------------------------------------------------------
// PUT /v1/me/email
// -----------------------------------------------------------------------

type meChangeEmailReq struct {
	Email string `json:"email"`
}

func (h *MeHandler) ChangeEmail(c *gin.Context) {
	uid := middleware.MustUserID(c)
	var req meChangeEmailReq
	if err := c.ShouldBindJSON(&req); err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	if err := h.users.ChangeEmail(c.Request.Context(), uid, req.Email); err != nil {
		writeUserErrorProblem(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// -----------------------------------------------------------------------
// Sessions: GET list / DELETE current / DELETE by id
//
// A "session" here maps 1:1 to a refresh-token family (see §2.16); we
// don't track per-family metadata like user-agent/ip, so the list surface
// returns only the family_id and a `current` flag (set when the listing
// request's access_token happens to live in that family — but we can't
// tell that cheaply, so `current` is false for all but the one that just
// got issued in the same tab; the client keeps track of that locally).
// -----------------------------------------------------------------------

// ListSessions handles GET /v1/me/sessions. Each item carries the
// captured user-agent / IP / last_seen / created_at so the account-
// management UI can describe each active session (§2.2).
func (h *MeHandler) ListSessions(c *gin.Context) {
	uid := middleware.MustUserID(c)
	views, err := h.tokens.ListSessionsWithMeta(c.Request.Context(), service.ScopeUser, uid)
	if err != nil {
		ProblemInternal(c, "Failed to list sessions")
		return
	}
	currentFamily := middleware.CurrentFamilyID(c)
	items := make([]gin.H, 0, len(views))
	for _, v := range views {
		items = append(items, gin.H{
			"id":         v.FamilyID,
			"user_agent": v.Meta.UserAgent,
			"ip":         v.Meta.IP,
			"last_seen":  v.Meta.LastSeen,
			"created_at": v.Meta.CreatedAt,
			"current":    v.FamilyID == currentFamily,
		})
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

// DeleteCurrentSession handles DELETE /v1/me/sessions/current — aka logout.
// Per §2.11 Qt expects this to revoke the full refresh family (access +
// refresh together), not just the active access_token. We publish
// session.revoked so any open realtime WS for this user drops its local
// access_token immediately (§2.17).
func (h *MeHandler) DeleteCurrentSession(c *gin.Context) {
	uid := middleware.MustUserID(c)
	family := middleware.CurrentFamilyID(c)

	if at := middleware.CurrentAccessToken(c); at != "" {
		_ = h.tokens.RevokeAccessToken(c.Request.Context(), service.ScopeUser, at)
	}
	if family != "" {
		_ = h.tokens.RevokeFamilyForSubject(c.Request.Context(), service.ScopeUser, uid, family)
	}
	h.bus.Publish(c.Request.Context(), service.Event{
		Type:   service.EventSessionRevoked,
		UserID: uid,
		Data: map[string]interface{}{
			"family_id": family,
			"reason":    "self_logout",
		},
	})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// DeleteSessionByID handles DELETE /v1/me/sessions/:session_id where
// session_id is a family UUID returned by ListSessions.
func (h *MeHandler) DeleteSessionByID(c *gin.Context) {
	uid := middleware.MustUserID(c)
	familyID := c.Param("session_id")
	if err := h.tokens.RevokeFamilyForSubject(c.Request.Context(), service.ScopeUser, uid, familyID); err != nil {
		if err == service.ErrTokenNotFound {
			ProblemNotFound(c, "SESSION_NOT_FOUND", "Session not found for this user")
			return
		}
		ProblemInternal(c, err.Error())
		return
	}
	h.bus.Publish(c.Request.Context(), service.Event{
		Type:   service.EventSessionRevoked,
		UserID: uid,
		Data:   map[string]interface{}{"family_id": familyID},
	})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
