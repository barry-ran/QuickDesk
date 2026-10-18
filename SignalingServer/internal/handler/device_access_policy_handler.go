package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"quickdesk/signaling/internal/middleware"
	"quickdesk/signaling/internal/models"
	"quickdesk/signaling/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// DeviceAccessPolicyHandler serves the owner-controlled access policy:
//
//	GET /v1/me/devices/:device_id/access-policy
//	PUT /v1/me/devices/:device_id/access-policy
//
// The policy is enforced on the device-side verify endpoint, so the owner of
// the *controlled* machine decides who may even try to connect.
type DeviceAccessPolicyHandler struct {
	devices  *service.DeviceService
	policies *service.DeviceAccessPolicyService
	bus      *service.EventBus
}

func NewDeviceAccessPolicyHandler(devices *service.DeviceService, policies *service.DeviceAccessPolicyService, bus *service.EventBus) *DeviceAccessPolicyHandler {
	return &DeviceAccessPolicyHandler{devices: devices, policies: policies, bus: bus}
}

type deviceAccessPolicyReq struct {
	RemoteDisabled     bool                          `json:"remote_disabled"`
	AllowIPs           []string                      `json:"allow_ips"`
	MaxSessions        int                           `json:"max_sessions"`
	AllowedWindows     []service.AccessWindow        `json:"allowed_windows"`
	Timezone           string                        `json:"timezone"`
	ExceptionDates     []service.AccessExceptionDate `json:"exception_dates"`
	ApprovalRequired   bool                          `json:"approval_required"`
	ApprovalMode       string                        `json:"approval_mode"`
	ApprovalTrustedIPs []string                      `json:"approval_trusted_ips"`
}

// deviceAccessPolicyView is the wire shape: `allow_ips` is returned as a real
// array (the column stores JSON) so clients never parse a string-of-JSON.
type deviceAccessPolicyView struct {
	DeviceID           string                        `json:"device_id"`
	RemoteDisabled     bool                          `json:"remote_disabled"`
	AllowIPs           []string                      `json:"allow_ips"`
	MaxSessions        int                           `json:"max_sessions"`
	AllowedWindows     []service.AccessWindow        `json:"allowed_windows"`
	Timezone           string                        `json:"timezone"`
	ExceptionDates     []service.AccessExceptionDate `json:"exception_dates"`
	ApprovalRequired   bool                          `json:"approval_required"`
	ApprovalMode       string                        `json:"approval_mode"`
	ApprovalTrustedIPs []string                      `json:"approval_trusted_ips"`
	UpdatedAt          *time.Time                    `json:"updated_at,omitempty"`
}

func (h *DeviceAccessPolicyHandler) Get(c *gin.Context) {
	deviceID, ok := h.ownedDeviceID(c)
	if !ok {
		return
	}
	policy, err := h.policies.Get(c.Request.Context(), deviceID)
	if err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, policyView(deviceID, policy))
}

func (h *DeviceAccessPolicyHandler) Put(c *gin.Context) {
	deviceID, ok := h.ownedDeviceID(c)
	if !ok {
		return
	}
	var req deviceAccessPolicyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	uid := middleware.MustUserID(c)
	policy, err := h.policies.Upsert(c.Request.Context(), deviceID, uid,
		service.UpsertDeviceAccessPolicyInput{
			RemoteDisabled:     req.RemoteDisabled,
			AllowIPs:           req.AllowIPs,
			MaxSessions:        req.MaxSessions,
			AllowedWindows:     req.AllowedWindows,
			Timezone:           req.Timezone,
			ExceptionDates:     req.ExceptionDates,
			ApprovalRequired:   req.ApprovalRequired,
			ApprovalMode:       req.ApprovalMode,
			ApprovalTrustedIPs: req.ApprovalTrustedIPs,
		})
	if err != nil {
		if errors.Is(err, service.ErrInvalidAllowIP) ||
			errors.Is(err, service.ErrInvalidSessionLimit) ||
			errors.Is(err, service.ErrInvalidAccessWindow) ||
			errors.Is(err, service.ErrInvalidTimezone) ||
			errors.Is(err, service.ErrInvalidAccessException) ||
			errors.Is(err, service.ErrInvalidApprovalMode) {
			ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
			return
		}
		ProblemInternal(c, err.Error())
		return
	}
	if h.bus != nil {
		h.bus.Publish(c.Request.Context(), service.Event{
			Type:     service.EventDevicePolicyChanged,
			UserID:   uid,
			DeviceID: deviceID,
			Data: map[string]interface{}{
				"device_id":            deviceID,
				"remote_disabled":      policy.RemoteDisabled,
				"allow_ips":            decodeAllowIPsForEvent(policy.AllowIPs),
				"max_sessions":         policy.MaxSessions,
				"allowed_windows":      decodeAccessWindowsForView(policy.AllowedWindows),
				"timezone":             policy.Timezone,
				"exception_dates":      decodeAccessExceptionsForView(policy.ExceptionDates),
				"approval_required":    policy.ApprovalRequired,
				"approval_mode":        service.EffectiveApprovalMode(policy),
				"approval_trusted_ips": decodeAllowIPsForEvent(policy.ApprovalTrustedIPs),
				// The actor's observed address, lifted onto the audit row by
				// the audit subscriber.
				"ip": c.ClientIP(),
			},
		})
	}
	c.JSON(http.StatusOK, policyView(deviceID, policy))
}

// ownedDeviceID resolves :device_id and enforces that the caller owns it.
func (h *DeviceAccessPolicyHandler) ownedDeviceID(c *gin.Context) (string, bool) {
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

func policyView(deviceID string, policy *models.DeviceAccessPolicy) deviceAccessPolicyView {
	view := deviceAccessPolicyView{DeviceID: deviceID, AllowIPs: []string{}}
	if policy == nil {
		// The endpoint documents "the documented no-policy defaults" and clients
		// render their form straight from this response, so the empty case has to
		// have the same shape as a stored one. Returning Go's zero values here
		// gave clients `null` for three documented arrays and "" for
		// `approval_mode`, which the conformance smoke caught on a device that
		// never had a policy row.
		view.AllowedWindows = []service.AccessWindow{}
		view.ExceptionDates = []service.AccessExceptionDate{}
		view.ApprovalMode = service.ApprovalModeOff
		view.ApprovalTrustedIPs = []string{}
		return view
	}
	view.RemoteDisabled = policy.RemoteDisabled
	view.MaxSessions = policy.MaxSessions
	view.Timezone = policy.Timezone
	view.AllowedWindows = decodeAccessWindowsForView(policy.AllowedWindows)
	view.ExceptionDates = decodeAccessExceptionsForView(policy.ExceptionDates)
	view.ApprovalRequired = policy.ApprovalRequired
	view.ApprovalMode = service.EffectiveApprovalMode(policy)
	view.ApprovalTrustedIPs = decodeAllowIPsForEvent(policy.ApprovalTrustedIPs)
	if entries := decodeAllowIPsForEvent(policy.AllowIPs); len(entries) > 0 {
		view.AllowIPs = entries
	}
	updated := policy.UpdatedAt
	view.UpdatedAt = &updated
	return view
}

func decodeAllowIPsForEvent(encoded string) []string {
	if encoded == "" {
		return []string{}
	}
	var entries []string
	if err := json.Unmarshal([]byte(encoded), &entries); err != nil {
		return []string{}
	}
	if entries == nil {
		return []string{}
	}
	return entries
}

func decodeAccessWindowsForView(encoded string) []service.AccessWindow {
	if encoded == "" {
		return []service.AccessWindow{}
	}
	var windows []service.AccessWindow
	if err := json.Unmarshal([]byte(encoded), &windows); err != nil {
		return []service.AccessWindow{}
	}
	if windows == nil {
		return []service.AccessWindow{}
	}
	return windows
}

func decodeAccessExceptionsForView(encoded string) []service.AccessExceptionDate {
	if encoded == "" {
		return []service.AccessExceptionDate{}
	}
	var exceptions []service.AccessExceptionDate
	if err := json.Unmarshal([]byte(encoded), &exceptions); err != nil {
		return []service.AccessExceptionDate{}
	}
	if exceptions == nil {
		return []service.AccessExceptionDate{}
	}
	return exceptions
}
