package models

import "time"

type Settings struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Site
	SiteEnabled bool   `gorm:"default:true" json:"site_enabled"`
	SiteName    string `gorm:"size:100" json:"site_name"`
	LoginLogo   string `gorm:"size:500" json:"login_logo"`
	SmallLogo   string `gorm:"size:500" json:"small_logo"`
	Favicon     string `gorm:"size:500" json:"favicon"`

	// ICE / TURN / STUN (newline-separated in DB for cleaner storage)
	TurnURLs          string `gorm:"type:text" json:"turn_urls"`
	TurnAuthSecret    string `gorm:"size:500" json:"turn_auth_secret"`
	TurnCredentialTTL int    `gorm:"default:86400" json:"turn_credential_ttl"`
	StunURLs          string `gorm:"type:text" json:"stun_urls"`
	// Bumped each time an admin writes any TURN/STUN field. Hosts compare
	// this against the version echoed in heartbeat responses to decide
	// whether to refetch /v1/ice-config (§2.19).
	TurnConfigVersion int64 `gorm:"not null;default:1" json:"turn_config_version"`

	// Security
	APIKey           string `gorm:"size:500" json:"api_key"`
	AllowedOrigins   string `gorm:"type:text" json:"allowed_origins"`
	AdminIPWhitelist string `gorm:"type:text" json:"admin_ip_whitelist"`

	// Aliyun SMS
	SmsAccessKeyID     string `gorm:"size:200" json:"sms_access_key_id"`
	SmsAccessKeySecret string `gorm:"size:200" json:"sms_access_key_secret"`
	SmsSignName        string `gorm:"size:100" json:"sms_sign_name"`
	SmsTemplateCode    string `gorm:"size:100" json:"sms_template_code"`
}

func (Settings) TableName() string {
	return "settings"
}
