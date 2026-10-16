package service

import (
	"context"
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"quickdesk/signaling/internal/models"

	"gorm.io/gorm"
)

// SessionRecordingService stores and retrieves user-owned recording metadata.
// Media production remains the responsibility of the remote-control engine.
type SessionRecordingService struct {
	db *gorm.DB
	// removeFile is a seam for tests; production deletes through os.Remove.
	removeFile func(string) error
}

func NewSessionRecordingService(db *gorm.DB) *SessionRecordingService {
	return &SessionRecordingService{db: db, removeFile: os.Remove}
}

type CreateSessionRecordingInput struct {
	DeviceID        string
	DeviceName      string
	SessionID       string
	Status          string
	Format          string
	ArtifactPath    string
	ArtifactURL     string
	ThumbnailPath   string
	DurationSeconds int
	SizeBytes       int64
	StartedAt       time.Time
	EndedAt         *time.Time
}

var ErrInvalidRecordingStatus = errors.New("status must be recording|completed|failed")

func ValidateSessionRecordingInput(in CreateSessionRecordingInput) error {
	if strings.TrimSpace(in.DeviceID) == "" {
		return errors.New("device_id is required")
	}
	if strings.TrimSpace(in.Status) == "" {
		return ErrInvalidRecordingStatus
	}
	switch strings.TrimSpace(in.Status) {
	case "recording", "completed", "failed":
	default:
		return ErrInvalidRecordingStatus
	}
	if in.DurationSeconds < 0 {
		return errors.New("duration_seconds must be >= 0")
	}
	if in.SizeBytes < 0 {
		return errors.New("size_bytes must be >= 0")
	}
	return nil
}

func (s *SessionRecordingService) Create(
	ctx context.Context,
	userID uint,
	in CreateSessionRecordingInput,
) (*models.SessionRecording, error) {
	in.DeviceID = strings.TrimSpace(in.DeviceID)
	in.DeviceName = strings.TrimSpace(in.DeviceName)
	in.SessionID = strings.TrimSpace(in.SessionID)
	in.Status = strings.TrimSpace(in.Status)
	in.Format = strings.TrimSpace(in.Format)
	in.ArtifactPath = strings.TrimSpace(in.ArtifactPath)
	in.ArtifactURL = strings.TrimSpace(in.ArtifactURL)
	in.ThumbnailPath = strings.TrimSpace(in.ThumbnailPath)
	if err := ValidateSessionRecordingInput(in); err != nil {
		return nil, err
	}
	startedAt := in.StartedAt.UTC()
	if startedAt.IsZero() {
		startedAt = time.Now().UTC()
	}
	now := time.Now().UTC()
	row := &models.SessionRecording{
		UserID:          userID,
		DeviceID:        in.DeviceID,
		DeviceName:      in.DeviceName,
		SessionID:       in.SessionID,
		Status:          in.Status,
		Format:          in.Format,
		ArtifactPath:    in.ArtifactPath,
		ArtifactURL:     in.ArtifactURL,
		ThumbnailPath:   in.ThumbnailPath,
		DurationSeconds: in.DurationSeconds,
		SizeBytes:       in.SizeBytes,
		StartedAt:       startedAt,
		EndedAt:         in.EndedAt,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := s.db.WithContext(ctx).Create(row).Error; err != nil {
		return nil, err
	}
	return row, nil
}

type SessionRecordingListParams struct {
	BeforeID uint
	Limit    int
}

type UpdateSessionRecordingInput struct {
	Status          *string
	Format          *string
	ArtifactPath    *string
	ArtifactURL     *string
	ThumbnailPath   *string
	DurationSeconds *int
	SizeBytes       *int64
	EndedAt         *time.Time
}

func (s *SessionRecordingService) Update(
	ctx context.Context,
	userID uint,
	id uint,
	in UpdateSessionRecordingInput,
) (*models.SessionRecording, error) {
	updates := map[string]interface{}{}
	if in.Status != nil {
		status := strings.TrimSpace(*in.Status)
		switch status {
		case "recording", "completed", "failed":
			updates["status"] = status
		default:
			return nil, ErrInvalidRecordingStatus
		}
	}
	if in.Format != nil {
		updates["format"] = strings.TrimSpace(*in.Format)
	}
	if in.ArtifactPath != nil {
		updates["artifact_path"] = strings.TrimSpace(*in.ArtifactPath)
	}
	if in.ArtifactURL != nil {
		updates["artifact_url"] = strings.TrimSpace(*in.ArtifactURL)
	}
	if in.ThumbnailPath != nil {
		updates["thumbnail_path"] = strings.TrimSpace(*in.ThumbnailPath)
	}
	if in.DurationSeconds != nil {
		if *in.DurationSeconds < 0 {
			return nil, errors.New("duration_seconds must be >= 0")
		}
		updates["duration_seconds"] = *in.DurationSeconds
	}
	if in.SizeBytes != nil {
		if *in.SizeBytes < 0 {
			return nil, errors.New("size_bytes must be >= 0")
		}
		updates["size_bytes"] = *in.SizeBytes
	}
	if in.EndedAt != nil {
		updates["ended_at"] = in.EndedAt.UTC()
	}
	if len(updates) == 0 {
		return s.Get(ctx, userID, id)
	}
	updates["updated_at"] = time.Now().UTC()
	result := s.db.WithContext(ctx).
		Model(&models.SessionRecording{}).
		Where("id = ? AND user_id = ?", id, userID).
		Updates(updates)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return s.Get(ctx, userID, id)
}

func (s *SessionRecordingService) List(
	ctx context.Context,
	userID uint,
	p SessionRecordingListParams,
) ([]models.SessionRecording, error) {
	q := s.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("id DESC")
	if p.BeforeID > 0 {
		q = q.Where("id < ?", p.BeforeID)
	}
	limit := p.Limit
	if limit <= 0 {
		limit = 50
	}
	var rows []models.SessionRecording
	if err := q.Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *SessionRecordingService) Get(
	ctx context.Context,
	userID uint,
	id uint,
) (*models.SessionRecording, error) {
	var row models.SessionRecording
	err := s.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *SessionRecordingService) Delete(ctx context.Context, userID uint, id uint) error {
	var row models.SessionRecording
	if err := s.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		First(&row).Error; err != nil {
		return err
	}
	result := s.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		Delete(&models.SessionRecording{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	// The row is the only record of where the bytes live, so the files have to
	// go with it - otherwise every delete leaks a video nothing can ever
	// reclaim (the retention sweep works from rows).
	s.removeStoredMedia(row)
	return nil
}

// removeStoredMedia drops the media files a row pointed at. A failure is
// reported but not fatal: the row is already gone, and the lifecycle's orphan
// pass will retry the deletion later.
func (s *SessionRecordingService) removeStoredMedia(row models.SessionRecording) {
	// The `.part` siblings matter for the resumable upload path: a client can
	// stop halfway, and the row is then the only thing that knows a staging file
	// exists. Without this, deleting a half-uploaded recording would leave its
	// partial bytes on disk forever.
	paths := make([]string, 0, 4)
	for _, path := range []string{row.ArtifactPath, row.ThumbnailPath} {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		paths = append(paths, path, path+".part")
	}
	for _, path := range paths {
		if err := s.removeFile(path); err != nil && !os.IsNotExist(err) {
			log.Printf("[Recording] could not delete stored media %s: %v", path, err)
		}
	}
}

// Exists reports whether a recording id is still indexed.
func (s *SessionRecordingService) Exists(ctx context.Context, id uint) (bool, error) {
	var count int64
	err := s.db.WithContext(ctx).
		Model(&models.SessionRecording{}).
		Where("id = ?", id).
		Count(&count).Error
	return count > 0, err
}

// SessionRecordingAdminListParams filters the cross-user recording index used
// by the admin console.
type SessionRecordingAdminListParams struct {
	AfterID  uint
	Limit    int
	UserID   uint
	DeviceID string
	Status   string
	Search   string
}

// ListAdmin returns recordings across all users plus the total match count.
func (s *SessionRecordingService) ListAdmin(
	ctx context.Context,
	p SessionRecordingAdminListParams,
) ([]models.SessionRecording, int64, error) {
	q := s.db.WithContext(ctx).Model(&models.SessionRecording{})
	if p.UserID > 0 {
		q = q.Where("user_id = ?", p.UserID)
	}
	if p.DeviceID != "" {
		q = q.Where("device_id = ?", strings.TrimSpace(p.DeviceID))
	}
	if p.Status != "" {
		q = q.Where("status = ?", strings.TrimSpace(p.Status))
	}
	if p.Search != "" {
		like := "%" + strings.TrimSpace(p.Search) + "%"
		q = q.Where("device_name LIKE ? OR session_id LIKE ?", like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if p.AfterID > 0 {
		q = q.Where("id > ?", p.AfterID)
	}
	limit := p.Limit
	if limit <= 0 {
		limit = 50
	}
	var rows []models.SessionRecording
	if err := q.Order("id ASC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// GetAdmin loads a single recording regardless of owner.
func (s *SessionRecordingService) GetAdmin(ctx context.Context, id uint) (*models.SessionRecording, error) {
	var row models.SessionRecording
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// DeleteAdmin removes a recording regardless of owner.
func (s *SessionRecordingService) DeleteAdmin(ctx context.Context, id uint) error {
	var row models.SessionRecording
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		return err
	}
	result := s.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&models.SessionRecording{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	s.removeStoredMedia(row)
	return nil
}

// ---------------------------------------------------------------------
// Media artifacts
//
// The control plane never inspects media bytes; it accepts an uploaded
// artifact, stores it outside any statically served directory, and hands it
// back through an ownership-checked route.
// ---------------------------------------------------------------------

var (
	ErrArtifactTooLarge   = errors.New("artifact exceeds the configured upload limit")
	ErrArtifactEmpty      = errors.New("artifact is empty")
	ErrArtifactTypeDenied = errors.New("unsupported artifact extension")
)

// artifactExtensions is an allow-list. Recordings are media files, never
// executables, so anything unrecognised is rejected instead of being stored
// under a guessed name.
var artifactExtensions = map[string]string{
	".mp4":  "mp4",
	".m4v":  "mp4",
	".mkv":  "mkv",
	".webm": "webm",
	".h264": "h264",
	".grf":  "grf",
}

// NormalizeArtifactExtension returns the lower-cased extension when it is on
// the allow-list, including the leading dot.
func NormalizeArtifactExtension(filename string) (string, error) {
	ext := strings.ToLower(filepath.Ext(strings.TrimSpace(filename)))
	if _, ok := artifactExtensions[ext]; !ok {
		return "", ErrArtifactTypeDenied
	}
	return ext, nil
}

// ArtifactFormat maps a stored extension back to the value persisted in the
// `format` column.
func ArtifactFormat(ext string) string {
	return artifactExtensions[strings.ToLower(ext)]
}

// ArtifactPath builds the on-disk location for one recording. Each user gets
// its own directory and the file name is derived from the recording id, so a
// hostile upload filename can never influence the resulting path.
func ArtifactPath(baseDir string, userID uint, recordingID uint, ext string) string {
	name := "rec-" + formatUintID(recordingID) + ext
	return filepath.Join(baseDir, formatUintID(userID), name)
}

// ArtifactURL is the relative route the clients resolve against the server
// base URL. Storing the route (rather than an absolute URL) keeps the value
// valid no matter which host name the deployment is reached by.
func ArtifactURL(recordingID uint) string {
	return "/v1/me/recordings/" + formatUintID(recordingID) + "/artifact"
}

func formatUintID(v uint) string {
	if v == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[i:])
}

// AttachArtifact records the stored media file on an existing row. The caller
// has already written the bytes to disk; this only updates the index.
func (s *SessionRecordingService) AttachArtifact(
	ctx context.Context,
	userID uint,
	id uint,
	storedPath string,
	ext string,
	sizeBytes int64,
) (*models.SessionRecording, error) {
	if sizeBytes <= 0 {
		return nil, ErrArtifactEmpty
	}
	format := ArtifactFormat(ext)
	if format == "" {
		return nil, ErrArtifactTypeDenied
	}
	updates := map[string]interface{}{
		"artifact_path": storedPath,
		"artifact_url":  ArtifactURL(id),
		"size_bytes":    sizeBytes,
		"format":        format,
		"updated_at":    time.Now().UTC(),
	}
	result := s.db.WithContext(ctx).
		Model(&models.SessionRecording{}).
		Where("id = ? AND user_id = ?", id, userID).
		Updates(updates)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	row, err := s.Get(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	// Once the bytes are on the server "completed" is the truth, whatever the row
	// said before: without this, re-uploading a capture that failed earlier would
	// store the file and still leave the row reading "failed". A client that
	// reports failure after this point still wins, because its PATCH lands later.
	if row.Status != "completed" {
		row, err = s.Update(ctx, userID, id, UpdateSessionRecordingInput{Status: strPtr("completed")})
		if err != nil {
			return nil, err
		}
	}
	return row, nil
}

// ArtifactFor loads a recording and asserts ownership before the handler
// streams bytes. Returning the row keeps the handler free of ownership logic.
func (s *SessionRecordingService) ArtifactFor(ctx context.Context, userID uint, id uint) (*models.SessionRecording, error) {
	row, err := s.Get(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(row.ArtifactPath) == "" {
		return nil, gorm.ErrRecordNotFound
	}
	return row, nil
}

// ArtifactForAdmin is the cross-user variant used by the admin download route.
func (s *SessionRecordingService) ArtifactForAdmin(ctx context.Context, id uint) (*models.SessionRecording, error) {
	row, err := s.GetAdmin(ctx, id)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(row.ArtifactPath) == "" {
		return nil, gorm.ErrRecordNotFound
	}
	return row, nil
}

func strPtr(v string) *string { return &v }

// ---------------------------------------------------------------------
// Thumbnails
//
// A thumbnail is a small still image used by the recordings list. It shares the
// artifact storage rules (allow-list, derived file name, no static mount) but is
// validated by content, not just by extension: an image route is a much more
// attractive place to smuggle something than a video one.
// ---------------------------------------------------------------------

var thumbnailExtensions = map[string]string{
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
}

// ErrInvalidThumbnailContent means the bytes do not look like the declared type.
var ErrInvalidThumbnailContent = errors.New("thumbnail bytes do not match an allowed image type")

// NormalizeThumbnailExtension accepts jpg/jpeg/png.
func NormalizeThumbnailExtension(filename string) (string, error) {
	ext := strings.ToLower(filepath.Ext(strings.TrimSpace(filename)))
	if _, ok := thumbnailExtensions[ext]; !ok {
		return "", ErrArtifactTypeDenied
	}
	return ext, nil
}

// ThumbnailContentType maps a stored extension to the response content type.
func ThumbnailContentType(ext string) string {
	return thumbnailExtensions[strings.ToLower(ext)]
}

// ThumbnailExtensionForType returns the canonical on-disk extension for a
// sniffed image content type, or "" when the type is not allowed.
func ThumbnailExtensionForType(contentType string) string {
	switch strings.ToLower(strings.TrimSpace(contentType)) {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	}
	return ""
}

// SniffThumbnailType verifies the leading bytes of an upload.
func SniffThumbnailType(head []byte) (string, error) {
	if len(head) >= 3 && head[0] == 0xFF && head[1] == 0xD8 && head[2] == 0xFF {
		return "image/jpeg", nil
	}
	if len(head) >= 8 && head[0] == 0x89 && head[1] == 'P' && head[2] == 'N' && head[3] == 'G' &&
		head[4] == 0x0D && head[5] == 0x0A && head[6] == 0x1A && head[7] == 0x0A {
		return "image/png", nil
	}
	return "", ErrInvalidThumbnailContent
}

// ThumbnailPath mirrors ArtifactPath: the name comes from the ids, never from
// the uploaded file name.
func ThumbnailPath(baseDir string, userID uint, recordingID uint, ext string) string {
	name := "thumb-" + formatUintID(recordingID) + ext
	return filepath.Join(baseDir, formatUintID(userID), name)
}

// ThumbnailURL is the relative route clients resolve against the server base.
func ThumbnailURL(recordingID uint) string {
	return "/v1/me/recordings/" + formatUintID(recordingID) + "/thumbnail"
}

// AttachThumbnail records the stored preview image on an existing row.
func (s *SessionRecordingService) AttachThumbnail(
	ctx context.Context,
	userID uint,
	id uint,
	storedPath string,
	ext string,
	sizeBytes int64,
) (*models.SessionRecording, error) {
	if sizeBytes <= 0 {
		return nil, ErrArtifactEmpty
	}
	if ThumbnailContentType(ext) == "" {
		return nil, ErrArtifactTypeDenied
	}
	result := s.db.WithContext(ctx).
		Model(&models.SessionRecording{}).
		Where("id = ? AND user_id = ?", id, userID).
		Updates(map[string]interface{}{
			"thumbnail_path": storedPath,
			"thumbnail_url":  ThumbnailURL(id),
			"updated_at":     time.Now().UTC(),
		})
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return s.Get(ctx, userID, id)
}

// ThumbnailFor returns the row when it owns a preview image.
func (s *SessionRecordingService) ThumbnailFor(ctx context.Context, userID uint, id uint) (*models.SessionRecording, error) {
	row, err := s.Get(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(row.ThumbnailPath) == "" {
		return nil, gorm.ErrRecordNotFound
	}
	return row, nil
}

// ThumbnailForAdmin is the cross-user variant.
func (s *SessionRecordingService) ThumbnailForAdmin(ctx context.Context, id uint) (*models.SessionRecording, error) {
	row, err := s.GetAdmin(ctx, id)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(row.ThumbnailPath) == "" {
		return nil, gorm.ErrRecordNotFound
	}
	return row, nil
}
