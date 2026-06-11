package models

import "time"

// User represents a QuickDesk end-user account.
//
// All JSON tags use snake_case to stay consistent with the rest of the
// v1 contract (devices, connections, favorites, etc.). Admin-web was
// the last holdout using camelCase (deviceCount / channelType / createdAt
// / updatedAt) — migrated to snake_case per architect review to avoid a
// split-brain across the admin and user surfaces.
type User struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Username    string    `gorm:"size:64;not null;uniqueIndex" json:"username"`
	Phone       string    `gorm:"size:32" json:"phone"`
	Email       string    `gorm:"size:128" json:"email"`
	Password    string    `gorm:"size:128" json:"-"` // bcrypt hash, never exposed in JSON
	Level       string    `gorm:"size:10;default:'V1'" json:"level"` // V1/V2/V3/V4/V5
	DeviceCount int       `gorm:"default:0" json:"device_count"`
	ChannelType string    `gorm:"size:20;default:'全球'" json:"channel_type"` // 全球/中国大陆
	Status      bool      `gorm:"default:true" json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TableName overrides the default table name.
func (User) TableName() string {
	return "users"
}
