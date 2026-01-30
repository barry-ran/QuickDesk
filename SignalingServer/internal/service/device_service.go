package service

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"quickdesk/signaling/internal/models"
	"quickdesk/signaling/internal/repository"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type DeviceService struct {
	repo  *repository.DeviceRepository
	redis *redis.Client
}

func NewDeviceService(repo *repository.DeviceRepository, redis *redis.Client) *DeviceService {
	return &DeviceService{
		repo:  repo,
		redis: redis,
	}
}

// GenerateUniqueDeviceID generates a unique 9-digit device ID
func (s *DeviceService) GenerateUniqueDeviceID(ctx context.Context) (string, error) {
	maxRetries := 10
	
	for i := 0; i < maxRetries; i++ {
		// Generate 9-digit random number (100000000-999999999)
		deviceID := fmt.Sprintf("%09d", rand.Intn(900000000)+100000000)
		
		// Check if already exists
		_, err := s.repo.GetByDeviceID(ctx, deviceID)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				// Not found, this ID is available
				log.Printf("Generated unique device_id: %s (attempt %d)", deviceID, i+1)
				return deviceID, nil
			}
			// Other error
			return "", fmt.Errorf("failed to check device ID: %w", err)
		}
		
		// ID already exists, retry
		log.Printf("Device ID collision: %s (attempt %d/%d)", deviceID, i+1, maxRetries)
	}
	
	// If all retries failed, use timestamp-based ID as fallback
	timestamp := time.Now().UnixMilli() % 1000000000
	deviceID := fmt.Sprintf("%09d", timestamp)
	
	// Final check
	_, err := s.repo.GetByDeviceID(ctx, deviceID)
	if err != nil && err == gorm.ErrRecordNotFound {
		log.Printf("Generated fallback device_id: %s", deviceID)
		return deviceID, nil
	}
	
	return "", fmt.Errorf("failed to generate unique device ID after %d retries", maxRetries)
}

// RegisterDeviceRequest represents a device registration request
type RegisterDeviceRequest struct {
	DeviceID   string `json:"device_id"`   // Optional: if provided, verify/use existing
	OS         string `json:"os"`
	OSVersion  string `json:"os_version"`
	AppVersion string `json:"app_version"`
}

// RegisterDeviceResponse represents the response for device registration
type RegisterDeviceResponse struct {
	DeviceID string `json:"device_id"`
	IsNew    bool   `json:"is_new"` // true if newly created, false if already existed
}

// RegisterDevice registers a new device or returns existing one
// Returns (device, isNew, error)
func (s *DeviceService) RegisterDevice(ctx context.Context, req *RegisterDeviceRequest) (*models.Device, bool, error) {
	// If device_id is provided, check if it exists
	if req.DeviceID != "" {
		existingDevice, err := s.repo.GetByDeviceID(ctx, req.DeviceID)
		if err == nil {
			// Device exists, update last_seen and return
			log.Printf("Device already exists: device_id=%s", req.DeviceID)
			s.repo.UpdateLastSeen(ctx, req.DeviceID)
			return existingDevice, false, nil
		}
		if err != gorm.ErrRecordNotFound {
			// Unexpected error
			return nil, false, fmt.Errorf("failed to check device: %w", err)
		}
		// Device not found, will create new one below
		log.Printf("Device ID %s not found in database, creating new device", req.DeviceID)
	}
	
	// Generate unique device ID if not provided
	deviceID := req.DeviceID
	if deviceID == "" {
		var err error
		deviceID, err = s.GenerateUniqueDeviceID(ctx)
		if err != nil {
			return nil, false, err
		}
	}
	
	device, err := s.RegisterDeviceWithID(ctx, deviceID, req)
	if err != nil {
		return nil, false, err
	}
	
	return device, true, nil
}

// RegisterDeviceWithID registers a device with a specific device ID
func (s *DeviceService) RegisterDeviceWithID(ctx context.Context, deviceID string, req *RegisterDeviceRequest) (*models.Device, error) {
	// Create device
	device := &models.Device{
		DeviceID:   deviceID,
		DeviceUUID: uuid.New().String(),
		OS:         req.OS,
		OSVersion:  req.OSVersion,
		AppVersion: req.AppVersion,
		Online:     false,
		LastSeen:   time.Now(),
	}
	
	if err := s.repo.Create(ctx, device); err != nil {
		return nil, fmt.Errorf("failed to create device: %w", err)
	}
	
	log.Printf("Device registered: device_id=%s, uuid=%s, os=%s",
		device.DeviceID, device.DeviceUUID, device.OS)
	
	return device, nil
}

// GetByDeviceID retrieves a device by device_id
func (s *DeviceService) GetByDeviceID(ctx context.Context, deviceID string) (*models.Device, error) {
	return s.repo.GetByDeviceID(ctx, deviceID)
}

// SetDeviceOnline sets a device's online status
func (s *DeviceService) SetDeviceOnline(ctx context.Context, deviceID string, online bool) error {
	if err := s.repo.SetOnline(ctx, deviceID, online); err != nil {
		return err
	}
	
	// Also cache in Redis for quick lookup
	key := fmt.Sprintf("device:online:%s", deviceID)
	if online {
		s.redis.Set(ctx, key, "1", 24*time.Hour)
	} else {
		s.redis.Del(ctx, key)
	}
	
	return nil
}

// IsDeviceOnline checks if a device is online
func (s *DeviceService) IsDeviceOnline(ctx context.Context, deviceID string) (bool, error) {
	// Check Redis cache first
	key := fmt.Sprintf("device:online:%s", deviceID)
	val, err := s.redis.Get(ctx, key).Result()
	if err == nil && val == "1" {
		return true, nil
	}
	
	// Fallback to database
	device, err := s.repo.GetByDeviceID(ctx, deviceID)
	if err != nil {
		return false, err
	}
	
	return device.Online, nil
}
