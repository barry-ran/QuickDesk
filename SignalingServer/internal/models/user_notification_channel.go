package models

import "time"

// UserNotificationChannel is a user-owned webhook endpoint: the per-user
// counterpart of the admin `webhooks` table. Admin webhooks only carry
// system-scope events (see §2.17 M4); anything tied to one user's devices
// has to be delivered to *that* user's own channel, which is what this table
// is for.
//
// `secret` is intentionally `json:"-"`: it is used to sign outbound
// deliveries and must never come back through the API.
type UserNotificationChannel struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	UserID        uint       `gorm:"not null;index" json:"user_id"`
	Name          string     `gorm:"size:100" json:"name"`
	URL           string     `gorm:"size:500;not null" json:"url"`
	Secret        string     `gorm:"size:200" json:"-"`
	Events        string     `gorm:"type:text;not null" json:"events"` // JSON array of event types
	Enabled       bool       `gorm:"default:true" json:"enabled"`
	LastTriggered *time.Time `json:"last_triggered"`
	LastStatus    int        `json:"last_status"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// TableName overrides the default table name.
func (UserNotificationChannel) TableName() string {
	return "user_notification_channels"
}
