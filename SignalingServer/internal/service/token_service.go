package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// Token TTLs (see docs §2.1).
const (
	UserAccessTokenTTL  = 2 * time.Hour
	UserRefreshTokenTTL = 30 * 24 * time.Hour

	AdminAccessTokenTTL  = 1 * time.Hour
	AdminRefreshTokenTTL = 7 * 24 * time.Hour

	HostSignalTokenTTL   = 5 * time.Minute  // POST /v1/devices/:id/signal-tokens
	ClientSignalTokenTTL = 60 * time.Second // POST /v1/devices/:id/access-code:verify
)

// TokenScope namespaces tokens so a user access_token can never be parsed as
// an admin token (or vice versa). See §2.16 J5.
type TokenScope string

const (
	ScopeUser  TokenScope = "user"
	ScopeAdmin TokenScope = "admin"
)

// SignalRole tags a signal_token with which side it's for.
type SignalRole string

const (
	SignalRoleHost   SignalRole = "host"
	SignalRoleClient SignalRole = "client"
)

// Sentinel errors so callers can map to RFC7807 codes.
var (
	ErrTokenNotFound      = errors.New("token not found")
	ErrRefreshInvalid     = errors.New("refresh token invalid or rotated")
	ErrRefreshFamilyBreak = errors.New("refresh token reuse detected; family revoked")
)

// FamilyBreakInfo is attached to ErrRefreshFamilyBreak so handlers can
// publish session.revoked events to the affected subject without needing
// extra plumbing. Retrieve it via errors.As(err, *FamilyBreakInfo).
type FamilyBreakInfo struct {
	Scope     TokenScope
	FamilyID  string
	SubjectID uint
}

func (f *FamilyBreakInfo) Error() string {
	return "refresh token reuse detected; family revoked"
}

func (f *FamilyBreakInfo) Unwrap() error { return ErrRefreshFamilyBreak }

// TokenService manages access/refresh tokens (user + admin) and one-shot
// signal tokens. Everything lives in Redis; we don't touch the DB.
type TokenService struct {
	rdb *redis.Client
}

func NewTokenService(rdb *redis.Client) *TokenService {
	return &TokenService{rdb: rdb}
}

// =====================================================================
// Key helpers
// =====================================================================

func (t *TokenService) accessKey(scope TokenScope, token string) string {
	return fmt.Sprintf("qd:session:%s:access:%s", scope, token)
}

func (t *TokenService) refreshKey(scope TokenScope, token string) string {
	return fmt.Sprintf("qd:session:%s:refresh:%s", scope, token)
}

func (t *TokenService) familyKey(scope TokenScope, family string) string {
	return fmt.Sprintf("qd:session:%s:family:%s", scope, family)
}

// userFamiliesKey indexes every family belonging to subject (user/admin) so
// we can enumerate / revoke all sessions of a user without scanning. This
// is the *only* per-subject reverse index we keep; it's intentionally
// simple (one Redis Set, bounded by the subject's device count).
func (t *TokenService) userFamiliesKey(scope TokenScope, subjectID uint) string {
	return fmt.Sprintf("qd:session:%s:user_families:%d", scope, subjectID)
}

func (t *TokenService) signalKey(token string) string {
	return fmt.Sprintf("qd:signal_token:%s", token)
}

// spentKey holds family_id for a refresh token that has already been
// rotated, so a second attempt to use the same token (token reuse,
// §2.16) can find the family and revoke it.
func (t *TokenService) spentKey(scope TokenScope, token string) string {
	return fmt.Sprintf("qd:session:%s:spent:%s", scope, token)
}

// sessionMetaKey stores per-family human-visible metadata (user-agent /
// IP / last_seen) so GET /v1/me/sessions can describe each active session.
func (t *TokenService) sessionMetaKey(scope TokenScope, familyID string) string {
	return fmt.Sprintf("qd:session:%s:meta:%s", scope, familyID)
}

// =====================================================================
// Access + refresh issuance
// =====================================================================

// SessionMetadata is the captured-at-login context for a session family.
// None of these are security-sensitive; they exist so users can tell
// their own sessions apart on the "manage devices" page.
type SessionMetadata struct {
	UserAgent string    `json:"user_agent"`
	IP        string    `json:"ip"`
	LastSeen  time.Time `json:"last_seen"`
	CreatedAt time.Time `json:"created_at"`
}

// SessionTokens is what we hand back to clients on login / refresh.
type SessionTokens struct {
	AccessToken      string
	AccessExpiresAt  time.Time
	RefreshToken     string
	RefreshExpiresAt time.Time
	FamilyID         string
}

// refreshPayload is the JSON we store in the refresh token's Redis value.
type refreshPayload struct {
	SubjectID  uint   `json:"subject_id"` // user_id or admin_id
	FamilyID   string `json:"family_id"`
	Generation int    `json:"generation"`
	Scope      string `json:"scope"`
}

// accessPayload is what we put under the access_token key. We used to
// store just the subject_id as a plain string; we now store a tiny JSON
// envelope so the access token also knows *which family* it belongs to
// (required for DELETE /v1/me/sessions/current to kill the full session).
type accessPayload struct {
	SubjectID uint   `json:"subject_id"`
	FamilyID  string `json:"family_id"`
	Scope     string `json:"scope"`
}

// IssueSession creates a new (access_token, refresh_token) pair for
// subjectID under scope. The refresh is wrapped in a fresh family so that
// only this device's tokens get revoked if reuse is detected. `meta` is
// remembered on the family so the /v1/me/sessions listing can surface
// it; pass a zero value when you don't care (e.g. admin internal flows).
func (t *TokenService) IssueSession(ctx context.Context, scope TokenScope, subjectID uint, meta SessionMetadata) (SessionTokens, error) {
	access, err := randToken()
	if err != nil {
		return SessionTokens{}, err
	}
	refresh, err := randToken()
	if err != nil {
		return SessionTokens{}, err
	}
	family := uuid.NewString()

	now := time.Now().UTC()
	accessTTL, refreshTTL := scopeTTLs(scope)

	ap := accessPayload{SubjectID: subjectID, FamilyID: family, Scope: string(scope)}
	apBody, _ := json.Marshal(ap)
	if err := t.rdb.Set(ctx, t.accessKey(scope, access), string(apBody), accessTTL).Err(); err != nil {
		return SessionTokens{}, fmt.Errorf("store access token: %w", err)
	}

	payload := refreshPayload{SubjectID: subjectID, FamilyID: family, Generation: 0, Scope: string(scope)}
	body, _ := json.Marshal(payload)
	if err := t.rdb.Set(ctx, t.refreshKey(scope, refresh), string(body), refreshTTL).Err(); err != nil {
		return SessionTokens{}, fmt.Errorf("store refresh token: %w", err)
	}
	if err := t.rdb.SAdd(ctx, t.familyKey(scope, family), refresh).Err(); err != nil {
		return SessionTokens{}, fmt.Errorf("track family: %w", err)
	}
	t.rdb.Expire(ctx, t.familyKey(scope, family), refreshTTL)

	// Reverse index: subject_id -> {family_id, ...}.
	t.rdb.SAdd(ctx, t.userFamiliesKey(scope, subjectID), family)
	t.rdb.Expire(ctx, t.userFamiliesKey(scope, subjectID), refreshTTL)

	// Human-visible session metadata (never sensitive).
	if meta.CreatedAt.IsZero() {
		meta.CreatedAt = now
	}
	if meta.LastSeen.IsZero() {
		meta.LastSeen = now
	}
	metaBody, _ := json.Marshal(meta)
	t.rdb.Set(ctx, t.sessionMetaKey(scope, family), string(metaBody), refreshTTL)

	return SessionTokens{
		AccessToken:      access,
		AccessExpiresAt:  now.Add(accessTTL),
		RefreshToken:     refresh,
		RefreshExpiresAt: now.Add(refreshTTL),
		FamilyID:         family,
	}, nil
}

// VerifyAccessToken returns the subject ID stored under an access_token, or
// ErrTokenNotFound if the token is missing/expired.
func (t *TokenService) VerifyAccessToken(ctx context.Context, scope TokenScope, token string) (uint, error) {
	_, id, err := t.LookupAccessToken(ctx, scope, token)
	return id, err
}

// LookupAccessToken returns both the family_id and subject_id stored under
// an access_token. Bonus `family_id` lets callers like
// DELETE /v1/me/sessions/current revoke the whole session atomically.
func (t *TokenService) LookupAccessToken(ctx context.Context, scope TokenScope, token string) (familyID string, subjectID uint, err error) {
	val, rerr := t.rdb.Get(ctx, t.accessKey(scope, token)).Result()
	if rerr != nil {
		if rerr == redis.Nil {
			return "", 0, ErrTokenNotFound
		}
		return "", 0, rerr
	}
	// Backwards-compatibility: older tokens stored just the subject_id as
	// a bare decimal string. Try JSON first, fall back to Sscanf.
	var ap accessPayload
	if jerr := json.Unmarshal([]byte(val), &ap); jerr == nil && ap.SubjectID != 0 {
		return ap.FamilyID, ap.SubjectID, nil
	}
	var id uint
	fmt.Sscanf(val, "%d", &id)
	if id == 0 {
		return "", 0, ErrTokenNotFound
	}
	return "", id, nil
}

// TouchSession refreshes the LastSeen timestamp on the family metadata.
// Called by UserAuth.Required() on every authenticated request so the
// sessions listing surface stays live without extra storage writes on the
// hot path.
func (t *TokenService) TouchSession(ctx context.Context, scope TokenScope, familyID string) {
	if familyID == "" {
		return
	}
	key := t.sessionMetaKey(scope, familyID)
	val, err := t.rdb.Get(ctx, key).Result()
	if err != nil {
		return
	}
	var m SessionMetadata
	if json.Unmarshal([]byte(val), &m) != nil {
		return
	}
	m.LastSeen = time.Now().UTC()
	body, _ := json.Marshal(m)
	// Preserve the original TTL on the key.
	ttl, _ := t.rdb.TTL(ctx, key).Result()
	if ttl <= 0 {
		_, refreshTTL := scopeTTLs(scope)
		ttl = refreshTTL
	}
	t.rdb.Set(ctx, key, string(body), ttl)
}

// ListSessionsWithMeta returns every active family for subjectID together
// with its captured metadata (user-agent / IP / last_seen / created_at).
type SessionView struct {
	FamilyID string          `json:"id"`
	Meta     SessionMetadata `json:"meta"`
}

func (t *TokenService) ListSessionsWithMeta(ctx context.Context, scope TokenScope, subjectID uint) ([]SessionView, error) {
	fams, err := t.rdb.SMembers(ctx, t.userFamiliesKey(scope, subjectID)).Result()
	if err != nil {
		return nil, err
	}
	out := make([]SessionView, 0, len(fams))
	for _, fam := range fams {
		var view = SessionView{FamilyID: fam}
		val, err := t.rdb.Get(ctx, t.sessionMetaKey(scope, fam)).Result()
		if err == nil {
			_ = json.Unmarshal([]byte(val), &view.Meta)
		}
		out = append(out, view)
	}
	return out, nil
}

// RevokeAccessToken removes a single access token (used for explicit
// logout). The refresh family is revoked separately via RevokeFamily.
func (t *TokenService) RevokeAccessToken(ctx context.Context, scope TokenScope, token string) error {
	return t.rdb.Del(ctx, t.accessKey(scope, token)).Err()
}

// RotateRefreshToken consumes the old refresh, issues a new (access,
// refresh) pair within the same family, and bumps the generation. If the
// caller presents a refresh that's already been rotated (token reuse), we
// nuke the entire family — all of that user's sessions on this device are
// invalidated. See §2.16 family logic.
func (t *TokenService) RotateRefreshToken(ctx context.Context, scope TokenScope, oldRefresh string) (SessionTokens, error) {
	// GETDEL is atomic: only one of two concurrent refreshes wins.
	body, err := t.rdb.GetDel(ctx, t.refreshKey(scope, oldRefresh)).Result()
	if err != nil {
		if err == redis.Nil {
			// Two possibilities:
			//   1) Token never existed (typo / very old client) → just 401.
			//   2) Token was already rotated → reuse → revoke the whole
			//      family we remembered under the spent shadow key.
			if fam, ferr := t.rdb.Get(ctx, t.spentKey(scope, oldRefresh)).Result(); ferr == nil && fam != "" {
				subject := t.subjectForFamily(ctx, scope, fam)
				t.RevokeFamily(ctx, scope, fam)
				return SessionTokens{}, &FamilyBreakInfo{Scope: scope, FamilyID: fam, SubjectID: subject}
			}
			return SessionTokens{}, ErrRefreshInvalid
		}
		return SessionTokens{}, err
	}

	var payload refreshPayload
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		return SessionTokens{}, ErrRefreshInvalid
	}
	if TokenScope(payload.Scope) != scope {
		// Cross-scope token reuse — definitely tampered, kill family.
		t.RevokeFamily(ctx, scope, payload.FamilyID)
		return SessionTokens{}, &FamilyBreakInfo{Scope: scope, FamilyID: payload.FamilyID, SubjectID: payload.SubjectID}
	}

	// Remove the old token from the family set and record it in the
	// "spent" shadow so any subsequent use is detected as reuse.
	t.rdb.SRem(ctx, t.familyKey(scope, payload.FamilyID), oldRefresh)
	_, refreshTTL := scopeTTLs(scope)
	t.rdb.Set(ctx, t.spentKey(scope, oldRefresh), payload.FamilyID, refreshTTL)

	access, err := randToken()
	if err != nil {
		return SessionTokens{}, err
	}
	newRefresh, err := randToken()
	if err != nil {
		return SessionTokens{}, err
	}
	now := time.Now().UTC()
	accessTTL, refreshTTL := scopeTTLs(scope)

	// Re-use the access payload envelope so the new access token also
	// knows its family (needed for DELETE /v1/me/sessions/current).
	newAP := accessPayload{SubjectID: payload.SubjectID, FamilyID: payload.FamilyID, Scope: string(scope)}
	newAPBody, _ := json.Marshal(newAP)
	if err := t.rdb.Set(ctx, t.accessKey(scope, access), string(newAPBody), accessTTL).Err(); err != nil {
		return SessionTokens{}, err
	}
	newPayload := refreshPayload{
		SubjectID:  payload.SubjectID,
		FamilyID:   payload.FamilyID,
		Generation: payload.Generation + 1,
		Scope:      string(scope),
	}
	nb, _ := json.Marshal(newPayload)
	if err := t.rdb.Set(ctx, t.refreshKey(scope, newRefresh), string(nb), refreshTTL).Err(); err != nil {
		return SessionTokens{}, err
	}
	t.rdb.SAdd(ctx, t.familyKey(scope, payload.FamilyID), newRefresh)
	t.rdb.Expire(ctx, t.familyKey(scope, payload.FamilyID), refreshTTL)

	return SessionTokens{
		AccessToken:      access,
		AccessExpiresAt:  now.Add(accessTTL),
		RefreshToken:     newRefresh,
		RefreshExpiresAt: now.Add(refreshTTL),
		FamilyID:         payload.FamilyID,
	}, nil
}

// RevokeFamily kills every refresh token in the family and the family set
// itself. Access tokens issued from this family will expire on their own;
// callers that care can publish a `session.revoked` event so connected
// clients drop their access_token immediately.
func (t *TokenService) RevokeFamily(ctx context.Context, scope TokenScope, familyID string) {
	if familyID == "" {
		return
	}
	key := t.familyKey(scope, familyID)
	members, _ := t.rdb.SMembers(ctx, key).Result()
	pipe := t.rdb.Pipeline()
	for _, m := range members {
		pipe.Del(ctx, t.refreshKey(scope, m))
	}
	pipe.Del(ctx, key)
	pipe.Del(ctx, t.sessionMetaKey(scope, familyID))
	_, _ = pipe.Exec(ctx)
}

// RevokeAllForSubject nukes every active family belonging to subjectID in
// the given scope. Use for admin-initiated "force logout" and DELETE
// /v1/me/sessions/:session_id (where session_id is a family_id).
func (t *TokenService) RevokeAllForSubject(ctx context.Context, scope TokenScope, subjectID uint) {
	idx := t.userFamiliesKey(scope, subjectID)
	families, _ := t.rdb.SMembers(ctx, idx).Result()
	for _, fam := range families {
		t.RevokeFamily(ctx, scope, fam)
	}
	t.rdb.Del(ctx, idx)
}

// ListFamilies returns all active family IDs for a subject. Callers pair
// each family with a best-effort "what is this" description (it's opaque
// to us — just a UUID).
func (t *TokenService) ListFamilies(ctx context.Context, scope TokenScope, subjectID uint) ([]string, error) {
	return t.rdb.SMembers(ctx, t.userFamiliesKey(scope, subjectID)).Result()
}

// RevokeFamilyForSubject removes a single family from a subject's reverse
// index and then revokes it.
func (t *TokenService) RevokeFamilyForSubject(ctx context.Context, scope TokenScope, subjectID uint, familyID string) error {
	if familyID == "" {
		return errors.New("family_id required")
	}
	// Confirm the family belongs to this subject before revoking — avoids
	// letting one user stomp another's session with a guessed UUID.
	ok, err := t.rdb.SIsMember(ctx, t.userFamiliesKey(scope, subjectID), familyID).Result()
	if err != nil {
		return err
	}
	if !ok {
		return ErrTokenNotFound
	}
	t.rdb.SRem(ctx, t.userFamiliesKey(scope, subjectID), familyID)
	t.RevokeFamily(ctx, scope, familyID)
	return nil
}

// subjectForFamily looks up the subject_id stored on any refresh token
// that still belongs to the given family. Used by RotateRefreshToken to
// identify who owns a reuse-detected family so the caller can publish
// session.revoked.
func (t *TokenService) subjectForFamily(ctx context.Context, scope TokenScope, familyID string) uint {
	members, err := t.rdb.SMembers(ctx, t.familyKey(scope, familyID)).Result()
	if err != nil || len(members) == 0 {
		return 0
	}
	for _, m := range members {
		raw, err := t.rdb.Get(ctx, t.refreshKey(scope, m)).Result()
		if err != nil {
			continue
		}
		var p refreshPayload
		if json.Unmarshal([]byte(raw), &p) == nil {
			return p.SubjectID
		}
	}
	return 0
}

// =====================================================================
// Signal tokens (one-shot, used at WS first-frame auth)
// =====================================================================

type SignalTokenPayload struct {
	DeviceID string     `json:"device_id"`
	Role     SignalRole `json:"role"`
	ClientID string     `json:"client_id,omitempty"`
}

// IssueSignalToken creates a single-use signal token. role determines TTL.
func (t *TokenService) IssueSignalToken(ctx context.Context, payload SignalTokenPayload) (string, time.Time, error) {
	tok, err := randToken()
	if err != nil {
		return "", time.Time{}, err
	}
	ttl := ClientSignalTokenTTL
	if payload.Role == SignalRoleHost {
		ttl = HostSignalTokenTTL
	}
	body, _ := json.Marshal(payload)
	if err := t.rdb.Set(ctx, t.signalKey(tok), string(body), ttl).Err(); err != nil {
		return "", time.Time{}, err
	}
	return tok, time.Now().UTC().Add(ttl), nil
}

// ConsumeSignalToken atomically reads and deletes a signal token. The
// returned payload is what was supplied at issuance; ErrTokenNotFound if
// the token was already consumed or expired.
func (t *TokenService) ConsumeSignalToken(ctx context.Context, token string) (SignalTokenPayload, error) {
	val, err := t.rdb.GetDel(ctx, t.signalKey(token)).Result()
	if err != nil {
		if err == redis.Nil {
			return SignalTokenPayload{}, ErrTokenNotFound
		}
		return SignalTokenPayload{}, err
	}
	var p SignalTokenPayload
	if err := json.Unmarshal([]byte(val), &p); err != nil {
		return SignalTokenPayload{}, ErrTokenNotFound
	}
	return p, nil
}

// ClientSignalSessionTTL is how long a client signal_token remains valid
// after first successful auth, allowing reconnection within this window.
const ClientSignalSessionTTL = 10 * time.Minute

// ValidateAndExtendSignalToken reads a signal token WITHOUT deleting it,
// and extends its TTL for client reconnection. Used for client-role tokens
// so the Chromium client can reconnect after a brief network disruption
// without needing a fresh token from Qt.
func (t *TokenService) ValidateAndExtendSignalToken(ctx context.Context, token string) (SignalTokenPayload, error) {
	key := t.signalKey(token)
	val, err := t.rdb.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return SignalTokenPayload{}, ErrTokenNotFound
		}
		return SignalTokenPayload{}, err
	}
	var p SignalTokenPayload
	if err := json.Unmarshal([]byte(val), &p); err != nil {
		return SignalTokenPayload{}, ErrTokenNotFound
	}
	// Extend TTL so the client can reconnect within the session window.
	t.rdb.Expire(ctx, key, ClientSignalSessionTTL)
	return p, nil
}

// =====================================================================
// internals
// =====================================================================

func randToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func scopeTTLs(scope TokenScope) (time.Duration, time.Duration) {
	if scope == ScopeAdmin {
		return AdminAccessTokenTTL, AdminRefreshTokenTTL
	}
	return UserAccessTokenTTL, UserRefreshTokenTTL
}
