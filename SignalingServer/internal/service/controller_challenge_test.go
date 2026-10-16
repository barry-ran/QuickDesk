package service

import (
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
	"testing"
	"time"
)

func testControllerKey(t *testing.T) (string, ed25519.PrivateKey) {
	t.Helper()
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	return base64.StdEncoding.EncodeToString(publicKey), privateKey
}

// The happy path: the server mints a challenge for one (device, grant, client)
// and the controller proves it holds the private half.
func TestControllerChallengeRoundTrip(t *testing.T) {
	secret := "challenge-secret"
	publicKey, privateKey := testControllerKey(t)

	challenge, expiresAt, err := SignControllerChallenge(secret, "123456789", 42, "guest-1", 0)
	if err != nil {
		t.Fatalf("sign challenge: %v", err)
	}
	if !expiresAt.After(time.Now().UTC()) {
		t.Fatalf("expiry %s is not in the future", expiresAt)
	}
	parsed, err := VerifyControllerChallenge(secret, challenge)
	if err != nil {
		t.Fatalf("verify challenge: %v", err)
	}
	if parsed.DeviceID != "123456789" || parsed.GrantID != 42 || parsed.ClientID != "guest-1" {
		t.Fatalf("challenge parsed as %+v", parsed)
	}

	signature := base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, []byte(challenge)))
	if err := VerifyControllerProof(secret, publicKey, challenge, signature, "123456789", 42, "guest-1"); err != nil {
		t.Fatalf("valid proof refused: %v", err)
	}
}

// A client id is opaque and may contain the separators our payload uses.
func TestControllerChallengeHandlesAwkwardClientIDs(t *testing.T) {
	secret := "challenge-secret"
	clientID := "a&b=c/1+2 3"
	publicKey, privateKey := testControllerKey(t)

	challenge, _, err := SignControllerChallenge(secret, "123456789", 7, clientID, 0)
	if err != nil {
		t.Fatalf("sign challenge: %v", err)
	}
	parsed, err := VerifyControllerChallenge(secret, challenge)
	if err != nil {
		t.Fatalf("verify challenge: %v", err)
	}
	if parsed.ClientID != clientID {
		t.Fatalf("client id round-tripped as %q, want %q", parsed.ClientID, clientID)
	}
	signature := base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, []byte(challenge)))
	if err := VerifyControllerProof(secret, publicKey, challenge, signature, "123456789", 7, clientID); err != nil {
		t.Fatalf("valid proof refused: %v", err)
	}
}

func TestControllerChallengeRejectsBadTokens(t *testing.T) {
	secret := "challenge-secret"
	challenge, _, err := SignControllerChallenge(secret, "123456789", 42, "guest-1", 0)
	if err != nil {
		t.Fatalf("sign challenge: %v", err)
	}

	cases := []struct {
		name  string
		token string
	}{
		{"empty", ""},
		{"no separator", "not-a-token"},
		{"wrong secret", func() string {
			other, _, _ := SignControllerChallenge("other-secret", "123456789", 42, "guest-1", 0)
			return other
		}()},
		{"tampered payload", "x" + challenge},
		{"body without signature", strings.SplitN(challenge, ".", 2)[0] + ".zzz"},
	}
	for _, tc := range cases {
		if _, err := VerifyControllerChallenge(secret, tc.token); !errors.Is(err, ErrInvalidControllerChallenge) {
			t.Fatalf("%s: err = %v, want ErrInvalidControllerChallenge", tc.name, err)
		}
	}
}

// An expired challenge is refused even though its signature is intact.
func TestControllerChallengeExpires(t *testing.T) {
	secret := "challenge-secret"
	// Minted directly rather than through SignControllerChallenge: waiting out a
	// real TTL would slow the suite down for no extra coverage.
	expired := time.Now().Add(-time.Minute).UTC()
	payload := controllerChallengePayload("123456789", 42, "guest-1", "nonce", expired.Unix())
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	token := base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." +
		base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	if _, err := VerifyControllerChallenge(secret, token); !errors.Is(err, ErrInvalidControllerChallenge) {
		t.Fatalf("expired challenge err = %v, want ErrInvalidControllerChallenge", err)
	}
}

// The binding is what stops a signature from being replayed somewhere else: a
// token minted for another device, grant or client is refused even when the
// signature itself is correct.
func TestControllerProofIsBoundToTheChallenge(t *testing.T) {
	secret := "challenge-secret"
	publicKey, privateKey := testControllerKey(t)
	challenge, _, err := SignControllerChallenge(secret, "123456789", 42, "guest-1", 0)
	if err != nil {
		t.Fatalf("sign challenge: %v", err)
	}
	signature := base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, []byte(challenge)))

	cases := []struct {
		name      string
		deviceID  string
		grantID   uint
		clientID  string
		signature string
		want      error
	}{
		{"another device", "987654321", 42, "guest-1", signature, ErrInvalidControllerChallenge},
		{"another grant", "123456789", 43, "guest-1", signature, ErrInvalidControllerChallenge},
		{"another client", "123456789", 42, "guest-2", signature, ErrInvalidControllerChallenge},
		{"wrong signature", "123456789", 42, "guest-1",
			base64.StdEncoding.EncodeToString(make([]byte, ed25519.SignatureSize)), ErrControllerSignatureInvalid},
	}
	for _, tc := range cases {
		err := VerifyControllerProof(secret, publicKey, challenge, tc.signature, tc.deviceID, tc.grantID, tc.clientID)
		if !errors.Is(err, tc.want) {
			t.Fatalf("%s: err = %v, want %v", tc.name, err, tc.want)
		}
	}
}

func TestControllerProofRequiresChallengeAndSignature(t *testing.T) {
	publicKey, _ := testControllerKey(t)
	err := VerifyControllerProof("secret", publicKey, "", "", "123456789", 42, "guest-1")
	if !errors.Is(err, ErrControllerSignatureInvalid) {
		t.Fatalf("missing proof err = %v, want ErrControllerSignatureInvalid", err)
	}
	// A grant with no bound key has nothing to prove.
	if err := VerifyControllerProof("secret", "", "", "", "123456789", 42, "guest-1"); err != nil {
		t.Fatalf("unbound grant should not require a proof, got %v", err)
	}
}

func TestParseControllerPublicKeyValidatesShape(t *testing.T) {
	publicKey, _ := testControllerKey(t)
	if _, err := ParseControllerPublicKey(publicKey); err != nil {
		t.Fatalf("valid key refused: %v", err)
	}
	for _, bad := range []string{"", "not base64!", base64.StdEncoding.EncodeToString([]byte("too short"))} {
		if _, err := ParseControllerPublicKey(bad); !errors.Is(err, ErrInvalidControllerPublicKey) {
			t.Fatalf("ParseControllerPublicKey(%q) err = %v, want ErrInvalidControllerPublicKey", bad, err)
		}
	}
}
