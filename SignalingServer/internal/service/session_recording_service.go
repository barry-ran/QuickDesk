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
