package repository

import (
	"context"
	"quickdesk/signaling/internal/models"
	"time"

	"gorm.io/gorm"
)

type DeviceRepository struct {
	db *gorm.DB
}

func NewDeviceRepository(db *gorm.DB) *DeviceRepository {
	return &DeviceRepository{db: db}
}

// Create creates a new device
func (r *DeviceRepository) Create(ctx context.Context, device *models.Device) error {
	return r.db.WithContext(ctx).Create(device).Error
}

// GetByDeviceID retrieves a device by device_id
func (r *DeviceRepository) GetByDeviceID(ctx context.Context, deviceID string) (*models.Device, error) {
	var device models.Device
	err := r.db.WithContext(ctx).Where("device_id = ?", deviceID).First(&device).Error
	return &device, err
}

// GetByUUID retrieves a device by UUID
func (r *DeviceRepository) GetByUUID(ctx context.Context, uuid string) (*models.Device, error) {
	var device models.Device
	err := r.db.WithContext(ctx).Where("device_uuid = ?", uuid).First(&device).Error
	return &device, err
}

// Update updates a device
func (r *DeviceRepository) Update(ctx context.Context, device *models.Device) error {
	return r.db.WithContext(ctx).Save(device).Error
}

// SetOnline updates the online status of a device
func (r *DeviceRepository) SetOnline(ctx context.Context, deviceID string, online bool) error {
	return r.db.WithContext(ctx).Model(&models.Device{}).
		Where("device_id = ?", deviceID).
		Update("online", online).Error
}

// UpdateLastSeen updates the last_seen timestamp of a device
func (r *DeviceRepository) UpdateLastSeen(ctx context.Context, deviceID string) error {
	return r.db.WithContext(ctx).Model(&models.Device{}).
		Where("device_id = ?", deviceID).
		Update("last_seen", time.Now()).Error
}

// List retrieves devices with pagination
func (r *DeviceRepository) List(ctx context.Context, offset, limit int) ([]models.Device, error) {
	var devices []models.Device
	err := r.db.WithContext(ctx).Offset(offset).Limit(limit).Find(&devices).Error
	return devices, err
}
