package models

import "time"

// UpdateRelease is one build published to a self-hosted update channel. The
// server owns the index (which version exists for which platform/arch/channel,
// where the bytes live and what they hash to); the bytes themselves can live
// on any static host the operator controls.
//
// (channel, platform, arch, version) is unique: publishing the same version
// twice is a mistake, not a second release, and the client would otherwise get
// an arbitrary one of the duplicates.
type UpdateRelease struct {
	ID      uint   `gorm:"primaryKey" json:"id"`
	Channel string `gorm:"size:32;not null;uniqueIndex:idx_update_release_version,priority:1" json:"channel"`
	// Platform is windows / macos / linux; Arch is x64 / arm64 / universal.
	Platform string `gorm:"size:32;not null;uniqueIndex:idx_update_release_version,priority:2" json:"platform"`
	Arch     string `gorm:"size:32;not null;uniqueIndex:idx_update_release_version,priority:3" json:"arch"`
	// Version is the dotted numeric form the clients compare, e.g. 4.2.5.0.
	Version string `gorm:"size:64;not null;uniqueIndex:idx_update_release_version,priority:4" json:"version"`
	// DownloadURL is where the installer comes from; SHA256 is mandatory so a
	// self-hosted channel can verify what it downloaded.
	DownloadURL string    `gorm:"size:1024;not null" json:"download_url"`
	SHA256      string    `gorm:"size:64;not null" json:"sha256"`
	SizeBytes   int64     `gorm:"not null;default:0" json:"size_bytes"`
	Notes       string    `gorm:"type:text" json:"notes"`
	Mandatory   bool      `gorm:"not null;default:false" json:"mandatory"`
	PublishedAt time.Time `gorm:"not null;index" json:"published_at"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
