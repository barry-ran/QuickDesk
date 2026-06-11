package service

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"time"

	"quickdesk/signaling/internal/models"
	"quickdesk/signaling/internal/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DeviceService owns the device lifecycle under the v1 API:
//   * Provision (搂2.5): allocate (device_id, device_secret) on first run
//     or rotate on machine_fingerprint mismatch.
//   * Heartbeat / last-seen bookkeeping.
//   * Access-code storage + verify.
//   * Secret verification (for device_secret Bearer middleware).
//
// All presence state (online/wsconn) lives in PresenceService; DeviceService
// only touches PostgreSQL.
type DeviceService struct {
	repo    *repository.DeviceRepository
	secrets *DeviceSecretService
}

func NewDeviceService(repo *repository.DeviceRepository, secrets *DeviceSecretService) *DeviceService {
	return &DeviceService{repo: repo, secrets: secrets}
}

// ProvisionRequest is the decoded body of POST /v1/devices:provision.
type ProvisionRequest struct {
	DeviceUUID         string `json:"device_uuid"`
	MachineFingerprint string `json:"machine_fingerprint,omitempty"`
	OS                 string `json:"os,omitempty"`
	OSVersion          string `json:"os_version,omitempty"`
	AppVersion         string `json:"app_version,omitempty"`
}

// ProvisionResult is what we return to the host. The DeviceSecret is the
// plaintext; callers MUST treat it as sensitive and return it to the host
// exactly once.
type ProvisionResult struct {
	DeviceID     string
	DeviceSecret string
	IsNew        bool
}

// Provision allocates or rotates a device_secret for the given device_uuid.
// - New UUID 鈫?create device + allocate device_id + secret.
// - Existing UUID 鈫?keep device_id, user_id, access_code, etc. intact, but
//   rotate device_secret (old secret is invalidated).
func (s *DeviceService) Provision(ctx context.Context, req ProvisionRequest) (ProvisionResult, error) {
	if req.DeviceUUID == "" {
		return ProvisionResult{}, errors.New("device_uuid is required")
	}

	// 1. Generate a fresh plaintext secret + hash.
	plaintext, err := s.secrets.Generate()
	if err != nil {
		return ProvisionResult{}, fmt.Errorf("gen secret: %w", err)
	}
	hash, err := s.secrets.Hash(plaintext)
	if err != nil {
		return ProvisionResult{}, fmt.Errorf("hash secret: %w", err)
	}

	// 2. Existing UUID?
	existing, err := s.repo.GetByDeviceUUID(ctx, req.DeviceUUID)
	if err == nil {
		existing.DeviceSecretHash = hash
		if req.OS != "" {
			existing.OS = req.OS
		}
		if req.OSVersion != "" {
			existing.OSVersion = req.OSVersion
		}
		if req.AppVersion != "" {
			existing.AppVersion = req.AppVersion
		}
		if req.MachineFingerprint != "" {
			existing.MachineFingerprint = req.MachineFingerprint
		}
		existing.LastSeenAt = time.Now().UTC()
		if err := s.repo.Save(ctx, existing); err != nil {
			return ProvisionResult{}, fmt.Errorf("update existing device: %w", err)
		}
		log.Printf("[Device] Provision rotated secret for device_id=%s uuid=%s", existing.DeviceID, existing.DeviceUUID)
		return ProvisionResult{DeviceID: existing.DeviceID, DeviceSecret: plaintext, IsNew: false}, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return ProvisionResult{}, fmt.Errorf("lookup device_uuid: %w", err)
	}

	// 3. Brand new UUID 鈫?allocate fresh device_id.
	deviceID, err := s.allocateDeviceID(ctx)
	if err != nil {
		return ProvisionResult{}, err
	}

	device := &models.Device{
		DeviceID:           deviceID,
		DeviceUUID:         req.DeviceUUID,
		DeviceSecretHash:   hash,
		MachineFingerprint: req.MachineFingerprint,
		OS:                 req.OS,
		OSVersion:          req.OSVersion,
		AppVersion:         req.AppVersion,
		LastSeenAt:         time.Now().UTC(),
	}
	if err := s.repo.Create(ctx, device); err != nil {
		return ProvisionResult{}, fmt.Errorf("create device: %w", err)
	}
	log.Printf("[Device] Provisioned new device: device_id=%s uuid=%s", deviceID, req.DeviceUUID)
	return ProvisionResult{DeviceID: deviceID, DeviceSecret: plaintext, IsNew: true}, nil
}

// VerifyDeviceSecret is used by the Bearer-device-secret middleware.
func (s *DeviceService) VerifyDeviceSecret(ctx context.Context, deviceID, plaintext string) (bool, error) {
	d, err := s.repo.GetByDeviceID(ctx, deviceID)
	if err != nil {
		return false, err
	}
	if d.DeviceSecretHash == "" {
		return false, nil
	}
	return s.secrets.Verify(plaintext, d.DeviceSecretHash)
}

// RotateSecret generates a new plaintext secret (used by admin
// `POST /v1/admin/devices/:id/secret:rotate`). Returns the plaintext once.
func (s *DeviceService) RotateSecret(ctx context.Context, deviceID string) (string, error) {
	plaintext, err := s.secrets.Generate()
	if err != nil {
		return "", err
	}
	hash, err := s.secrets.Hash(plaintext)
	if err != nil {
		return "", err
	}
	if err := s.repo.SetDeviceSecretHash(ctx, deviceID, hash); err != nil {
		return "", err
	}
	return plaintext, nil
}

// Heartbeat updates DB last_seen_at + app_version/os (when provided).
// Presence key refresh is the caller's responsibility (PresenceService).
func (s *DeviceService) Heartbeat(ctx context.Context, deviceID, os, osVersion, appVersion string) error {
	if err := s.repo.UpdateLastSeen(ctx, deviceID); err != nil {
		return err
	}
	if os != "" || osVersion != "" || appVersion != "" {
		_ = s.repo.UpdateDeviceInfo(ctx, deviceID, os, osVersion, appVersion)
	}
	return nil
}

// SetAccessCode writes the plaintext access_code into DB. Called from
// PUT /v1/devices/:id/access-code (host-reported, Qt-uploaded).
func (s *DeviceService) SetAccessCode(ctx context.Context, deviceID, code string) error {
	return s.repo.SetAccessCode(ctx, deviceID, code)
}

// VerifyAccessCode does a constant-time comparison against the stored
// plaintext code in DB. Used by POST /v1/devices/:id/access-code:verify
// (and only there — host signaling auth goes through signal_token).
func (s *DeviceService) VerifyAccessCode(ctx context.Context, deviceID, code string) (deviceExists, matches bool, err error) {
	d, err := s.repo.GetByDeviceID(ctx, deviceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, false, nil
		}
		return false, false, err
	}
	if d.AccessCode == "" {
		return true, false, nil
	}
	// Constant-time compare (§2.16). subtle.ConstantTimeCompare already
	// handles the equal-length case safely (returns 0 for unequal lengths
	// without leaking timing).
	match := subtle.ConstantTimeCompare([]byte(d.AccessCode), []byte(code)) == 1
	return true, match, nil
}

// BindResult tells the caller what actually happened during BindToUser so
// it can publish the right event (device.bound vs. device.ownership.lost+
// device.added for a takeover).
type BindResult struct {
	Device        *models.Device
	PreviousOwner *uint // nil if the device was unbound before; non-nil on takeover
	AlreadyOwned  bool  // true if the caller already owned the device (idempotent POST)
}

// BindToUser implements POST /v1/me/devices: the current user claims the
// device, optionally taking it over from another owner.
//
// The whole flip happens inside a SERIALIZABLE-ish transaction with
// SELECT ... FOR UPDATE on the device row so two users racing to bind the
// same device see strict ordering (搂2.14).
func (s *DeviceService) BindToUser(ctx context.Context, deviceID string, userID uint) (BindResult, error) {
	var out BindResult
	err := s.repo.DB(ctx).Transaction(func(tx *gorm.DB) error {
		var d models.Device
		if err := tx.Clauses(lockForUpdate()).Where("device_id = ?", deviceID).First(&d).Error; err != nil {
			return err
		}

		// Idempotent: same user re-binding.
		if d.UserID != nil && *d.UserID == userID {
			d.LoggedIn = true
			if err := tx.Save(&d).Error; err != nil {
				return err
			}
			// Keep the user_devices row fresh (status=true).
			s.upsertUserDevice(tx, userID, deviceID)
			out = BindResult{Device: &d, AlreadyOwned: true}
			return nil
		}

		// Takeover: remember the previous owner so the handler can notify.
		var prev *uint
		if d.UserID != nil {
			prevCopy := *d.UserID
			prev = &prevCopy
			// Mark the previous owner's binding inactive.
			if err := tx.Model(&models.UserDevice{}).
				Where("user_id = ? AND device_id = ?", *d.UserID, deviceID).
				Update("status", false).Error; err != nil {
				return err
			}
		}

		d.UserID = &userID
		d.LoggedIn = true
		if err := tx.Save(&d).Error; err != nil {
			return err
		}
		s.upsertUserDevice(tx, userID, deviceID)
		out = BindResult{Device: &d, PreviousOwner: prev}
		return nil
	})
	return out, err
}

// upsertUserDevice creates or reactivates a (user_id, device_id) binding.
func (s *DeviceService) upsertUserDevice(tx *gorm.DB, userID uint, deviceID string) {
	var ud models.UserDevice
	err := tx.Where("user_id = ? AND device_id = ?", userID, deviceID).First(&ud).Error
	now := time.Now().UTC()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		tx.Create(&models.UserDevice{
			UserID:       userID,
			DeviceID:     deviceID,
			Status:       true,
			FirstBoundAt: now,
		})
		return
	}
	if err != nil {
		return
	}
	ud.Status = true
	tx.Save(&ud)
}

// UnbindFromUser implements DELETE /v1/me/devices/:id: clears user_id +
// logged_in on the device and flips the user_device row to inactive.
func (s *DeviceService) UnbindFromUser(ctx context.Context, deviceID string, userID uint) error {
	return s.repo.DB(ctx).Transaction(func(tx *gorm.DB) error {
		var d models.Device
		if err := tx.Clauses(lockForUpdate()).Where("device_id = ?", deviceID).First(&d).Error; err != nil {
			return err
		}
		if d.UserID == nil || *d.UserID != userID {
			return ErrDeviceNotOwned
		}
		d.UserID = nil
		d.LoggedIn = false
		if err := tx.Save(&d).Error; err != nil {
			return err
		}
		return tx.Model(&models.UserDevice{}).
			Where("user_id = ? AND device_id = ?", userID, deviceID).
			Update("status", false).Error
	})
}

// ClearSession implements DELETE /v1/me/devices/:id/session 鈥?flips
// logged_in=false while leaving ownership intact. Qt calls this as
// step 1 of the logout flow (搂2.11).
func (s *DeviceService) ClearSession(ctx context.Context, deviceID string, userID uint) error {
	return s.repo.DB(ctx).Transaction(func(tx *gorm.DB) error {
		var d models.Device
		if err := tx.Clauses(lockForUpdate()).Where("device_id = ?", deviceID).First(&d).Error; err != nil {
			return err
		}
		if d.UserID == nil || *d.UserID != userID {
			return ErrDeviceNotOwned
		}
		d.LoggedIn = false
		return tx.Save(&d).Error
	})
}

// PatchMeta partially updates device metadata the owner controls.
type PatchMetaInput struct {
	DeviceName *string
	Remark      *string // goes on user_devices, not devices
}

// PatchMeta updates device_name on devices and/or remark on user_devices.
// The caller must be the current owner; we enforce that inside the tx.
func (s *DeviceService) PatchMeta(ctx context.Context, deviceID string, userID uint, in PatchMetaInput) error {
	return s.repo.DB(ctx).Transaction(func(tx *gorm.DB) error {
		var d models.Device
		if err := tx.Where("device_id = ?", deviceID).First(&d).Error; err != nil {
			return err
		}
		if d.UserID == nil || *d.UserID != userID {
			return ErrDeviceNotOwned
		}
		if in.DeviceName != nil {
			if err := tx.Model(&models.Device{}).
				Where("device_id = ?", deviceID).
				Update("device_name", *in.DeviceName).Error; err != nil {
				return err
			}
		}
		if in.Remark != nil {
			s.upsertUserDevice(tx, userID, deviceID)
			if err := tx.Model(&models.UserDevice{}).
				Where("user_id = ? AND device_id = ?", userID, deviceID).
				Update("remark", *in.Remark).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// ErrDeviceNotOwned is returned when a per-user endpoint is invoked with a
// device_id that doesn't belong to the caller.
var ErrDeviceNotOwned = errors.New("device not owned by caller")

// AdminForceUnbind clears user_id + logged_in, used by
// POST /v1/admin/devices/:id/unbind.
func (s *DeviceService) AdminForceUnbind(ctx context.Context, deviceID string) (*uint, error) {
	var previousOwner *uint
	err := s.repo.DB(ctx).Transaction(func(tx *gorm.DB) error {
		var d models.Device
		if err := tx.Clauses(lockForUpdate()).Where("device_id = ?", deviceID).First(&d).Error; err != nil {
			return err
		}
		if d.UserID != nil {
			prev := *d.UserID
			previousOwner = &prev
			if err := tx.Model(&models.UserDevice{}).
				Where("user_id = ? AND device_id = ?", prev, deviceID).
				Update("status", false).Error; err != nil {
				return err
			}
		}
		d.UserID = nil
		d.LoggedIn = false
		return tx.Save(&d).Error
	})
	return previousOwner, err
}

func (s *DeviceService) GetByDeviceID(ctx context.Context, deviceID string) (*models.Device, error) {
	return s.repo.GetByDeviceID(ctx, deviceID)
}

func (s *DeviceService) GetByDeviceIDWithUser(ctx context.Context, deviceID string) (*models.Device, error) {
	return s.repo.GetByDeviceIDWithUser(ctx, deviceID)
}

func (s *DeviceService) ListByUser(ctx context.Context, userID uint) ([]models.Device, error) {
	return s.repo.ListByUser(ctx, userID)
}

// DeviceAdminListParams is re-exported from the repository so handlers
// don't have to import the repository package directly.
type DeviceAdminListParams = repository.AdminListParams

func (s *DeviceService) ListAdmin(ctx context.Context, p DeviceAdminListParams) ([]models.Device, int64, error) {
	return s.repo.ListAdmin(ctx, p)
}

func (s *DeviceService) CountSince(ctx context.Context, since time.Time) (int64, error) {
	return s.repo.CountSince(ctx, since)
}

func (s *DeviceService) Delete(ctx context.Context, deviceID string) error {
	return s.repo.Delete(ctx, deviceID)
}

// allocateDeviceID tries up to 10 cryptographically-random 9-digit IDs,
// falling back to UUID-derived uniqueness if collisions persist. A real
// collision at this sample space is essentially impossible but the retry
// loop is cheap insurance against a hot key.
func (s *DeviceService) allocateDeviceID(ctx context.Context) (string, error) {
	for i := 0; i < 10; i++ {
		id := randomNineDigit()
		_, err := s.repo.GetByDeviceID(ctx, id)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return id, nil
		}
		if err != nil {
			return "", fmt.Errorf("check device_id: %w", err)
		}
	}
	// Fallback using a UUID-derived numeric suffix.
	u := uuid.New()
	n := binary.BigEndian.Uint32(u[:4])
	fallback := fmt.Sprintf("%09d", n%1_000_000_000)
	if _, err := s.repo.GetByDeviceID(ctx, fallback); errors.Is(err, gorm.ErrRecordNotFound) {
		return fallback, nil
	}
	return "", errors.New("exhausted device_id allocation retries")
}

func randomNineDigit() string {
	var buf [4]byte
	_, _ = rand.Read(buf[:])
	n := binary.BigEndian.Uint32(buf[:])
	// 100_000_000 .. 999_999_999
	return fmt.Sprintf("%09d", 100_000_000+n%900_000_000)
}
