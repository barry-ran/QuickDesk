package models

import "time"

// DeviceOfflineAlertRule is one owner-scoped rule: "tell me when this device
// has been offline for longer than after_minutes".
//
// The row also carries the evaluator's episode state, because "offline since
// when" is not something presence can answer. Redis only says whether the
// heartbeat and WS keys exist *right now*; a rule that asked "was this device
// offline for ten minutes?" without recording the start of the episode would
// either fire immediately on the first offline observation or never fire at
// all.
//
//	offline_since - set the first time the evaluator saw the device offline,
//	                cleared when it comes back. NULL means "not in an episode".
//	alerted_at    - set when the alert for the current episode was published.
//	                Cleared together with offline_since, so the next outage
//	                alerts again instead of being suppressed forever.
//
// Three optional behaviours ride on the same row. `repeat_minutes` > 0 turns
// alerted_at from a one-shot flag into "when we last spoke up", so a long
// outage keeps reminding the owner instead of producing one easily-missed
// message. `snoozed_until` suppresses publishing until a deadline without
// pausing the episode clock. `maintenance_windows` is the recurring form of
// the same idea: a JSON array of {days, start, end} windows in
// `maintenance_timezone` that are always quiet, so a weekly backup window does
// not need a fresh snooze every time.
//
// Both fields are written with conditional UPDATEs so two server instances
// cannot both start an episode or both fire one alert.
type DeviceOfflineAlertRule struct {
	ID           uint   `gorm:"primaryKey" json:"id"`
	DeviceID     string `gorm:"uniqueIndex;size:20;not null" json:"device_id"`
	OwnerID      uint   `gorm:"index;not null" json:"owner_id"`
	Enabled      bool   `gorm:"not null;default:true" json:"enabled"`
	AfterMinutes int    `gorm:"not null;default:10" json:"after_minutes"`
	// RepeatMinutes re-alerts while the episode is still running. 0 = one
	// alert per outage (the original behaviour).
	RepeatMinutes int `gorm:"not null;default:0" json:"repeat_minutes"`

	OfflineSince *time.Time `json:"offline_since"`
	AlertedAt    *time.Time `json:"last_alerted_at"`
	// SnoozedUntil suppresses alerts until this instant. It is deliberately
	// independent of the episode state: an owner silencing alerts for a
	// maintenance window should still get an accurate "offline since" when it
	// ends.
	SnoozedUntil *time.Time `json:"snoozed_until"`
	// MaintenanceWindows is a JSON array of recurring quiet windows, each
	// {"days":[0-6],"start":"HH:MM","end":"HH:MM"}. Empty string means none.
	// It reuses the same window shape (and therefore the same midnight-wrap
	// rules) as the device access-window feature so operators see one model.
	MaintenanceWindows string `gorm:"type:text;not null;default:''" json:"maintenance_windows"`
	// MaintenanceTimezone is the IANA zone the windows are evaluated in. Empty
	// means the server's local zone.
	MaintenanceTimezone string `gorm:"size:64;not null;default:''" json:"maintenance_timezone"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName overrides the default table name.
func (DeviceOfflineAlertRule) TableName() string {
	return "device_offline_alert_rules"
}
