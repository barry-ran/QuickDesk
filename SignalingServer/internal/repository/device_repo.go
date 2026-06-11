package repository

import (
	"context"
	"time"

	"quickdesk/signaling/internal/models"

	"gorm.io/gorm"
)

// DeviceRepository wraps the `devices` table for the v1 schema.
//
// Notable differences from the old repo:
//   - No SetOnline 鈥?`online` is derived from Redis; use PresenceService.
//   - SetLoggedIn flips the user's intent column (not the WS-derived
//     online value).
//   - Listing no longer filters by an `online` column; if a view needs
//     that, it computes it post-query via PresenceService.
type DeviceRepository struct {
	db *gorm.DB
}

func NewDeviceRepository(db *gorm.DB) *DeviceRepository {
	return &DeviceRepository{db: db}
}

func (r *DeviceRepository) DB(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx)
}

func (r *DeviceRepository) Create(ctx context.Context, device *models.Device) error {
	return r.db.WithContext(ctx).Create(device).Error
}

func (r *DeviceRepository) GetByDeviceID(ctx context.Context, deviceID string) (*models.Device, error) {
	var d models.Device
	err := r.db.WithContext(ctx).Where("device_id = ?", deviceID).First(&d).Error
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *DeviceRepository) GetByDeviceIDWithUser(ctx context.Context, deviceID string) (*models.Device, error) {
	var d models.Device
	err := r.db.WithContext(ctx).Preload("User").Where("device_id = ?", deviceID).First(&d).Error
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *DeviceRepository) GetByDeviceUUID(ctx context.Context, uuid string) (*models.Device, error) {
	var d models.Device
	err := r.db.WithContext(ctx).Where("device_uuid = ?", uuid).First(&d).Error
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *DeviceRepository) Save(ctx context.Context, device *models.Device) error {
	return r.db.WithContext(ctx).Save(device).Error
}

// UpdateLastSeen bumps last_seen_at to now; called from heartbeat and WS
// auth so the DB has a coarse non-Redis trail for audit/debug.
func (r *DeviceRepository) UpdateLastSeen(ctx context.Context, deviceID string) error {
	return r.db.WithContext(ctx).Model(&models.Device{}).
		Where("device_id = ?", deviceID).
		Update("last_seen_at", time.Now().UTC()).Error
}

func (r *DeviceRepository) UpdateDeviceInfo(ctx context.Context, deviceID, os, osVersion, appVersion string) error {
	updates := map[string]interface{}{}
	if os != "" {
		updates["os"] = os
	}
	if osVersion != "" {
		updates["os_version"] = osVersion
	}
	if appVersion != "" {
		updates["app_version"] = appVersion
	}
	if len(updates) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Model(&models.Device{}).
		Where("device_id = ?", deviceID).
		Updates(updates).Error
}

func (r *DeviceRepository) SetLoggedIn(ctx context.Context, deviceID string, intent bool) error {
	return r.db.WithContext(ctx).Model(&models.Device{}).
		Where("device_id = ?", deviceID).
		Update("logged_in", intent).Error
}

func (r *DeviceRepository) SetAccessCode(ctx context.Context, deviceID, code string) error {
	return r.db.WithContext(ctx).Model(&models.Device{}).
		Where("device_id = ?", deviceID).
		Update("access_code", code).Error
}

func (r *DeviceRepository) SetDeviceSecretHash(ctx context.Context, deviceID, hash string) error {
	return r.db.WithContext(ctx).Model(&models.Device{}).
		Where("device_id = ?", deviceID).
		Update("device_secret_hash", hash).Error
}

func (r *DeviceRepository) SetMachineFingerprint(ctx context.Context, deviceID, fp string) error {
	return r.db.WithContext(ctx).Model(&models.Device{}).
		Where("device_id = ?", deviceID).
		Update("machine_fingerprint", fp).Error
}

func (r *DeviceRepository) SetDeviceName(ctx context.Context, deviceID, name string) error {
	return r.db.WithContext(ctx).Model(&models.Device{}).
		Where("device_id = ?", deviceID).
		Update("device_name", name).Error
}

// ListByUser returns devices owned by the given user, newest first.
func (r *DeviceRepository) ListByUser(ctx context.Context, userID uint) ([]models.Device, error) {
	var list []models.Device
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("updated_at DESC").
		Find(&list).Error
	return list, err
}

// AdminListParams carries the cursor + filter knobs admin device list
// accepts. AfterID supports keyset-style "WHERE id < after" pagination.
type AdminListParams struct {
	AfterID     uint
	Limit       int
	Sort, Order string
	Search, OS  string
}

// ListAdmin pages through all devices for admin views using keyset
// (cursor) pagination. `search` matches device_id / device_name.
func (r *DeviceRepository) ListAdmin(ctx context.Context, p AdminListParams) ([]models.Device, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.Device{})
	if p.Search != "" {
		like := "%" + p.Search + "%"
		query = query.Where("device_id LIKE ? OR device_name LIKE ?", like, like)
	}
	if p.OS != "" {
		query = query.Where("os = ?", p.OS)
	}

	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if p.AfterID > 0 {
		query = query.Where("id < ?", p.AfterID)
	}
	sort := p.Sort
	if sort == "" {
		sort = "id"
	}
	order := p.Order
	if order == "" {
		order = "desc"
	}
	var list []models.Device
	err := query.Order(sort + " " + order).
		Limit(p.Limit).
		Preload("User").
		Find(&list).Error
	return list, total, err
}

func (r *DeviceRepository) CountSince(ctx context.Context, since time.Time) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&models.Device{}).
		Where("created_at >= ?", since).Count(&n).Error
	return n, err
}

// UnbindByUser clears user_id + logged_in for every device of a user.
// Used when a user is hard-deleted (搂2.4 bug闃叉姢).
func (r *DeviceRepository) UnbindByUser(ctx context.Context, userID uint) error {
	return r.db.WithContext(ctx).Model(&models.Device{}).
		Where("user_id = ?", userID).
		Updates(map[string]interface{}{
			"user_id":          nil,
			"logged_in": false,
		}).Error
}

func (r *DeviceRepository) Delete(ctx context.Context, deviceID string) error {
	return r.db.WithContext(ctx).Where("device_id = ?", deviceID).
		Delete(&models.Device{}).Error
}
