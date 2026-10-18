package service

import (
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strconv"
	"strings"
	"time"
)

// DefaultControllerChallengeTTL bounds how long a proof-of-possession
// challenge stays usable. It only has to outlive one round trip - mint the
// challenge, sign it, present the grant - so it is deliberately short.
const DefaultControllerChallengeTTL = 2 * time.Minute

var (
	// ErrInvalidControllerChallenge covers every way a challenge token can be
	// unusable: malformed, wrong signature, or expired.
	ErrInvalidControllerChallenge = errors.New("invalid or expired controller challenge")
	// ErrInvalidControllerPublicKey is returned when a bound key is not base64
	// of exactly one 32-byte Ed25519 public key.
	ErrInvalidControllerPublicKey = errors.New("bound_client_public_key must be base64 of a 32-byte Ed25519 public key")
	// ErrControllerSignatureInvalid is returned when the signature does not
	// verify against the bound key over the challenge.
	ErrControllerSignatureInvalid = errors.New("controller signature does not match the bound public key")
)

// ControllerChallenge is the short-lived, server-signed statement a controller
// must sign to redeem a grant that is bound to a public key.
//
// The token carries the binding rather than storing server state: a challenge
// is only useful together with a signature by the private key the owner bound
// to the grant, so there is nothing to exhaust and nothing to clean up.
type ControllerChallenge struct {
	DeviceID  string
	GrantID   uint
	ClientID  string
	ExpiresAt time.Time
}

func controllerChallengePayload(deviceID string, grantID uint, clientID string, nonce string, expUnix int64) string {
	// The client id is base64url-encoded rather than escaped: it is an opaque
	// string chosen by the client and may contain '&' or '='.
	return "did=" + deviceID +
		"&gid=" + formatUintID(grantID) +
		"&cid=" + base64.RawURLEncoding.EncodeToString([]byte(clientID)) +
		"&nonce=" + nonce +
		"&exp=" + strconv.FormatInt(expUnix, 10)
}

// SignControllerChallenge mints a challenge bound to one device, grant and
// client id. ttl <= 0 means DefaultControllerChallengeTTL.
func SignControllerChallenge(secret, deviceID string, grantID uint, clientID string, ttl time.Duration) (string, time.Time, error) {
	if strings.TrimSpace(secret) == "" {
		return "", time.Time{}, errors.New("controller challenge secret is not configured")
	}
	if ttl <= 0 {
		ttl = DefaultControllerChallengeTTL
	}
	nonceRaw := make([]byte, 16)
	if _, err := rand.Read(nonceRaw); err != nil {
		return "", time.Time{}, err
	}
	exp := time.Now().Add(ttl).UTC().Truncate(time.Second)
	payload := controllerChallengePayload(deviceID, grantID, clientID,
		base64.RawURLEncoding.EncodeToString(nonceRaw), exp.Unix())
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + sig, exp, nil
}

// VerifyControllerChallenge checks the signature and the expiry, and returns
// what the challenge was minted for.
func VerifyControllerChallenge(secret, token string) (*ControllerChallenge, error) {
	body, sig, ok := strings.Cut(strings.TrimSpace(token), ".")
	if !ok || body == "" || sig == "" {
		return nil, ErrInvalidControllerChallenge
	}
	raw, err := base64.RawURLEncoding.DecodeString(body)
	if err != nil {
		return nil, ErrInvalidControllerChallenge
	}
	payload := string(raw)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(sig)) {
		return nil, ErrInvalidControllerChallenge
	}

	values := map[string]string{}
	for _, kv := range strings.Split(payload, "&") {
		k, v, ok := strings.Cut(kv, "=")
		if !ok {
			return nil, ErrInvalidControllerChallenge
		}
		values[k] = v
	}
	if values["did"] == "" || values["nonce"] == "" {
		return nil, ErrInvalidControllerChallenge
	}
	grantID, err := strconv.ParseUint(values["gid"], 10, 64)
	if err != nil || grantID == 0 {
		return nil, ErrInvalidControllerChallenge
	}
	clientID, err := base64.RawURLEncoding.DecodeString(values["cid"])
	if err != nil {
		return nil, ErrInvalidControllerChallenge
	}
	expUnix, err := strconv.ParseInt(values["exp"], 10, 64)
	if err != nil {
		return nil, ErrInvalidControllerChallenge
	}
	expiresAt := time.Unix(expUnix, 0).UTC()
	if !time.Now().UTC().Before(expiresAt) {
		return nil, ErrInvalidControllerChallenge
	}
	return &ControllerChallenge{
		DeviceID:  values["did"],
		GrantID:   uint(grantID),
		ClientID:  string(clientID),
		ExpiresAt: expiresAt,
	}, nil
}

// ParseControllerPublicKey accepts the base64 (standard alphabet) encoding of a
// raw 32-byte Ed25519 public key, which is what every crypto library hands out
// for that curve.
func ParseControllerPublicKey(encoded string) (ed25519.PublicKey, error) {
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encoded))
	if err != nil || len(raw) != ed25519.PublicKeySize {
		return nil, ErrInvalidControllerPublicKey
	}
	return ed25519.PublicKey(raw), nil
}

// VerifyControllerSignature checks that signature (base64, standard alphabet)
// is the bound key's signature over the challenge string exactly as the
// server returned it. Signing the token rather than the decoded payload keeps
// the client side to "sign these bytes" with no parsing of our envelope.
func VerifyControllerSignature(publicKeyB64, challenge, signatureB64 string) error {
	publicKey, err := ParseControllerPublicKey(publicKeyB64)
	if err != nil {
		return err
	}
	signature, err := base64.StdEncoding.DecodeString(strings.TrimSpace(signatureB64))
	if err != nil || len(signature) != ed25519.SignatureSize {
		return ErrControllerSignatureInvalid
	}
	if !ed25519.Verify(publicKey, []byte(challenge), signature) {
		return ErrControllerSignatureInvalid
	}
	return nil
}

// VerifyControllerProof runs the whole check for one redemption: the challenge
// must have been minted for this device, this grant and this client id, must
// still be valid, and must carry this controller's signature.
func VerifyControllerProof(secret, publicKeyB64, challenge, signature, deviceID string, grantID uint, clientID string) error {
	if publicKeyB64 == "" {
		return nil
	}
	if strings.TrimSpace(challenge) == "" || strings.TrimSpace(signature) == "" {
		return ErrControllerSignatureInvalid
	}
	parsed, err := VerifyControllerChallenge(secret, challenge)
	if err != nil {
		return err
	}
	// A challenge minted for another device, grant or client must not admit
	// this one: the binding is what keeps a signature from being replayed
	// against a different code.
	if parsed.DeviceID != deviceID || parsed.GrantID != grantID || parsed.ClientID != clientID {
		return ErrInvalidControllerChallenge
	}
	return VerifyControllerSignature(publicKeyB64, challenge, signature)
}
