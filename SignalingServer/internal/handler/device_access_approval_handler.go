package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"quickdesk/signaling/internal/middleware"
	"quickdesk/signaling/internal/models"
	"quickdesk/signaling/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// DeviceAccessApprovalHandler serves the second-factor approval surface:
//
//	GET  /v1/me/devices/:device_id/access-approvals
//	POST /v1/me/devices/:device_id/access-approvals/:approval_id:approve
//	POST /v1/me/devices/:device_id/access-approvals/:approval_id:deny
//	POST /v1/devices/:device_id/access-approvals:claim
//
// The owner endpoints need a user token; the claim endpoint is called by the
// requesting controller with the one-time secret it received from
// `access-code:verify`, so it only needs the global API-key gate.
type DeviceAccessApprovalHandler struct {
	devices   *service.DeviceService
	approvals *service.DeviceAccessApprovalService
	grants    *service.DeviceAccessGrantService
	tokens    *service.TokenService
	presence  *service.PresenceService
	access    *service.DeviceAccessService
	bus       *service.EventBus
}

func NewDeviceAccessApprovalHandler(
	devices *service.DeviceService,
	approvals *service.DeviceAccessApprovalService,
	grants *service.DeviceAccessGrantService,
	tokens *service.TokenService,
	presence *service.PresenceService,
	access *service.DeviceAccessService,
	bus *service.EventBus,
) *DeviceAccessApprovalHandler {
	return &DeviceAccessApprovalHandler{
		devices:   devices,
		approvals: approvals,
		grants:    grants,
		tokens:    tokens,
		presence:  presence,
		access:    access,
		bus:       bus,
	}
}

// approvalStatuses is the set the list filter accepts.
var approvalStatuses = map[string]struct{}{
	models.ApprovalPending:  {},
	models.ApprovalApproved: {},
	models.ApprovalDenied:   {},
	models.ApprovalExpired:  {},
	models.ApprovalConsumed: {},
}

// approvalView is the wire shape. It never carries the secret hash.
type approvalView struct {
	ID        uint       `json:"id"`
	DeviceID  string     `json:"device_id"`
	ClientID  string     `json:"client_id"`
	ClientIP  string     `json:"client_ip"`
	Source    string     `json:"source"`
	GrantID   *uint      `json:"grant_id,omitempty"`
	Status    string     `json:"status"`
	ExpiresAt time.Time  `json:"expires_at"`
	DecidedAt *time.Time `json:"decided_at,omitempty"`
	Reason    string     `json:"reason,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

func approvalViewOf(row models.DeviceAccessApproval, now time.Time) approvalView {
	return approvalView{
		ID:        row.ID,
		DeviceID:  row.DeviceID,
		ClientID:  row.ClientID,
		ClientIP:  row.ClientIP,
		Source:    row.Source,
		GrantID:   row.GrantID,
		Status:    row.EffectiveStatusAt(now),
		ExpiresAt: row.ExpiresAt,
		DecidedAt: row.DecidedAt,
		Reason:    row.Reason,
		CreatedAt: row.CreatedAt,
	}
}

func (h *DeviceAccessApprovalHandler) List(c *gin.Context) {
	deviceID, ok := h.ownedDeviceID(c)
	if !ok {
		return
	}
	statuses, ok := parseApprovalStatusFilter(c.Query("status"))
	if !ok {
		ProblemBadRequest(c, ProblemCodeInvalidRequest,
			"status must be one or more of pending, approved, denied, expired, consumed")
		return
	}
	limit := 0
	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			ProblemBadRequest(c, ProblemCodeInvalidRequest, "limit must be a positive integer")
			return
		}
		limit = parsed
	}
	rows, err := h.approvals.List(c.Request.Context(), deviceID, statuses, limit)
	if err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	now := time.Now().UTC()
	views := make([]approvalView, 0, len(rows))
	for _, row := range rows {
		views = append(views, approvalViewOf(row, now))
	}
	c.JSON(http.StatusOK, gin.H{"items": views})
}

// Summary answers `GET /v1/me/devices/:device_id/access-approvals:summary`.
//
//	?since=   RFC3339 start of the window (default: the last 24 hours)
//	&limit=   how many refused sources to list (default 5, clamped to 10)
//
// The list endpoint shows individual requests; this is the view that makes
// "one address has been refused six times this afternoon" legible without the
// owner counting rows by hand.
func (h *DeviceAccessApprovalHandler) Summary(c *gin.Context) {
	deviceID, ok := h.ownedDeviceID(c)
	if !ok {
		return
	}
	since := time.Now().UTC().Add(-service.ApprovalSummaryDefaultWindow)
	if raw := strings.TrimSpace(c.Query("since")); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			ProblemBadRequest(c, ProblemCodeInvalidRequest,
				"since must be RFC3339, e.g. 2026-10-16T00:00:00Z")
			return
		}
		since = parsed.UTC()
	}
	limit := 0
	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			ProblemBadRequest(c, ProblemCodeInvalidRequest, "limit must be a positive integer")
			return
		}
		limit = parsed
	}
	summary, err := h.approvals.Summary(c.Request.Context(), deviceID, since, limit)
	if err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, summary)
}

func (h *DeviceAccessApprovalHandler) Approve(c *gin.Context) { h.decide(c, true) }
func (h *DeviceAccessApprovalHandler) Deny(c *gin.Context)    { h.decide(c, false) }

type decideApprovalReq struct {
	Reason string `json:"reason"`
}

func (h *DeviceAccessApprovalHandler) decide(c *gin.Context, approve bool) {
	deviceID, ok := h.ownedDeviceID(c)
	if !ok {
		return
	}
	approvalID, err := strconv.ParseUint(c.Param("approval_id"), 10, 32)
	if err != nil || approvalID == 0 {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "invalid approval id")
		return
	}
	// The body is optional: approving with no reason is the common case.
	var req decideApprovalReq
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
			return
		}
	}
	uid := middleware.MustUserID(c)
	row, changed, err := h.approvals.Decide(
		c.Request.Context(), deviceID, uint(approvalID), approve, uid, req.Reason)
	if err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	if row == nil {
		ProblemNotFound(c, ProblemCodeNotFound, "Approval not found")
		return
	}
	now := time.Now().UTC()
	if !changed {
		// Someone already decided (or it timed out). Report the current
		// state rather than silently pretending the click worked.
		c.JSON(http.StatusConflict, gin.H{
			"status":  row.EffectiveStatusAt(now),
			"message": "This request was already decided",
		})
		return
	}
	view := approvalViewOf(*row, now)
	if !approve {
		h.record(c, deviceID, row, service.DeviceAccessDeniedApproval, firstNonEmpty(req.Reason, "owner denied the session"))
	}
	if h.bus != nil {
		h.bus.Publish(c.Request.Context(), service.Event{
			Type:     service.EventDeviceAccessApprovalDecided,
			UserID:   row.OwnerID,
			DeviceID: deviceID,
			Data: map[string]interface{}{
				"approval_id": row.ID,
				"status":      view.Status,
				"source":      row.Source,
				"client_id":   row.ClientID,
				"reason":      row.Reason,
				// The deciding owner's observed address, matching the other
				// audited user actions; the audit subscriber lifts it onto the
				// row.
				"ip": c.ClientIP(),
			},
		})
	}
	c.JSON(http.StatusOK, view)
}

// claimReq is what the requesting controller sends back to exchange a granted
// approval for a signal token.
type claimReq struct {
	ApprovalID uint   `json:"approval_id" binding:"required"`
	Secret     string `json:"secret" binding:"required"`
}

func (h *DeviceAccessApprovalHandler) Claim(c *gin.Context) {
	deviceID := c.Param("device_id")
	var req claimReq
	if err := c.ShouldBindJSON(&req); err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	row, outcome, err := h.approvals.Peek(c.Request.Context(), deviceID, req.ApprovalID, req.Secret)
	if err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	switch outcome {
	case service.ClaimNotFound:
		ProblemNotFound(c, ProblemCodeNotFound, "Approval not found")
		return
	case service.ClaimPending:
		c.JSON(http.StatusAccepted, gin.H{
			"status":        models.ApprovalPending,
			"expires_at":    row.ExpiresAt,
			"poll_after_ms": 2000,
		})
		return
	case service.ClaimDenied:
		ProblemForbidden(c, ProblemCodeForbidden, "The device owner denied this session")
		return
	case service.ClaimExpired:
		ProblemForbidden(c, ProblemCodeForbidden, "The approval request expired")
		return
	case service.ClaimConsumed:
		c.JSON(http.StatusConflict, gin.H{
			"status":  models.ApprovalConsumed,
			"message": "This approval was already used",
		})
		return
	case service.ClaimConfirmed:
		// Confirmed: fall through to the exchange below.
	default:
		ProblemInternal(c, "unexpected approval outcome")
		return
	}

	// The host may have gone away while the owner was deciding. Check before
	// consuming the approval so an offline host does not burn it.
	if !h.presence.IsOnline(c.Request.Context(), deviceID) {
		ProblemConflict(c, ProblemCodeHostOffline, "Host is offline")
		return
	}

	// A grant-backed approval only spends the guest code's use once the
	// session can actually start, mirroring the non-approval path.
	if row.GrantID != nil && h.grants != nil {
		consumed, cerr := h.grants.Consume(c.Request.Context(), *row.GrantID, row.ClientID, row.ClientIP)
		if cerr != nil {
			ProblemInternal(c, cerr.Error())
			return
		}
		if !consumed {
			h.record(c, deviceID, row, service.DeviceAccessDeniedInvalidCode,
				"access grant is no longer valid")
			ProblemForbidden(c, ProblemCodeInvalidCode, "Access code is incorrect")
			return
		}
		if h.bus != nil {
			h.bus.Publish(c.Request.Context(), service.Event{
				Type:     service.EventDeviceAccessGrantUsed,
				UserID:   row.OwnerID,
				DeviceID: deviceID,
				Data: map[string]interface{}{
					"grant_id":  *row.GrantID,
					"client_id": row.ClientID,
					"approval":  true,
				},
			})
		}
	}

	// Atomically redeem the approval: the UPDATE re-checks status, so a
	// duplicate claim loses here instead of minting a second token.
	row, outcome, err = h.approvals.Claim(c.Request.Context(), deviceID, req.ApprovalID, req.Secret)
	if err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	if outcome != service.ClaimConfirmed {
		c.JSON(http.StatusConflict, gin.H{
			"status":  models.ApprovalConsumed,
			"message": "This approval was already used",
		})
		return
	}

	tok, exp, err := h.tokens.IssueSignalToken(c.Request.Context(), service.SignalTokenPayload{
		DeviceID: deviceID,
		Role:     service.SignalRoleClient,
		ClientID: row.ClientID,
	})
	if err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	h.record(c, deviceID, row, service.DeviceAccessGranted, "owner approved the session")
	c.JSON(http.StatusOK, verifyCodeResp{
		SignalToken: tok,
		ExpiresAt:   exp.UTC().Format(time.RFC3339),
		Status:      models.ApprovalGranted,
	})
}

// record writes one device access log row for the approval lifecycle. Failures
// are ignored: the audit trail must never break the session flow.
func (h *DeviceAccessApprovalHandler) record(
	c *gin.Context,
	deviceID string,
	row *models.DeviceAccessApproval,
	result, reason string,
) {
	if h.access == nil || row == nil {
		return
	}
	_, _ = h.access.Record(c.Request.Context(), service.RecordDeviceAccessInput{
		DeviceID: deviceID,
		ClientID: row.ClientID,
		IP:       row.ClientIP,
		Result:   result,
		Reason:   reason,
	})
}

func parseApprovalStatusFilter(raw string) ([]string, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return []string{models.ApprovalPending}, true
	}
	out := make([]string, 0, 4)
	for _, part := range strings.Split(trimmed, ",") {
		status := strings.TrimSpace(part)
		if status == "" {
			continue
		}
		if _, ok := approvalStatuses[status]; !ok {
			return nil, false
		}
		out = append(out, status)
	}
	if len(out) == 0 {
		return []string{models.ApprovalPending}, true
	}
	return out, true
}

// ownedDeviceID resolves :device_id and enforces that the caller owns it.
func (h *DeviceAccessApprovalHandler) ownedDeviceID(c *gin.Context) (string, bool) {
	uid := middleware.MustUserID(c)
	deviceID := c.Param("device_id")
	d, err := h.devices.GetByDeviceID(c.Request.Context(), deviceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ProblemNotFound(c, ProblemCodeDeviceNotFound, "Device not found")
			return "", false
		}
		ProblemInternal(c, err.Error())
		return "", false
	}
	if d.UserID == nil || *d.UserID != uid {
		ProblemForbidden(c, ProblemCodeForbidden, "Device is not owned by this user")
		return "", false
	}
	return deviceID, true
}
