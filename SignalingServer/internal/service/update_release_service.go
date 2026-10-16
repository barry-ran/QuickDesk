package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"quickdesk/signaling/internal/models"

	"gorm.io/gorm"
)

// The self-hosted update channel. The server owns the index - which version
// exists for which platform/arch/channel, where the bytes live, what they hash
// to and whether the update is mandatory - and answers the one question a
// deployed client asks: "is there something newer than me?".
//
// The bytes themselves are not re-hosted here: DownloadURL is an absolute
// http(s) URL the operator controls (a CDN, object storage, or the same
// reverse proxy in front of this server). That keeps the channel a metadata
// service and leaves artifact storage to whatever already serves the installs.

const (
	UpdateChannelStable  = "stable"
	UpdateChannelBeta    = "beta"
	UpdateChannelNightly = "nightly"
	// DefaultUpdateChannel is what a client that does not name a channel gets.
	DefaultUpdateChannel = UpdateChannelStable
)

// UpdateReleaseValidationError marks a caller mistake, so handlers can answer
// 400 instead of 500 without string-matching.
type UpdateReleaseValidationError struct{ Message string }

func (e *UpdateReleaseValidationError) Error() string { return e.Message }

func updateInvalid(format string, args ...interface{}) error {
	return &UpdateReleaseValidationError{Message: fmt.Sprintf(format, args...)}
}

// ErrUpdateReleaseExists is returned when the same channel/platform/arch/version
// is published twice.
var ErrUpdateReleaseExists = errors.New("that version is already published for this channel, platform and arch")

var (
	updateChannels   = map[string]bool{UpdateChannelStable: true, UpdateChannelBeta: true, UpdateChannelNightly: true}
	updatePlatforms  = map[string]bool{"windows": true, "macos": true, "linux": true}
	updateArches     = map[string]bool{"x64": true, "arm64": true, "universal": true}
	maxVersionParts  = 8
	sha256HexPattern = func(s string) bool {
		if len(s) != 64 {
			return false
		}
		for _, r := range s {
			if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
				return false
			}
		}
		return true
	}
)

// ParseUpdateVersion turns a dotted numeric version (4.2.5.0) into its parts. A
// leading `v` is accepted because that is what release tags look like, but the
// parts themselves must be plain numbers - a channel that cannot order two
// versions cannot decide whether an update exists.
func ParseUpdateVersion(version string) ([]int64, error) {
	raw := strings.TrimSpace(version)
	raw = strings.TrimPrefix(strings.TrimPrefix(raw, "v"), "V")
	if raw == "" {
		return nil, updateInvalid("version is required")
	}
	parts := strings.Split(raw, ".")
	if len(parts) > maxVersionParts {
		return nil, updateInvalid("version has more than %d parts", maxVersionParts)
	}
	out := make([]int64, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			return nil, updateInvalid("version %q has an empty part", version)
		}
		value, err := strconv.ParseInt(part, 10, 64)
		if err != nil || value < 0 {
			return nil, updateInvalid("version %q must be dotted non-negative numbers, like 4.2.5.0", version)
		}
		out = append(out, value)
	}
	return out, nil
}

// CompareUpdateVersions returns -1 when a < b, 0 when they are equal and 1 when
// a > b. Missing trailing parts count as zero, so 4.2 and 4.2.0.0 are equal.
func CompareUpdateVersions(a, b string) (int, error) {
	left, err := ParseUpdateVersion(a)
	if err != nil {
		return 0, err
	}
	right, err := ParseUpdateVersion(b)
	if err != nil {
		return 0, err
	}
	width := len(left)
	if len(right) > width {
		width = len(right)
	}
	for i := 0; i < width; i++ {
		var l, r int64
		if i < len(left) {
			l = left[i]
		}
		if i < len(right) {
			r = right[i]
		}
		if l < r {
			return -1, nil
		}
		if l > r {
			return 1, nil
		}
	}
	return 0, nil
}

// UpdateReleaseFilter narrows an admin listing.
type UpdateReleaseFilter struct {
	Channel  string
	Platform string
	Arch     string
	Limit    int
}

// PublishUpdateReleaseInput is one admin publish request.
type PublishUpdateReleaseInput struct {
	Channel     string
	Platform    string
	Arch        string
	Version     string
	DownloadURL string
	SHA256      string
	SizeBytes   int64
	Notes       string
	Mandatory   bool
	PublishedAt time.Time
}

// UpdateCheckResult is the answer to a client asking whether it is behind.
type UpdateCheckResult struct {
	UpdateAvailable bool   `json:"update_available"`
	Mandatory       bool   `json:"mandatory"`
	Channel         string `json:"channel"`
	Platform        string `json:"platform"`
	Arch            string `json:"arch"`
	CurrentVersion  string `json:"current_version,omitempty"`
	// Latest is always in the response, null when the channel is empty, so a
	// checker never has to distinguish "no release" from "key missing".
	Latest *models.UpdateRelease `json:"latest"`
}

// UpdateReleaseService owns the update index.
type UpdateReleaseService struct{ db *gorm.DB }

func NewUpdateReleaseService(db *gorm.DB) *UpdateReleaseService {
	return &UpdateReleaseService{db: db}
}

func normalizeUpdateChannel(channel string) (string, error) {
	name := strings.ToLower(strings.TrimSpace(channel))
	if name == "" {
		name = DefaultUpdateChannel
	}
	if !updateChannels[name] {
		return "", updateInvalid("channel must be one of stable, beta or nightly")
	}
	return name, nil
}

func normalizeUpdatePlatform(platform string) (string, error) {
	name := strings.ToLower(strings.TrimSpace(platform))
	if name == "" {
		return "", updateInvalid("platform is required")
	}
	if !updatePlatforms[name] {
		return "", updateInvalid("platform must be one of windows, macos or linux")
	}
	return name, nil
}

func normalizeUpdateArch(arch string) (string, error) {
	name := strings.ToLower(strings.TrimSpace(arch))
	if name == "" {
		return "", updateInvalid("arch is required")
	}
	if !updateArches[name] {
		return "", updateInvalid("arch must be one of x64, arm64 or universal")
	}
	return name, nil
}

// Publish adds a release to the index. The same version may exist once per
// channel/platform/arch; publishing it again is a conflict rather than an
// update, so a fat-fingered re-run cannot silently replace an artifact.
func (s *UpdateReleaseService) Publish(ctx context.Context, in PublishUpdateReleaseInput) (*models.UpdateRelease, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("update release service is not configured")
	}
	channel, err := normalizeUpdateChannel(in.Channel)
	if err != nil {
		return nil, err
	}
	platform, err := normalizeUpdatePlatform(in.Platform)
	if err != nil {
		return nil, err
	}
	arch, err := normalizeUpdateArch(in.Arch)
	if err != nil {
		return nil, err
	}
	if _, err := ParseUpdateVersion(in.Version); err != nil {
		return nil, err
	}
	downloadURL := strings.TrimSpace(in.DownloadURL)
	parsed, err := url.Parse(downloadURL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, updateInvalid("download_url must be an absolute http(s) URL")
	}
	sha := strings.ToLower(strings.TrimSpace(in.SHA256))
	if !sha256HexPattern(sha) {
		return nil, updateInvalid("sha256 must be the 64-character hex digest of the artifact")
	}
	if in.SizeBytes < 0 {
		return nil, updateInvalid("size_bytes must not be negative")
	}
	publishedAt := in.PublishedAt.UTC()
	if in.PublishedAt.IsZero() {
		publishedAt = time.Now().UTC()
	}

	release := models.UpdateRelease{
		Channel:     channel,
		Platform:    platform,
		Arch:        arch,
		Version:     strings.TrimSpace(in.Version),
		DownloadURL: downloadURL,
		SHA256:      sha,
		SizeBytes:   in.SizeBytes,
		Notes:       strings.TrimSpace(in.Notes),
		Mandatory:   in.Mandatory,
		PublishedAt: publishedAt,
	}
	if err := s.db.WithContext(ctx).Create(&release).Error; err != nil {
		if isUniqueViolation(err) {
			return nil, ErrUpdateReleaseExists
		}
		return nil, err
	}
	return &release, nil
}

// List returns the newest releases first. Filters are exact matches; an empty
// filter lists the whole index.
func (s *UpdateReleaseService) List(ctx context.Context, filter UpdateReleaseFilter) ([]models.UpdateRelease, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("update release service is not configured")
	}
	query := s.db.WithContext(ctx).Model(&models.UpdateRelease{})
	if trimmed := strings.TrimSpace(filter.Channel); trimmed != "" {
		query = query.Where("channel = ?", strings.ToLower(trimmed))
	}
	if trimmed := strings.TrimSpace(filter.Platform); trimmed != "" {
		query = query.Where("platform = ?", strings.ToLower(trimmed))
	}
	if trimmed := strings.TrimSpace(filter.Arch); trimmed != "" {
		query = query.Where("arch = ?", strings.ToLower(trimmed))
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = 200
	}
	if limit > 1000 {
		limit = 1000
	}
	var out []models.UpdateRelease
	if err := query.Order("published_at DESC, id DESC").Limit(limit).Find(&out).Error; err != nil {
		return nil, err
	}
	if out == nil {
		out = []models.UpdateRelease{}
	}
	return out, nil
}

// Delete removes one release. A missing id is reported so an operator does not
// think a typo deleted something.
func (s *UpdateReleaseService) Delete(ctx context.Context, id uint) error {
	if s == nil || s.db == nil {
		return errors.New("update release service is not configured")
	}
	result := s.db.WithContext(ctx).Delete(&models.UpdateRelease{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// Latest resolves the newest release for one channel/platform/arch.
func (s *UpdateReleaseService) Latest(ctx context.Context, platform, arch, channel string) (*models.UpdateRelease, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("update release service is not configured")
	}
	normalizedPlatform, err := normalizeUpdatePlatform(platform)
	if err != nil {
		return nil, err
	}
	normalizedArch, err := normalizeUpdateArch(arch)
	if err != nil {
		return nil, err
	}
	normalizedChannel, err := normalizeUpdateChannel(channel)
	if err != nil {
		return nil, err
	}
	var release models.UpdateRelease
	err = s.db.WithContext(ctx).
		Where("channel = ? AND platform = ? AND arch = ?", normalizedChannel, normalizedPlatform, normalizedArch).
		Order("published_at DESC, id DESC").
		First(&release).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &release, nil
}

// Check answers the client's question. Without current_version it reports that
// a release exists; with one it only reports an update when the published
// version is strictly newer, so re-checking the same version is not an update.
func (s *UpdateReleaseService) Check(ctx context.Context, platform, arch, channel, currentVersion string) (UpdateCheckResult, error) {
	platformName, err := normalizeUpdatePlatform(platform)
	if err != nil {
		return UpdateCheckResult{}, err
	}
	archName, err := normalizeUpdateArch(arch)
	if err != nil {
		return UpdateCheckResult{}, err
	}
	channelName, err := normalizeUpdateChannel(channel)
	if err != nil {
		return UpdateCheckResult{}, err
	}
	current := strings.TrimSpace(currentVersion)
	if current != "" {
		if _, err := ParseUpdateVersion(current); err != nil {
			return UpdateCheckResult{}, err
		}
	}
	latest, err := s.Latest(ctx, platformName, archName, channelName)
	if err != nil {
		return UpdateCheckResult{}, err
	}
	result := UpdateCheckResult{
		Channel:        channelName,
		Platform:       platformName,
		Arch:           archName,
		CurrentVersion: current,
		Latest:         latest,
	}
	if latest == nil {
		return result, nil
	}
	if current == "" {
		result.UpdateAvailable = true
		result.Mandatory = latest.Mandatory
		return result, nil
	}
	comparison, err := CompareUpdateVersions(latest.Version, current)
	if err != nil {
		return UpdateCheckResult{}, err
	}
	if comparison > 0 {
		result.UpdateAvailable = true
		result.Mandatory = latest.Mandatory
	}
	return result, nil
}

// isUniqueViolation keeps the service from depending on the Postgres driver's
// concrete error type while still turning a duplicate publish into a conflict.
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "duplicate key") || strings.Contains(message, "unique constraint")
}
