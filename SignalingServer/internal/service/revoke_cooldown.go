package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// RevokeCooldown keeps a short "you were just kicked out" window per device and
// source address.
//
// Revoking a session closes the socket, but the access code that admitted it is
// still valid - so without this a controller that was just thrown out can
// reconnect immediately, and the kick buys nothing. The window is keyed by the
// device plus the observed client IP, which is the same identity the approval
// denial cooldown uses and for the same reason: `client_id` is self-declared, so
// it cannot be the thing that keeps somebody out. A TTL key is enough state -
// this is a delay, not a ban, and it has to expire on its own even if the
// process that armed it dies.
type RevokeCooldown struct {
	rdb *redis.Client
	ttl time.Duration
}

func NewRevokeCooldown(rdb *redis.Client, ttl time.Duration) *RevokeCooldown {
	return &RevokeCooldown{rdb: rdb, ttl: ttl}
}

// TTL is the configured window. Zero means the feature is off, which is what an
// operator who explicitly wants immediate reconnects sets.
func (c *RevokeCooldown) TTL() time.Duration {
	if c == nil {
		return 0
	}
	return c.ttl
}

func (c *RevokeCooldown) key(deviceID, ip string) string {
	return fmt.Sprintf("qd:sessions:revoked:%s:%s", deviceID, ip)
}

// Arm starts (or restarts) the window for one device and source. An empty
// address is ignored: a key without an address would lock out everybody, which
// is the opposite of the intent.
func (c *RevokeCooldown) Arm(ctx context.Context, deviceID, ip string) error {
	if c == nil || c.rdb == nil || c.ttl <= 0 {
		return nil
	}
	deviceID = strings.TrimSpace(deviceID)
	ip = strings.TrimSpace(ip)
	if deviceID == "" || ip == "" {
		return nil
	}
	return c.rdb.Set(ctx, c.key(deviceID, ip), "1", c.ttl).Err()
}

// Armed reports whether that device and source are still inside the window.
func (c *RevokeCooldown) Armed(ctx context.Context, deviceID, ip string) (bool, error) {
	if c == nil || c.rdb == nil || c.ttl <= 0 {
		return false, nil
	}
	deviceID = strings.TrimSpace(deviceID)
	ip = strings.TrimSpace(ip)
	if deviceID == "" || ip == "" {
		return false, nil
	}
	n, err := c.rdb.Exists(ctx, c.key(deviceID, ip)).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// Remaining is how long the window still has to run, for a Retry-After hint.
func (c *RevokeCooldown) Remaining(ctx context.Context, deviceID, ip string) (time.Duration, error) {
	if c == nil || c.rdb == nil || c.ttl <= 0 {
		return 0, nil
	}
	deviceID = strings.TrimSpace(deviceID)
	ip = strings.TrimSpace(ip)
	if deviceID == "" || ip == "" {
		return 0, nil
	}
	ttl, err := c.rdb.TTL(ctx, c.key(deviceID, ip)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return 0, nil
		}
		return 0, err
	}
	if ttl < 0 {
		return 0, nil
	}
	return ttl, nil
}
