package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"quickdesk/signaling/internal/models"

	"gorm.io/gorm"
)

// Device access results. These strings are persisted and shown to the device
// owner, so keep them stable.
const (
	// DeviceAccessGranted means the access code matched and the host was
	// online, so the server minted a signal token for the controller.
	DeviceAccessGranted = "granted"
	// DeviceAccessDeniedInvalidCode means the submitted code was wrong.
	DeviceAccessDeniedInvalidCode = "denied_invalid_code"
	// DeviceAccessDeniedRateLimited covers both the preflight block (too many
	// attempts from this IP / device / pair) and tripping the per-device
	// failure counter.
	DeviceAccessDeniedRateLimited = "denied_rate_limited"
	// DeviceAccessHostOffline means the code matched but the host was not
	// reachable, so no session could start. Not counted as a failure.
	DeviceAccessHostOffline = "host_offline"
	// DeviceAccessSessionStarted means the controller actually established
	// the signaling channel with a valid token: the attempt turned into a
	// real control session. Recorded by the realtime handler, not by verify.
	DeviceAccessSessionStarted = "session_started"
	// DeviceAccessDeniedPolicy means the device's own access policy refused
	// the client (remote access disabled, or the client IP is outside the
	// allow list). The access code is never even compared.
	DeviceAccessDeniedPolicy = "denied_policy"
	// DeviceAccessDeniedSessionLimit means the access code was accepted but
	// the device already had as many controllers as its policy allows.
	DeviceAccessDeniedSessionLimit = "denied_session_limit"
	// DeviceAccessPendingApproval means the code matched but the device
	// policy requires the owner to approve the session first. The access log
	// shows it as neither granted nor denied until the owner decides.
	DeviceAccessPendingApproval = "pending_approval"
	// DeviceAccessDeniedApproval means the owner (or the approval deadline)
	// refused the session even though the code was correct.
	DeviceAccessDeniedApproval = "denied_approval"
)

// ErrInvalidDeviceAccessResult is returned when a caller passes a result
// string outside the set above.
var ErrInvalidDeviceAccessResult = errors.New("invalid device access result")

// IsDeviceAccessResult reports whether result is one of the canonical values.
func IsDeviceAccessResult(result string) bool {
	switch result {
	case DeviceAccessGranted,
		DeviceAccessDeniedInvalidCode,
		DeviceAccessDeniedRateLimited,
		DeviceAccessHostOffline,
		DeviceAccessSessionStarted,
		DeviceAccessDeniedPolicy,
		DeviceAccessDeniedSessionLimit,
		DeviceAccessPendingApproval,
		DeviceAccessDeniedApproval:
		return true
	default:
		return false
	}
}

// DeviceAccessService persists device access attempts and serves them back to
// the device owner.
type DeviceAccessService struct {
	db *gorm.DB
}

func NewDeviceAccessService(db *gorm.DB) *DeviceAccessService {
	return &DeviceAccessService{db: db}
}

// RecordDeviceAccessInput is one access attempt as observed by the server.
type RecordDeviceAccessInput struct {
	DeviceID  string
	ClientID  string
	SessionID string
	IP        string
	Result    string
	Reason    string
}

// Record persists one access attempt. Callers treat a failure as
// non-fatal: the verification response must not depend on the audit write.
func (s *DeviceAccessService) Record(ctx context.Context, in RecordDeviceAccessInput) (*models.DeviceAccessLog, error) {
	if !IsDeviceAccessResult(in.Result) {
		return nil, ErrInvalidDeviceAccessResult
	}
	if strings.TrimSpace(in.DeviceID) == "" {
		return nil, errors.New("device_id is required")
	}
	if len(in.Reason) > 255 {
		in.Reason = in.Reason[:255]
	}
	row := &models.DeviceAccessLog{
		DeviceID:  in.DeviceID,
		OwnerID:   s.OwnerID(ctx, in.DeviceID),
		ClientID:  in.ClientID,
		SessionID: in.SessionID,
		IP:        in.IP,
		Result:    in.Result,
		Reason:    in.Reason,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.db.WithContext(ctx).Create(row).Error; err != nil {
		return nil, err
	}
	return row, nil
}

// OwnerID returns the user that currently owns deviceID, or 0 when the device
// is unbound. Lookup errors collapse to 0 on purpose: an access attempt
// against an unbound device is still worth recording.
func (s *DeviceAccessService) OwnerID(ctx context.Context, deviceID string) uint {
	var ud models.UserDevice
	err := s.db.WithContext(ctx).
		Where("device_id = ? AND status = ?", deviceID, true).
		Order("id ASC").
		First(&ud).Error
	if err != nil {
		return 0
	}
	return ud.UserID
}

// DeviceAccessListParams carries the cursor used by
// GET /v1/me/devices/:device_id/access-logs (keyset on the descending id).
type DeviceAccessListParams struct {
	BeforeID uint
	Limit    int
}

// List returns the device's access attempts newest-first.
func (s *DeviceAccessService) List(ctx context.Context, deviceID string, p DeviceAccessListParams) ([]models.DeviceAccessLog, error) {
	limit := p.Limit
	if limit <= 0 {
		limit = defaultDeviceAccessLimit
	}
	if limit > maxDeviceAccessLimit {
		limit = maxDeviceAccessLimit
	}
	q := s.db.WithContext(ctx).
		Model(&models.DeviceAccessLog{}).
		Where("device_id = ?", deviceID)
	if p.BeforeID > 0 {
		q = q.Where("id < ?", p.BeforeID)
	}
	var out []models.DeviceAccessLog
	err := q.Order("id DESC").Limit(limit).Find(&out).Error
	return out, err
}

const (
	defaultDeviceAccessLimit = 50
	maxDeviceAccessLimit     = 200
)

// ListForSession returns the access-log rows that carry one signaling session
// id, newest first.
//
// In practice that is the `session_started` row the realtime handler wrote when
// the controller authenticated: it is the one entry in the device's timeline
// that names the session, and pairing it with the session row is what turns "an
// attempt" into "the attempt that became this session".
func (s *DeviceAccessService) ListForSession(
	ctx context.Context,
	deviceID string,
	sessionID string,
	limit int,
) ([]models.DeviceAccessLog, error) {
	if strings.TrimSpace(deviceID) == "" || strings.TrimSpace(sessionID) == "" {
		return []models.DeviceAccessLog{}, nil
	}
	if limit <= 0 {
		limit = defaultDeviceAccessLimit
	}
	if limit > maxDeviceAccessLimit {
		limit = maxDeviceAccessLimit
	}
	var out []models.DeviceAccessLog
	if err := s.db.WithContext(ctx).
		Where("device_id = ? AND session_id = ?", strings.TrimSpace(deviceID), strings.TrimSpace(sessionID)).
		Order("id DESC").
		Limit(limit).
		Find(&out).Error; err != nil {
		return nil, err
	}
	if out == nil {
		out = []models.DeviceAccessLog{}
	}
	return out, nil
}
