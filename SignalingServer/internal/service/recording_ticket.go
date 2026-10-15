package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strconv"
	"strings"
	"time"
)

// ErrInvalidRecordingTicket covers every way a media ticket can be unusable:
// malformed, wrong signature, or expired. Callers should not distinguish them.
var ErrInvalidRecordingTicket = errors.New("invalid or expired recording ticket")

// RecordingTicket is the short-lived, signature-based credential used to fetch
// a media artifact with a plain browser navigation.
//
// A browser cannot attach an Authorization header to <video src>, window.open
// or an <a download>, so a Bearer-only route is unusable for playback. The
// ticket carries the (recording, user) binding and an expiry, and is signed with
// a server secret — no database round trip and no revocable server-side state.
type RecordingTicket struct {
	RecordingID uint
	UserID      uint
	// Kind scopes the ticket to one media type, so a link handed out for
	// playback cannot be replayed against the thumbnail route or vice versa.
	Kind      string
	ExpiresAt time.Time
}

// Accepted values for RecordingTicket.Kind.
const (
	TicketKindArtifact  = "artifact"
	TicketKindThumbnail = "thumbnail"
)

func recordingTicketPayload(recordingID, userID uint, kind string, expUnix int64) string {
	return "rid=" + formatUintID(recordingID) +
		"&uid=" + formatUintID(userID) +
		"&kind=" + kind +
		"&exp=" + strconv.FormatInt(expUnix, 10)
}

// SignRecordingTicket mints a ticket valid for ttl.
func SignRecordingTicket(secret string, recordingID, userID uint, kind string, ttl time.Duration) (string, time.Time) {
	// Truncate to whole seconds: the expiry travels as a unix timestamp, so
	// returning sub-second precision here would misrepresent the real deadline.
	exp := time.Now().Add(ttl).UTC().Truncate(time.Second)
	if kind == "" {
		kind = TicketKindArtifact
	}
	payload := recordingTicketPayload(recordingID, userID, kind, exp.Unix())
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	body := base64.RawURLEncoding.EncodeToString([]byte(payload))
	return body + "." + sig, exp
}

// VerifyRecordingTicket checks the signature and the expiry.
func VerifyRecordingTicket(secret, ticket string) (*RecordingTicket, error) {
	body, sig, ok := strings.Cut(strings.TrimSpace(ticket), ".")
	if !ok || body == "" || sig == "" {
		return nil, ErrInvalidRecordingTicket
	}
	raw, err := base64.RawURLEncoding.DecodeString(body)
	if err != nil {
		return nil, ErrInvalidRecordingTicket
	}
	payload := string(raw)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(sig)) {
		return nil, ErrInvalidRecordingTicket
	}

	values := map[string]string{}
	for _, kv := range strings.Split(payload, "&") {
		k, v, ok := strings.Cut(kv, "=")
		if !ok {
			return nil, ErrInvalidRecordingTicket
		}
		values[k] = v
	}
	recordingID, err := strconv.ParseUint(values["rid"], 10, 64)
	if err != nil || recordingID == 0 {
		return nil, ErrInvalidRecordingTicket
	}
	userID, err := strconv.ParseUint(values["uid"], 10, 64)
	if err != nil || userID == 0 {
		return nil, ErrInvalidRecordingTicket
	}
	expUnix, err := strconv.ParseInt(values["exp"], 10, 64)
	if err != nil {
		return nil, ErrInvalidRecordingTicket
	}
	exp := time.Unix(expUnix, 0).UTC()
	if time.Now().UTC().After(exp) {
		return nil, ErrInvalidRecordingTicket
	}
	kind := values["kind"]
	if kind == "" {
		kind = TicketKindArtifact
	}
	return &RecordingTicket{
		RecordingID: uint(recordingID),
		UserID:      uint(userID),
		Kind:        kind,
		ExpiresAt:   exp,
	}, nil
}
