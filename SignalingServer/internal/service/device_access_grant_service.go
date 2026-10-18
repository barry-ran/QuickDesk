package service

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"quickdesk/signaling/internal/models"

	"gorm.io/gorm"
)

// ErrInvalidAccessGrant means the requested grant is malformed or would never
// admit anyone (non-positive lifetime, negative use cap, or unusable binding).
var ErrInvalidAccessGrant = errors.New("access grants need expires_in_minutes > 0, max_uses >= 0, bound_ips entries that are IPs or CIDR blocks, and a bound client public key (if any) alongside a client id")

// grantCodeAlphabet avoids characters that are easy to confuse when someone
// reads a code aloud or retypes it (0/O, 1/I/L).
const grantCodeAlphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"

// DeviceAccessGrantService issues and redeems time-boxed access grants.
type DeviceAccessGrantService struct {
	db *gorm.DB
}

func NewDeviceAccessGrantService(db *gorm.DB) *DeviceAccessGrantService {
	return &DeviceAccessGrantService{db: db}
}

// CreateDeviceAccessGrantInput is the validated body of the create endpoint.
type CreateDeviceAccessGrantInput struct {
	Label            string
	ExpiresInMinutes int
	MaxUses          int
	Code             string
	// BoundClientID and BoundIPs optionally restrict who may redeem the code.
	// Both empty (the default) keeps the old "anyone holding the code" rule.
	BoundClientID string
	// BoundClientPublicKey upgrades the client-id binding to a proof: base64 of
	// a raw 32-byte Ed25519 public key whose holder must sign the challenge.
	BoundClientPublicKey string
	BoundIPs             []string
}

// Create issues a grant for one device.
func (s *DeviceAccessGrantService) Create(ctx context.Context, deviceID string, ownerID uint, in CreateDeviceAccessGrantInput) (*models.DeviceAccessGrant, error) {
	if in.ExpiresInMinutes <= 0 || in.MaxUses < 0 {
		return nil, ErrInvalidAccessGrant
	}
	if err := ValidateAllowIPs(in.BoundIPs); err != nil {
		return nil, fmt.Errorf("%w: bound_ips %s", ErrInvalidAccessGrant, err.Error())
	}
	if len(strings.TrimSpace(in.BoundClientID)) > 64 {
		return nil, fmt.Errorf("%w: bound_client_id is limited to 64 characters", ErrInvalidAccessGrant)
	}
	boundKey := strings.TrimSpace(in.BoundClientPublicKey)
	if boundKey != "" {
		// Validate on the way in rather than at redemption time: a grant whose
		// binding can never be satisfied is a grant nobody can use, and the
		// owner would only find out by handing the code to a guest.
		if _, err := ParseControllerPublicKey(boundKey); err != nil {
			return nil, fmt.Errorf("%w: %s", ErrInvalidAccessGrant, err.Error())
		}
		if strings.TrimSpace(in.BoundClientID) == "" {
			// Without a client id the binding would accept a signature from the
			// key while the "who" column stayed empty, which makes the audit
			// value and the binding disagree.
			return nil, fmt.Errorf("%w: bound_client_public_key requires bound_client_id", ErrInvalidAccessGrant)
		}
	}
	boundIPs, err := json.Marshal(normalizeAllowIPs(in.BoundIPs))
	if err != nil {
		return nil, err
	}
	code := strings.TrimSpace(in.Code)
	if code == "" {
		generated, err := generateGrantCode()
		if err != nil {
			return nil, err
		}
		code = generated
	}
	row := &models.DeviceAccessGrant{
		DeviceID:             deviceID,
		OwnerID:              ownerID,
		Code:                 code,
		Label:                strings.TrimSpace(in.Label),
		ExpiresAt:            time.Now().UTC().Add(time.Duration(in.ExpiresInMinutes) * time.Minute),
		MaxUses:              in.MaxUses,
		BoundClientID:        strings.TrimSpace(in.BoundClientID),
		BoundClientPublicKey: boundKey,
		BoundIPs:             string(boundIPs),
	}
	if err := s.db.WithContext(ctx).Create(row).Error; err != nil {
		return nil, err
	}
	return row, nil
}

// List returns every grant for a device, newest first.
func (s *DeviceAccessGrantService) List(ctx context.Context, deviceID string) ([]models.DeviceAccessGrant, error) {
	var rows []models.DeviceAccessGrant
	err := s.db.WithContext(ctx).
		Where("device_id = ?", deviceID).
		Order("id DESC").
		Find(&rows).Error
	return rows, err
}

// Get returns one grant scoped to its device, or (nil, nil) when it is not
// there.
func (s *DeviceAccessGrantService) Get(ctx context.Context, deviceID string, grantID uint) (*models.DeviceAccessGrant, error) {
	var row models.DeviceAccessGrant
	err := s.db.WithContext(ctx).
		Where("id = ? AND device_id = ?", grantID, deviceID).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// Revoke cancels a grant. Revoking an already-revoked grant is a no-op.
func (s *DeviceAccessGrantService) Revoke(ctx context.Context, deviceID string, grantID uint) error {
	now := time.Now().UTC()
	return s.db.WithContext(ctx).
		Model(&models.DeviceAccessGrant{}).
		Where("id = ? AND device_id = ? AND revoked_at IS NULL", grantID, deviceID).
		Updates(map[string]interface{}{"revoked_at": now, "updated_at": now}).Error
}

// Match returns the usable grant whose code equals the presented one *and*
// whose binding admits this controller, or nil when none does.
//
// The candidates are loaded for the device and compared with
// subtle.ConstantTimeCompare rather than looked up by code, so the redeem path
// keeps the same constant-time habit as the device access code (and does not
// turn the code index into an oracle).
//
// A grant whose code matches but whose binding does not is treated exactly
// like an unknown code: the caller keeps its existing "invalid code" response,
// so a leaked code does not reveal that it was real.
func (s *DeviceAccessGrantService) Match(ctx context.Context, deviceID, code, clientID, clientIP string) (*models.DeviceAccessGrant, error) {
	if strings.TrimSpace(code) == "" {
		return nil, nil
	}
	var rows []models.DeviceAccessGrant
	if err := s.db.WithContext(ctx).
		Where("device_id = ? AND revoked_at IS NULL AND expires_at > ?", deviceID, time.Now().UTC()).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	presented := []byte(code)
	for i := range rows {
		if subtle.ConstantTimeCompare([]byte(rows[i].Code), presented) == 1 &&
			rows[i].UsableAt(time.Now().UTC()) &&
			GrantBindingAllows(rows[i], clientID, clientIP) {
			return &rows[i], nil
		}
	}
	return nil, nil
}

// GrantBindingAllows reports whether this controller may redeem the grant.
// An empty binding admits everyone, which is what grants meant before the
// fields existed.
//
// The client id half is client-asserted friction; the IP half is enforced
// against the address the server actually saw, so it survives a forwarded
// code.
func GrantBindingAllows(grant models.DeviceAccessGrant, clientID, clientIP string) bool {
	if bound := strings.TrimSpace(grant.BoundClientID); bound != "" {
		if strings.TrimSpace(clientID) != bound {
			return false
		}
	}
	boundIPs := decodeAllowIPs(grant.BoundIPs)
	if len(boundIPs) > 0 && !IPAllowed(boundIPs, clientIP) {
		return false
	}
	return true
}

// Consume records one started session against the grant. It re-checks the
// grant in the UPDATE itself, so two controllers racing the last use of a
// single-use grant cannot both win: the loser sees ok=false.
//
// The redeeming client is written in the same UPDATE, so the audit value can
// never disagree with the use count.
func (s *DeviceAccessGrantService) Consume(ctx context.Context, grantID uint, clientID, clientIP string) (bool, error) {
	now := time.Now().UTC()
	result := s.db.WithContext(ctx).
		Model(&models.DeviceAccessGrant{}).
		Where("id = ? AND revoked_at IS NULL AND expires_at > ? AND (max_uses = 0 OR uses < max_uses)",
			grantID, now).
		Updates(map[string]interface{}{
			"uses":                gorm.Expr("uses + 1"),
			"last_used_at":        now,
			"last_used_client_id": clipRunes(clientID, 64),
			"last_used_ip":        clipRunes(clientIP, 50),
			"updated_at":          now,
		})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected == 1, nil
}

func generateGrantCode() (string, error) {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate grant code: %w", err)
	}
	out := make([]byte, len(buf))
	for i, b := range buf {
		out[i] = grantCodeAlphabet[int(b)%len(grantCodeAlphabet)]
	}
	return string(out), nil
}

// clipRunes bounds a caller-supplied string to n characters so it cannot
// overflow its column. Postgres VARCHAR(n) counts characters, so this slices
// on runes rather than bytes.
func clipRunes(s string, n int) string {
	if n <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}
