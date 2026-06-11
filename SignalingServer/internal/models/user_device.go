package models

import "time"

// UserDevice records the binding relationship between a user and a device.
// Created when the user binds/takes over a device (POST /v1/me/devices) and
// flipped to status=false on unbind or takeover. `remark` lets a user label
// devices they've connected to (e.g. "mom's laptop") without affecting the
// device's own display name.
type UserDevice struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	UserID        uint      `gorm:"not null;uniqueIndex:idx_user_device;index" json:"user_id"`
	DeviceID      string    `gorm:"size:9;not null;uniqueIndex:idx_user_device;index" json:"device_id"`
	Remark        string    `gorm:"size:128" json:"remark"`
	FirstBoundAt  time.Time `json:"first_bound_at"`
	LastConnectAt time.Time `json:"last_connect_at"`
	ConnectCount  int       `gorm:"default:0" json:"connect_count"`
	Status        bool      `gorm:"not null;default:true" json:"status"` // true = active binding, false = unbound/taken-over

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Preload target.
	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (UserDevice) TableName() string { return "user_devices" }
