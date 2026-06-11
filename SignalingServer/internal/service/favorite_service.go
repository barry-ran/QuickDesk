package service

import (
	"context"
	"errors"
	"time"

	"quickdesk/signaling/internal/models"

	"gorm.io/gorm"
)

// FavoriteService implements the user-scoped /v1/me/favorites surface.
// Favorites are per-user bookmarks of remote devices, with an optional
// access_password cached so the user can connect without retyping it.
type FavoriteService struct {
	db *gorm.DB
}

func NewFavoriteService(db *gorm.DB) *FavoriteService {
	return &FavoriteService{db: db}
}

var (
	ErrFavoriteNotFound = errors.New("favorite not found")
	ErrFavoriteExists   = errors.New("favorite already exists for this device")
)

func (s *FavoriteService) List(ctx context.Context, userID uint) ([]models.UserFavorite, error) {
	var out []models.UserFavorite
	err := s.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("updated_at DESC").
		Find(&out).Error
	return out, err
}

type FavoriteInput struct {
	DeviceID       string
	DeviceName     string
	AccessPassword string
}

func (s *FavoriteService) Add(ctx context.Context, userID uint, in FavoriteInput) (*models.UserFavorite, error) {
	if in.DeviceID == "" {
		return nil, errors.New("device_id is required")
	}
	var existing models.UserFavorite
	err := s.db.WithContext(ctx).
		Where("user_id = ? AND device_id = ?", userID, in.DeviceID).
		First(&existing).Error
	if err == nil {
		return nil, ErrFavoriteExists
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	fav := &models.UserFavorite{
		UserID:         userID,
		DeviceID:       in.DeviceID,
		DeviceName:     in.DeviceName,
		AccessPassword: in.AccessPassword,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
	if err := s.db.WithContext(ctx).Create(fav).Error; err != nil {
		return nil, err
	}
	return fav, nil
}

type FavoritePatch struct {
	DeviceName     *string
	AccessPassword *string
}

func (s *FavoriteService) Update(ctx context.Context, userID uint, deviceID string, p FavoritePatch) (*models.UserFavorite, error) {
	var fav models.UserFavorite
	err := s.db.WithContext(ctx).
		Where("user_id = ? AND device_id = ?", userID, deviceID).
		First(&fav).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFavoriteNotFound
		}
		return nil, err
	}
	if p.DeviceName != nil {
		fav.DeviceName = *p.DeviceName
	}
	if p.AccessPassword != nil {
		fav.AccessPassword = *p.AccessPassword
	}
	fav.UpdatedAt = time.Now().UTC()
	if err := s.db.WithContext(ctx).Save(&fav).Error; err != nil {
		return nil, err
	}
	return &fav, nil
}

func (s *FavoriteService) Delete(ctx context.Context, userID uint, deviceID string) error {
	res := s.db.WithContext(ctx).
		Where("user_id = ? AND device_id = ?", userID, deviceID).
		Delete(&models.UserFavorite{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrFavoriteNotFound
	}
	return nil
}
