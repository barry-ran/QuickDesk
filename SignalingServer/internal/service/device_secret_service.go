package service

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

// DeviceSecretService encapsulates generation, hashing and verification of
// device_secret strings. The plaintext secret is ONLY returned to the host
// once, at provision (or rotate); subsequent verifications use the argon2id
// hash persisted in `devices.device_secret_hash`.
//
// Hash format is self-describing so we can tune parameters later without a
// migration:
//
//     argon2id$v=19$t=<time>$m=<memoryKB>$p=<parallelism>$<saltB64>$<hashB64>
type DeviceSecretService struct {
	time       uint32
	memoryKB   uint32
	threads    uint8
	keyLen     uint32
	saltLen    uint32
	secretLen  int
}

// NewDeviceSecretService returns a service preconfigured with sensible
// defaults (argon2id, t=2, m=64MB, p=1, 32-byte output). These values are
// fine for server-side secret verification on a signaling box; bump if
// hardware allows.
func NewDeviceSecretService() *DeviceSecretService {
	return &DeviceSecretService{
		time:      2,
		memoryKB:  64 * 1024,
		threads:   1,
		keyLen:    32,
		saltLen:   16,
		secretLen: 48, // 48 bytes hex ≈ 96 chars, plenty of entropy
	}
}

// Generate returns a new random plaintext device_secret as hex.
func (s *DeviceSecretService) Generate() (string, error) {
	buf := make([]byte, s.secretLen)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// Hash returns the canonical storage string for the given plaintext.
func (s *DeviceSecretService) Hash(secret string) (string, error) {
	salt := make([]byte, s.saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("gen salt: %w", err)
	}
	digest := argon2.IDKey([]byte(secret), salt, s.time, s.memoryKB, s.threads, s.keyLen)

	return fmt.Sprintf(
		"argon2id$v=%d$t=%d$m=%d$p=%d$%s$%s",
		argon2.Version,
		s.time, s.memoryKB, s.threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(digest),
	), nil
}

// Verify checks a plaintext secret against a stored hash. Uses
// constant-time comparison.
func (s *DeviceSecretService) Verify(secret, stored string) (bool, error) {
	parts := strings.Split(stored, "$")
	if len(parts) != 7 || parts[0] != "argon2id" {
		return false, errors.New("unknown hash format")
	}
	if !strings.HasPrefix(parts[1], "v=") {
		return false, errors.New("hash missing version")
	}
	tVal, err := parseParam(parts[2], "t=")
	if err != nil {
		return false, err
	}
	mVal, err := parseParam(parts[3], "m=")
	if err != nil {
		return false, err
	}
	pVal, err := parseParam(parts[4], "p=")
	if err != nil {
		return false, err
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, fmt.Errorf("decode salt: %w", err)
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[6])
	if err != nil {
		return false, fmt.Errorf("decode hash: %w", err)
	}

	got := argon2.IDKey([]byte(secret), salt,
		uint32(tVal), uint32(mVal), uint8(pVal), uint32(len(want)))

	return subtle.ConstantTimeCompare(got, want) == 1, nil
}

func parseParam(part, prefix string) (uint64, error) {
	if !strings.HasPrefix(part, prefix) {
		return 0, fmt.Errorf("expected %q, got %q", prefix, part)
	}
	return strconv.ParseUint(strings.TrimPrefix(part, prefix), 10, 32)
}
