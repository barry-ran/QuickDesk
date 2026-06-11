package models

import "time"

// Device represents a QuickDesk host device.
//
// Schema notes (see docs/dev/信令服务器API重构方案.md §4):
//   - `online` is NOT a column — it's derived at read time from Redis
//     presence keys (`qd:presence:device:{id}:hb` +
//     `qd:presence:device:{id}:ws:*`).
//   - API-level `logged_in` returned to clients is DERIVED:
//         logged_in = devices.logged_in AND (Redis online)
//     The column itself stores the *user intent* (bound and not explicitly
//     logged out); it flips only on explicit user actions (bind / unbind /
//     session logout) and never on WS connect/disconnect.
//   - `device_secret` itself is never persisted in cleartext. Only its
//     argon2id hash lives here; the plaintext is shown once to the host
//     during POST /v1/devices:provision or a rotate.
type Device struct {
	ID                 uint   `gorm:"primaryKey" json:"id"`
	DeviceID           string `gorm:"uniqueIndex;size:9;not null" json:"device_id"`    // 9-digit public ID
	DeviceUUID         string `gorm:"uniqueIndex;size:64;not null" json:"device_uuid"` // hardware-bound UUID
	DeviceSecretHash   string `gorm:"size:128;not null;default:''" json:"-"`           // argon2id(device_secret)
	MachineFingerprint string `gorm:"size:128;default:''" json:"-"`                    // last-seen machine fingerprint (anti-clone)
	OS                 string `gorm:"size:32" json:"os"`
	OSVersion          string `gorm:"size:32" json:"os_version"`
	AppVersion         string `gorm:"size:32" json:"app_version"`

	// Ownership. Nil when the device is unbound (or user was deleted).
	UserID     *uint  `gorm:"index" json:"user_id"`
	DeviceName string `gorm:"size:128" json:"device_name"`
	AccessCode string `gorm:"size:32" json:"access_code"` // plaintext (confirmed decision §0)

	// User intent — set true on POST /v1/me/devices, cleared on
	// DELETE /v1/me/devices/:id/session or DELETE /v1/me/devices/:id.
	LoggedIn bool `gorm:"column:logged_in;not null;default:false" json:"logged_in"`

	LastSeenAt time.Time `json:"last_seen_at"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`

	// Preload target.
	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (Device) TableName() string { return "devices" }
