package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"quickdesk/signaling/internal/models"

	"gorm.io/gorm"
)

// RemoteSessionService owns the control plane's record of who is holding a
// signaling session against which device, and when it ended.
//
// It is deliberately the same fact the device access log already records at the
// moment a session starts - but as a row with a lifecycle instead of an
// immutable timeline entry, so "is anyone in this machine right now" and "how
// long was that session" become answerable without reconstructing them from a
// log that never recorded the end.
type RemoteSessionService struct{ db *gorm.DB }

func NewRemoteSessionService(db *gorm.DB) *RemoteSessionService {
	return &RemoteSessionService{db: db}
}

// OpenRemoteSessionInput describes a session that just authenticated.
type OpenRemoteSessionInput struct {
	SessionID string
	DeviceID  string
	ClientID  string
	ClientIP  string
}

// Open records a newly established control session.
//
// It is idempotent on the session id: bookkeeping must never be the reason a
// signaling handshake fails, so a duplicate (a retry, or a race between two
// paths that both noticed the same session) returns the row that is already
// there rather than an error.
func (s *RemoteSessionService) Open(ctx context.Context, in OpenRemoteSessionInput) (*models.RemoteSession, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("remote session service is not configured")
	}
	sessionID := strings.TrimSpace(in.SessionID)
	deviceID := strings.TrimSpace(in.DeviceID)
	if sessionID == "" || deviceID == "" {
		return nil, errors.New("session_id and device_id are required")
	}
	row := models.RemoteSession{
		SessionID: sessionID,
		UserID:    s.ownerOf(ctx, deviceID),
		DeviceID:  deviceID,
		ClientID:  strings.TrimSpace(in.ClientID),
		ClientIP:  strings.TrimSpace(in.ClientIP),
		StartedAt: time.Now().UTC(),
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		if isUniqueViolation(err) {
			var existing models.RemoteSession
			if findErr := s.db.WithContext(ctx).Where("session_id = ?", sessionID).First(&existing).Error; findErr != nil {
				return nil, findErr
			}
			existing.Decorate()
			return &existing, nil
		}
		return nil, err
	}
	row.Decorate()
	return &row, nil
}

// ownerOf resolves the device's owner. An unbound device (or a device row that
// has not been created yet) yields 0, and the session is recorded with no owner
// rather than dropped: the signaling side legitimately knows about devices the
// control plane has not bound to an account.
func (s *RemoteSessionService) ownerOf(ctx context.Context, deviceID string) uint {
	var device models.Device
	if err := s.db.WithContext(ctx).
		Select("user_id").
		Where("device_id = ?", deviceID).
		First(&device).Error; err != nil {
		return 0
	}
	if device.UserID == nil {
		return 0
	}
	return *device.UserID
}

// Close marks a session finished and reports whether this call is the one that
// ended it. Closing twice is not an error - a reconnect, a sweep and a socket
// teardown can all race - but only the first call reports true, so a caller can
// publish exactly one "it ended" event.
func (s *RemoteSessionService) Close(ctx context.Context, sessionID string, reason string) (*models.RemoteSession, bool, error) {
	if s == nil || s.db == nil {
		return nil, false, errors.New("remote session service is not configured")
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil, false, errors.New("session_id is required")
	}
	now := time.Now().UTC()
	result := s.db.WithContext(ctx).
		Model(&models.RemoteSession{}).
		Where("session_id = ? AND ended_at IS NULL", sessionID).
		Updates(map[string]interface{}{
			"ended_at":   now,
			"end_reason": strings.TrimSpace(reason),
			"updated_at": now,
		})
	if result.Error != nil {
		return nil, false, result.Error
	}
	var row models.RemoteSession
	if err := s.db.WithContext(ctx).Where("session_id = ?", sessionID).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	row.Decorate()
	return &row, result.RowsAffected > 0, nil
}

// CloseOpenSessions finishes every session still marked open and reports how
// many were affected. The caller runs it at startup: a signaling socket cannot
// outlive the process that held it, so anything still open at boot belongs to a
// session that ended with the previous process, without the handler's deferred
// close ever running.
func (s *RemoteSessionService) CloseOpenSessions(ctx context.Context, reason string) (int64, error) {
	if s == nil || s.db == nil {
		return 0, errors.New("remote session service is not configured")
	}
	now := time.Now().UTC()
	result := s.db.WithContext(ctx).
		Model(&models.RemoteSession{}).
		Where("ended_at IS NULL").
		Updates(map[string]interface{}{
			"ended_at":   now,
			"end_reason": strings.TrimSpace(reason),
			"updated_at": now,
		})
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

// RemoteSessionFilter narrows a listing. Zero values mean "no filter", except
// Active, where nil is the zero value and means "either".
type RemoteSessionFilter struct {
	UserID   uint
	DeviceID string
	// Active nil lists both states, true only open sessions, false only finished.
	Active   *bool
	Limit    int
	OffsetID uint
}

// List returns newest first. Rows are keyed by descending id, which is the
// order they were opened in.
func (s *RemoteSessionService) List(ctx context.Context, filter RemoteSessionFilter) ([]models.RemoteSession, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("remote session service is not configured")
	}
	query := s.db.WithContext(ctx).Model(&models.RemoteSession{})
	if filter.UserID != 0 {
		query = query.Where("user_id = ?", filter.UserID)
	}
	if deviceID := strings.TrimSpace(filter.DeviceID); deviceID != "" {
		query = query.Where("device_id = ?", deviceID)
	}
	if filter.Active != nil {
		if *filter.Active {
			query = query.Where("ended_at IS NULL")
		} else {
			query = query.Where("ended_at IS NOT NULL")
		}
	}
	if filter.OffsetID != 0 {
		query = query.Where("id < ?", filter.OffsetID)
	}
	// The bound is only here so a caller cannot ask for an unbounded scan; it is
	// deliberately above the API's own page cap (maxCursorLimit) because the
	// handler asks for one row past the page to learn whether a next page exists.
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > maxRemoteSessionListLimit {
		limit = maxRemoteSessionListLimit
	}
	var out []models.RemoteSession
	if err := query.Order("id DESC").Limit(limit).Find(&out).Error; err != nil {
		return nil, err
	}
	if out == nil {
		out = []models.RemoteSession{}
	}
	for i := range out {
		out[i].Decorate()
	}
	return out, nil
}

// maxRemoteSessionListLimit is the service's own ceiling, kept above the API's
// page cap so the handler's "one extra row" probe is never truncated.
const maxRemoteSessionListLimit = 500

// Get loads one session by its signaling id.
func (s *RemoteSessionService) Get(ctx context.Context, sessionID string) (*models.RemoteSession, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("remote session service is not configured")
	}
	var row models.RemoteSession
	if err := s.db.WithContext(ctx).Where("session_id = ?", strings.TrimSpace(sessionID)).First(&row).Error; err != nil {
		return nil, err
	}
	row.Decorate()
	return &row, nil
}

// GetForUser loads one session the caller owns.
//
// A session belonging to somebody else reports "not found" rather than
// "forbidden": the caller learns nothing about whether that session id exists,
// which is the same choice the device-scoped routes make.
func (s *RemoteSessionService) GetForUser(ctx context.Context, userID uint, sessionID string) (*models.RemoteSession, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("remote session service is not configured")
	}
	if userID == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	var row models.RemoteSession
	if err := s.db.WithContext(ctx).
		Where("session_id = ? AND user_id = ?", strings.TrimSpace(sessionID), userID).
		First(&row).Error; err != nil {
		return nil, err
	}
	row.Decorate()
	return &row, nil
}
