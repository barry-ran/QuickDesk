package models

import "time"

// DeviceAccessLog is the device owner's view of incoming remote sessions:
// who tried to open a session on this machine, from where, and whether it
// worked.
//
// Rows are written by the server at access-code verification time, never by
// the controller, so a client cannot hide or forge its own attempts. The
// caller's identity is only ever an IP + client_id today: the desktop client
// authenticates against `access-code:verify` with the server API key /
// Origin whitelist rather than a user session, so the server has no user to
// attribute the attempt to.
type DeviceAccessLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	DeviceID  string    `gorm:"size:9;not null;index" json:"device_id"`
	OwnerID   uint      `gorm:"index" json:"owner_id"` // device owner at the time of the attempt, 0 = unbound
	ClientID  string    `gorm:"size:64" json:"client_id"`
	SessionID string    `gorm:"size:64" json:"session_id"` // signaling session id, set for session_started rows
	IP        string    `gorm:"size:50" json:"ip"`
	Result    string    `gorm:"size:32;not null;index" json:"result"`
	Reason    string    `gorm:"size:255" json:"reason"`
	CreatedAt time.Time `json:"created_at"`
}

// TableName overrides the default table name.
func (DeviceAccessLog) TableName() string {
	return "device_access_logs"
}
