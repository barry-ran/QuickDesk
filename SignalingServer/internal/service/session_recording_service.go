package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"quickdesk/signaling/internal/models"

	"gorm.io/gorm"
)

// SessionRecordingService stores and retrieves user-owned recording metadata.
// Media production remains the responsibility of the remote-control engine.
type SessionRecordingService struct {
	db *gorm.DB
}

func NewSessionRecordingService(db *gorm.DB) *SessionRecordingService {
	return &SessionRecordingService{db: db}
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
	result := s.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		Delete(&models.SessionRecording{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
