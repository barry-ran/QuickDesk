package models

import (
	"time"
)

// Device represents a QuickDesk host device
type Device struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	DeviceID   string    `gorm:"uniqueIndex;size:9;not null" json:"device_id"`       // 9位数字ID
	DeviceUUID string    `gorm:"uniqueIndex;not null" json:"device_uuid"`            // UUID
	OS         string    `gorm:"size:50" json:"os"`
	OSVersion  string    `gorm:"size:50" json:"os_version"`
	AppVersion string    `gorm:"size:20" json:"app_version"`
	Online     bool      `gorm:"default:false" json:"online"`
	LastSeen   time.Time `json:"last_seen"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// TableName overrides the table name
func (Device) TableName() string {
	return "devices"
}
