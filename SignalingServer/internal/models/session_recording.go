package models

import "time"

// SessionRecording is the control-plane index for a recorded remote session.
// The host/client engine owns the media artifact; this row makes recordings
// discoverable, auditable, and deletable through the user API.
type SessionRecording struct {
	ID              uint       `gorm:"primaryKey" json:"id"`
	UserID          uint       `gorm:"not null;index" json:"user_id"`
	DeviceID        string     `gorm:"size:9;not null;index" json:"device_id"`
	DeviceName      string     `gorm:"size:128" json:"device_name"`
	SessionID       string     `gorm:"size:64;index" json:"session_id"`
	Status          string     `gorm:"size:20;not null;index" json:"status"` // recording / completed / failed
	Format          string     `gorm:"size:20" json:"format"`                // mp4 / h264 / grf
	ArtifactPath    string     `gorm:"size:500" json:"artifact_path"`
	ArtifactURL     string     `gorm:"size:1000" json:"artifact_url"`
	ThumbnailPath   string     `gorm:"size:500" json:"thumbnail_path"`
	ThumbnailURL    string     `gorm:"size:1000" json:"thumbnail_url"`
	DurationSeconds int        `json:"duration_seconds"`
	SizeBytes       int64      `json:"size_bytes"`
	StartedAt       time.Time  `json:"started_at"`
	EndedAt         *time.Time `json:"ended_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`

	// BelongsTo
	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (SessionRecording) TableName() string {
	return "session_recordings"
}
