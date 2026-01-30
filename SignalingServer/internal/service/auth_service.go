package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// Temporary password length (6 digits)
	TempPasswordLength = 6
	// Temporary password TTL (2 hours, as per design doc)
	TempPasswordTTL = 2 * time.Hour
)

type AuthService struct {
	redis *redis.Client
}

func NewAuthService(redis *redis.Client) *AuthService {
	return &AuthService{redis: redis}
}

// GenerateTemporaryPassword generates a 6-digit random password
func (s *AuthService) GenerateTemporaryPassword() string {
	return fmt.Sprintf("%06d", rand.Intn(1000000))
}

// SetTemporaryPassword stores a temporary password for a device in Redis
func (s *AuthService) SetTemporaryPassword(ctx context.Context, deviceID, password string) error {
	key := fmt.Sprintf("temp_password:%s", deviceID)
	
	// Store hashed password
	hashedPassword := s.hashPassword(deviceID, password)
	
	err := s.redis.Set(ctx, key, hashedPassword, TempPasswordTTL).Err()
	if err != nil {
		return fmt.Errorf("failed to set temporary password: %w", err)
	}
	
	log.Printf("Temporary password set for device_id=%s (TTL=%v)", deviceID, TempPasswordTTL)
	return nil
}

// VerifyTemporaryPassword verifies a temporary password for a device
func (s *AuthService) VerifyTemporaryPassword(ctx context.Context, deviceID, password string) (bool, error) {
	key := fmt.Sprintf("temp_password:%s", deviceID)
	
	storedHash, err := s.redis.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			log.Printf("No temporary password found for device_id=%s", deviceID)
			return false, nil
		}
		return false, fmt.Errorf("failed to get temporary password: %w", err)
	}
	
	// Verify password
	expectedHash := s.hashPassword(deviceID, password)
	match := hmac.Equal([]byte(storedHash), []byte(expectedHash))
	
	if match {
		log.Printf("Temporary password verified for device_id=%s", deviceID)
	} else {
		log.Printf("Temporary password mismatch for device_id=%s", deviceID)
	}
	
	return match, nil
}

// hashPassword hashes a password with device ID as salt
func (s *AuthService) hashPassword(deviceID, password string) string {
	h := hmac.New(sha256.New, []byte(deviceID))
	h.Write([]byte(password))
	return hex.EncodeToString(h.Sum(nil))
}

// VerifyDevice verifies device credentials (used for WebSocket connections)
func (s *AuthService) VerifyDevice(ctx context.Context, deviceID, accessCode string) bool {
	// Try temporary password first
	if match, err := s.VerifyTemporaryPassword(ctx, deviceID, accessCode); err == nil && match {
		return true
	}
	
	// TODO: Add permanent password verification (from database)
	
	return false
}

// ClearTemporaryPassword removes the temporary password for a device
// Called when Host disconnects
func (s *AuthService) ClearTemporaryPassword(ctx context.Context, deviceID string) error {
	key := fmt.Sprintf("temp_password:%s", deviceID)
	
	err := s.redis.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to clear temporary password: %w", err)
	}
	
	log.Printf("Temporary password cleared for device_id=%s", deviceID)
	return nil
}
