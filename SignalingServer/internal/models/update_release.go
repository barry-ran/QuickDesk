package models

import "time"

// UpdateRelease is one build published to a self-hosted update channel. The
// server owns the index (which version exists for which platform/arch/channel,
// where the bytes live and what they hash to); the bytes can live on any static
// host the operator controls, or on this server when the operator uploads them.
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
	// self-hosted channel can verify what it downloaded. Empty means the release
	// is not fetchable yet - it was published without an external URL and no
	// artifact has been uploaded - and the check route skips it, because
	// advertising an update nobody can download is worse than reporting none.
	DownloadURL string    `gorm:"size:1024;not null" json:"download_url"`
	SHA256      string    `gorm:"size:64;not null" json:"sha256"`
	SizeBytes   int64     `gorm:"not null;default:0" json:"size_bytes"`
	Notes       string    `gorm:"type:text" json:"notes"`
	Mandatory   bool      `gorm:"not null;default:false" json:"mandatory"`
	PublishedAt time.Time `gorm:"not null;index" json:"published_at"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	// ArtifactPath is where a server-hosted installer lives on disk. It is set
	// only after a successful upload to
	// POST /v1/admin/update-releases/:id/artifact. Never serialised: the
	// filesystem layout is not part of the contract.
	ArtifactPath string `gorm:"size:1024;not null;default:''" json:"-"`
	// HostedHere is derived from ArtifactPath (not a column) so clients and the
	// admin UI can tell a re-hosted release from an externally hosted one
	// without learning where the server keeps its files.
	HostedHere bool `gorm:"-" json:"hosted_here"`
}

// Decorate fills the derived fields. Every path that hands a release back to a
// caller must go through it, or HostedHere silently reports false.
func (r *UpdateRelease) Decorate() {
	if r == nil {
		return
	}
	r.HostedHere = r.ArtifactPath != ""
}
