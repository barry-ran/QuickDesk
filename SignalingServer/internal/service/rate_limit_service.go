package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// RateLimitService implements the per-scope INCR+EXPIRE counters used by
// POST /v1/devices/:id/access-code:verify (§2.10):
//
//   qd:ratelimit:verify:{device_id}:{ip}     ≤5 errors / 60s
//   qd:ratelimit:verify:{device_id}          ≤30 errors / 60s
//   qd:ratelimit:ip:{ip}                     ≤60 total / 60s
//
// Only *failed* verifications bump the first two counters; the per-IP
// counter bumps on every request so that brute-forcing many device IDs
// from one IP is also bounded.
type RateLimitService struct {
	rdb *redis.Client
}

func NewRateLimitService(rdb *redis.Client) *RateLimitService {
	return &RateLimitService{rdb: rdb}
}

// Tunables mirror §2.10 and stay here as constants so they're easy to
// tweak without touching the handler.
const (
	verifyWindow           = 60 * time.Second
	verifyMaxDeviceIPError = 5
	verifyMaxDeviceError   = 30
	verifyMaxIP            = 60
)

// VerifyDecision describes whether a verify request may proceed.
type VerifyDecision struct {
	Allowed bool
	// Code/detail suggested when !Allowed. Callers should translate this
	// into a problem response.
	Code   string
	Detail string
	// RetryAfter is the number of seconds until the offending window
	// resets. Zero when Allowed.
	RetryAfter int
}

const (
	RLCodeTooManyAttempts = "TOO_MANY_ATTEMPTS" // (device,ip) or per-device
	RLCodeRateLimited     = "RATE_LIMITED"      // per-IP generic
)

// CheckVerify is called *before* the actual access_code comparison. The
// per-IP counter is always bumped; the error counters are reserved for
// CheckVerifyFailure below.
func (s *RateLimitService) CheckVerify(ctx context.Context, deviceID, ip string) VerifyDecision {
	ipCount, ipTTL, err := s.incrWindow(ctx, fmt.Sprintf("qd:ratelimit:ip:%s", ip), verifyWindow)
	if err != nil {
		log.Printf("[RateLimit] per-ip incr failed: %v", err)
		return VerifyDecision{Allowed: true} // fail-open on Redis hiccups
	}
	if ipCount > verifyMaxIP {
		return VerifyDecision{Code: RLCodeRateLimited, Detail: "Too many requests from this IP", RetryAfter: int(ipTTL.Seconds())}
	}
	// Pre-check error counters (no bump here — bump happens only on
	// verification failure).
	if cnt, ttl := s.peek(ctx, fmt.Sprintf("qd:ratelimit:verify:%s:%s", deviceID, ip)); cnt >= verifyMaxDeviceIPError {
		return VerifyDecision{Code: RLCodeTooManyAttempts, Detail: "Too many failed attempts for this device from your IP", RetryAfter: int(ttl.Seconds())}
	}
	if cnt, ttl := s.peek(ctx, fmt.Sprintf("qd:ratelimit:verify:%s", deviceID)); cnt >= verifyMaxDeviceError {
		return VerifyDecision{Code: RLCodeTooManyAttempts, Detail: "Too many failed attempts for this device", RetryAfter: int(ttl.Seconds())}
	}
	return VerifyDecision{Allowed: true}
}

// CheckVerifyFailure bumps the two error counters after a failed access
// code comparison. Called from the verify handler in the wrong-code path.
func (s *RateLimitService) CheckVerifyFailure(ctx context.Context, deviceID, ip string) {
	_, _, _ = s.incrWindow(ctx, fmt.Sprintf("qd:ratelimit:verify:%s:%s", deviceID, ip), verifyWindow)
	_, _, _ = s.incrWindow(ctx, fmt.Sprintf("qd:ratelimit:verify:%s", deviceID), verifyWindow)
}

// incrWindow atomically INCR a counter and sets TTL only on first hit
// (when INCR returns 1), preventing the window from being extended on
// subsequent requests. Returns (value, ttl) after the increment.
func (s *RateLimitService) incrWindow(ctx context.Context, key string, ttl time.Duration) (int64, time.Duration, error) {
	val, err := s.rdb.Incr(ctx, key).Result()
	if err != nil {
		return 0, 0, err
	}
	if val == 1 {
		s.rdb.Expire(ctx, key, ttl)
	}
	remaining, _ := s.rdb.TTL(ctx, key).Result()
	if remaining <= 0 {
		remaining = ttl
	}
	return val, remaining, nil
}

func (s *RateLimitService) peek(ctx context.Context, key string) (int64, time.Duration) {
	v, _ := s.rdb.Get(ctx, key).Int64()
	t, _ := s.rdb.TTL(ctx, key).Result()
	if t <= 0 {
		t = verifyWindow
	}
	return v, t
}

// VerifyFailureResult is returned from RecordVerifyFailure so callers can
// emit TOO_MANY_ATTEMPTS with a Retry-After on the request that just
// tripped the limit.
type VerifyFailureResult struct {
	TripsLimit    bool
	RetryAfterSec int
}

// VerifyPreflightResult merges the per-IP and per-device pre-checks into
// a single struct so handler code reads naturally. `Kind` tells the
// caller which dimension tripped so it can pick the right HTTP status
// (§2.10 — per-IP → 429, per-device/per-(device,ip) → 403).
type VerifyPreflightResult struct {
	Blocked       bool
	Kind          VerifyBlockKind
	Reason        string
	RetryAfterSec int
}

// VerifyBlockKind identifies which limit dimension blocked a verify call.
type VerifyBlockKind int

const (
	VerifyBlockNone     VerifyBlockKind = iota
	VerifyBlockPerIP                    // general IP throttle → 429
	VerifyBlockPerDevice                // per-device or per-(device,ip) → 403
)

// CheckVerifyPreflight runs the pre-comparison checks used by
// POST /v1/devices/:id/access-code:verify. It always bumps the per-IP
// counter (which caps total requests regardless of success) and reads
// the two error counters without bumping them.
func (s *RateLimitService) CheckVerifyPreflight(ctx context.Context, deviceID, ip string) (VerifyPreflightResult, error) {
	d := s.CheckVerify(ctx, deviceID, ip)
	if d.Allowed {
		return VerifyPreflightResult{}, nil
	}
	kind := VerifyBlockPerDevice
	if d.Code == RLCodeRateLimited {
		kind = VerifyBlockPerIP
	}
	return VerifyPreflightResult{
		Blocked:       true,
		Kind:          kind,
		Reason:        d.Detail,
		RetryAfterSec: d.RetryAfter,
	}, nil
}

// RecordVerifyFailure bumps both device-scoped error counters after a
// failed code comparison and tells the caller whether this specific
// request pushed either counter over its limit.
func (s *RateLimitService) RecordVerifyFailure(ctx context.Context, deviceID, ip string) (VerifyFailureResult, error) {
	dIPVal, dIPTTL, err := s.incrWindow(ctx, fmt.Sprintf("qd:ratelimit:verify:%s:%s", deviceID, ip), verifyWindow)
	if err != nil {
		return VerifyFailureResult{}, err
	}
	dVal, dTTL, err := s.incrWindow(ctx, fmt.Sprintf("qd:ratelimit:verify:%s", deviceID), verifyWindow)
	if err != nil {
		return VerifyFailureResult{}, err
	}
	if dIPVal >= verifyMaxDeviceIPError {
		return VerifyFailureResult{TripsLimit: true, RetryAfterSec: int(dIPTTL.Seconds())}, nil
	}
	if dVal >= verifyMaxDeviceError {
		return VerifyFailureResult{TripsLimit: true, RetryAfterSec: int(dTTL.Seconds())}, nil
	}
	return VerifyFailureResult{}, nil
}

// ResetVerifyFailures clears the device-scoped error counters after a
// successful verification so the next honest user isn't held hostage by
// earlier bad attempts from another IP.
func (s *RateLimitService) ResetVerifyFailures(ctx context.Context, deviceID, ip string) {
	_ = s.rdb.Del(ctx,
		fmt.Sprintf("qd:ratelimit:verify:%s:%s", deviceID, ip),
		fmt.Sprintf("qd:ratelimit:verify:%s", deviceID),
	).Err()
}

// -----------------------------------------------------------------------
// Hot-path throttles for heartbeat / signal-tokens (§2.10 footnote).
// These are a minimal 1-second floor enforced via SETNX; they exist so a
// compromised host can't hammer the server with thousands of heartbeats
// per second.
// -----------------------------------------------------------------------

// HeartbeatThrottle returns (blocked, err). When blocked, the caller must
// return 429 with Retry-After: 1.
func (s *RateLimitService) HeartbeatThrottle(ctx context.Context, deviceID string) (bool, error) {
	return s.minInterval(ctx, fmt.Sprintf("qd:ratelimit:hb:%s", deviceID), time.Second)
}

// SignalTokenThrottle mirrors HeartbeatThrottle for
// POST /v1/devices/:id/signal-tokens.
func (s *RateLimitService) SignalTokenThrottle(ctx context.Context, deviceID string) (bool, error) {
	return s.minInterval(ctx, fmt.Sprintf("qd:ratelimit:sigtok:%s", deviceID), time.Second)
}

// minInterval implements "at most one hit per TTL" using SETNX.
func (s *RateLimitService) minInterval(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	ok, err := s.rdb.SetNX(ctx, key, "1", ttl).Result()
	if err != nil {
		return false, err
	}
	// SETNX returns false when the key already existed → blocked.
	return !ok, nil
}
