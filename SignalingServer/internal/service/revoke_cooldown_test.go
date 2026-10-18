package service

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newRevokeCooldown(t *testing.T, ttl time.Duration) (*RevokeCooldown, *miniredis.Miniredis) {
	t.Helper()
	redisServer := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return NewRevokeCooldown(client, ttl), redisServer
}

// The window has to be narrow in the right way: the source that was kicked is
// refused, and nobody else is. Keying it by device *and* address is what makes
// that true, and the TTL is what makes it a delay rather than a ban.
func TestRevokeCooldownRefusesOnlyTheKickedSource(t *testing.T) {
	cooldown, redisServer := newRevokeCooldown(t, 5*time.Minute)
	ctx := context.Background()
	if err := cooldown.Arm(ctx, "111111111", "203.0.113.7"); err != nil {
		t.Fatalf("arm: %v", err)
	}
	for _, tc := range []struct {
		name     string
		deviceID string
		ip       string
		want     bool
	}{
		{"the kicked source", "111111111", "203.0.113.7", true},
		{"another source on the same device", "111111111", "203.0.113.8", false},
		{"the same source on another device", "222222222", "203.0.113.7", false},
	} {
		got, err := cooldown.Armed(ctx, tc.deviceID, tc.ip)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if got != tc.want {
			t.Fatalf("%s armed = %t, want %t", tc.name, got, tc.want)
		}
	}
	// The window expires on its own, without anything having to clean it up.
	redisServer.FastForward(6 * time.Minute)
	armed, err := cooldown.Armed(ctx, "111111111", "203.0.113.7")
	if err != nil {
		t.Fatalf("armed after expiry: %v", err)
	}
	if armed {
		t.Fatal("the window is still armed after its TTL")
	}
}

// 0 keeps the historical behaviour - a revoked controller may reconnect at
// once - so turning the feature off is a real off switch, not a very short one.
func TestRevokeCooldownDisabledAtZero(t *testing.T) {
	cooldown, _ := newRevokeCooldown(t, 0)
	ctx := context.Background()
	if err := cooldown.Arm(ctx, "111111111", "203.0.113.7"); err != nil {
		t.Fatalf("arm with the feature off: %v", err)
	}
	armed, err := cooldown.Armed(ctx, "111111111", "203.0.113.7")
	if err != nil {
		t.Fatalf("armed: %v", err)
	}
	if armed {
		t.Fatal("a disabled cooldown refused a source")
	}
	if ttl := cooldown.TTL(); ttl != 0 {
		t.Fatalf("TTL = %v, want 0", ttl)
	}
	if remaining, _ := cooldown.Remaining(ctx, "111111111", "203.0.113.7"); remaining != 0 {
		t.Fatalf("Remaining = %v with the feature off, want 0", remaining)
	}
}

// A missing address must never arm anything: an empty key segment would be a
// wildcard that locks out every source on the device, which is the opposite of
// what a kick is for.
func TestRevokeCooldownIgnoresBlankKeys(t *testing.T) {
	cooldown, redisServer := newRevokeCooldown(t, 5*time.Minute)
	ctx := context.Background()
	for _, tc := range []struct{ deviceID, ip string }{
		{"", "203.0.113.7"},
		{"111111111", ""},
		{"   ", "   "},
	} {
		if err := cooldown.Arm(ctx, tc.deviceID, tc.ip); err != nil {
			t.Fatalf("arm(%q, %q): %v", tc.deviceID, tc.ip, err)
		}
	}
	if keys := redisServer.Keys(); len(keys) != 0 {
		t.Fatalf("blank keys armed %v, want nothing", keys)
	}
}

// Remaining is the Retry-After hint the refusal carries, so it has to be inside
// the window and fall to zero once the window is over.
func TestRevokeCooldownRemainingTracksTheWindowsEnd(t *testing.T) {
	cooldown, redisServer := newRevokeCooldown(t, 90*time.Second)
	ctx := context.Background()
	if err := cooldown.Arm(ctx, "111111111", "203.0.113.7"); err != nil {
		t.Fatalf("arm: %v", err)
	}
	remaining, err := cooldown.Remaining(ctx, "111111111", "203.0.113.7")
	if err != nil {
		t.Fatalf("remaining: %v", err)
	}
	if remaining <= 0 || remaining > 90*time.Second {
		t.Fatalf("remaining = %v, want a positive value inside the window", remaining)
	}
	redisServer.FastForward(2 * time.Minute)
	if remaining, _ = cooldown.Remaining(ctx, "111111111", "203.0.113.7"); remaining != 0 {
		t.Fatalf("remaining after expiry = %v, want 0", remaining)
	}
}
