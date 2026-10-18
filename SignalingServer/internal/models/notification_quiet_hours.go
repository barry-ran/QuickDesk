package models

import "time"

// NotificationQuietHours is one user's "do not page me" setting for the
// notification outbox.
//
// It sits on the delivery side rather than on any single rule because the
// question it answers is about the person, not the event: a backup window, a
// night shift or a holiday puts every alert on hold at once, whatever produced
// them. The row is one per user and carries no history - the setting is
// current state, and the deliveries it held keep their own record of when they
// were actually sent.
//
// SnoozedUntil is the one-shot form ("quiet until this instant"), Windows the
// recurring form ({days, start, end} evaluated in Timezone). Both suppress
// sending without discarding anything: a held delivery stays pending and is
// attempted once the quiet period ends.
type NotificationQuietHours struct {
	UserID       uint       `gorm:"primaryKey" json:"user_id"`
	SnoozedUntil *time.Time `json:"snoozed_until"`
	// Windows is a JSON array of AccessWindow values. Empty (or "") means no
	// recurring quiet periods, which is why "no windows" has one representation
	// rather than both "" and "[]".
	Windows   string    `gorm:"type:text;not null;default:''" json:"-"`
	Timezone  string    `gorm:"size:64;not null;default:''" json:"timezone"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName overrides the default table name.
func (NotificationQuietHours) TableName() string {
	return "notification_quiet_hours"
}
