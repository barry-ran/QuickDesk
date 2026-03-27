package models

import "time"

type Settings struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	// Site
	SiteEnabled bool   `gorm:"default:true" json:"siteEnabled"`
	SiteName    string `gorm:"size:100" json:"siteName"`
	LoginLogo   string `gorm:"size:500" json:"loginLogo"`
	SmallLogo   string `gorm:"size:500" json:"smallLogo"`
	Favicon     string `gorm:"size:500" json:"favicon"`

	// ICE / TURN / STUN (newline-separated in DB for cleaner storage)
	TurnURLs          string `gorm:"type:text" json:"turnUrls"`
	TurnAuthSecret    string `gorm:"size:500" json:"turnAuthSecret"`
	TurnCredentialTTL int    `gorm:"default:86400" json:"turnCredentialTtl"`
	StunURLs          string `gorm:"type:text" json:"stunUrls"`

	// Security
	APIKey         string `gorm:"size:500" json:"apiKey"`
	AllowedOrigins string `gorm:"type:text" json:"allowedOrigins"`

	// Aliyun SMS
	SmsAccessKeyID     string `gorm:"size:200" json:"smsAccessKeyId"`
	SmsAccessKeySecret string `gorm:"size:200" json:"smsAccessKeySecret"`
	SmsSignName        string `gorm:"size:100" json:"smsSignName"`
	SmsTemplateCode    string `gorm:"size:100" json:"smsTemplateCode"`
}

func (Settings) TableName() string {
	return "settings"
}
