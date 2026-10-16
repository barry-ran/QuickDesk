package models

import "time"

// RemoteSession is the control plane's own record of one controller holding a
// signaling session against a device.
//
// Before this existed, the only trace of a session was a `session_started` row
// in the device access log: enough to say "someone got in", with no way to ask
// whether they are still in, how long it lasted, or which recording belongs to
// it. The row is opened when a controller's signaling socket finishes
// authentication and closed when that socket ends, which is the whole lifetime
// this server can observe: the media path is peer to peer, so a session whose
// signaling channel is gone is over as far as the control plane is concerned,
// whatever the peers do with their RTP afterwards.
type RemoteSession struct {
	ID uint `gorm:"primaryKey" json:"id"`
	// SessionID is the id the signaling server assigned to that socket, unique
	// so a duplicated open cannot produce two rows for one session.
	SessionID string `gorm:"size:64;not null;uniqueIndex" json:"session_id"`
	// UserID is the device's owner. 0 means the device was unbound when the
	// session started; the owner-scoped list never shows those rows.
	UserID   uint   `gorm:"not null;index" json:"user_id"`
	DeviceID string `gorm:"size:9;not null;index" json:"device_id"`
	// ClientID is the controller-declared client identifier, the same value the
	// access log records, which is what lets the two be read together.
	ClientID string `gorm:"size:64" json:"client_id"`
	ClientIP string `gorm:"size:50" json:"client_ip"`
	// StartedAt is when the signaling handshake completed.
	StartedAt time.Time `gorm:"not null;index" json:"started_at"`
	// EndedAt is null while the signaling channel is up.
	EndedAt   *time.Time `gorm:"index" json:"ended_at"`
	EndReason string     `gorm:"size:32" json:"end_reason"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	// Active is derived from EndedAt; it is not a column, because two sources of
	// truth for "is this still running" is one too many.
	Active bool `gorm:"-" json:"active"`
	// DurationSeconds is derived too: the run length so far for an open session,
	// the total for a finished one. Clients would otherwise each have to decide
	// what "now" means for a session that is still counting.
	DurationSeconds int64 `gorm:"-" json:"duration_seconds"`
}

func (RemoteSession) TableName() string { return "remote_sessions" }

// How a remote session ended.
const (
	// RemoteSessionEndDisconnected is the ordinary case: the controller's
	// signaling socket went away, whether the user closed it, the client crashed
	// or the network dropped.
	RemoteSessionEndDisconnected = "disconnected"
	// RemoteSessionEndServerRestart is stamped at startup on rows that were still
	// open. A signaling socket cannot outlive the process that held it, so those
	// sessions really did end - they just ended without the handler running.
	RemoteSessionEndServerRestart = "server_restart"
)

// Decorate fills the derived fields. Every path that returns a row to a caller
// must go through it, or `active` silently reports false.
func (r *RemoteSession) Decorate() {
	if r == nil {
		return
	}
	r.Active = r.EndedAt == nil
	r.DurationSeconds = int64(r.Duration(time.Now().UTC()).Seconds())
	if r.DurationSeconds < 0 {
		r.DurationSeconds = 0
	}
}

// Duration reports how long the session ran, using `now` for one still open.
func (r *RemoteSession) Duration(now time.Time) time.Duration {
	if r == nil {
		return 0
	}
	end := now
	if r.EndedAt != nil {
		end = *r.EndedAt
	}
	return end.Sub(r.StartedAt)
}
