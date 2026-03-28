package models

import (
	"time"
)

// Preset stores server-side preset configuration (single-row table).
// Notice and Links are JSON strings supporting i18n (zh_CN / en_US).
type Preset struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Notice       string    `gorm:"type:text;default:''" json:"notice"`
	Links        string    `gorm:"type:text;default:''" json:"links"`
	MinVersion   string    `gorm:"size:20;default:''" json:"min_version"`
	WebclientURL string    `gorm:"size:500;default:''" json:"webclient_url"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (Preset) TableName() string {
	return "presets"
}
