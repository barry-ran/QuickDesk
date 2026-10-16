package handler

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// SessionRevokeBus carries "close this session" commands between signaling
// instances.
//
// A session's socket lives in exactly one process, and that process is the only
// thing that can close it - so a revoke is not something an instance can do on
// its own once the fleet has more than one member. The session row already
// records which instance admitted it, so the channel is *addressed* rather than
// broadcast: publishing to `qd:sessions:instance:{id}:revoke` reaches the one
// process that can act, and the others never see it.
//
// Pub/sub is fire-and-forget, which is why the caller does not trust it: the
// revoke handler publishes and then waits for the session row to actually close
// before reporting success. A command that nobody heard ages out into a 409 and
// the orphan sweep still closes the row later - it never becomes a silent lie.
type SessionRevokeBus struct {
	rdb        *redis.Client
	instanceID string
	onRevoke   func(sessionID string)
}

// NewSessionRevokeBus returns the bus for one process. onRevoke is called for
// every command addressed to this instance; it runs on the subscriber
// goroutine, so it must not block for long.
func NewSessionRevokeBus(rdb *redis.Client, instanceID string, onRevoke func(sessionID string)) *SessionRevokeBus {
	return &SessionRevokeBus{
		rdb:        rdb,
		instanceID: strings.TrimSpace(instanceID),
		onRevoke:   onRevoke,
	}
}

func (b *SessionRevokeBus) channel(instanceID string) string {
	return "qd:sessions:instance:" + instanceID + ":revoke"
}

// Publish asks the instance that owns a session to close it.
func (b *SessionRevokeBus) Publish(ctx context.Context, instanceID, sessionID string) error {
	if b == nil || b.rdb == nil {
		return errors.New("session revoke bus is not configured")
	}
	instanceID = strings.TrimSpace(instanceID)
	sessionID = strings.TrimSpace(sessionID)
	if instanceID == "" || sessionID == "" {
		return errors.New("instance_id and session_id are required")
	}
	return b.rdb.Publish(ctx, b.channel(instanceID), sessionID).Err()
}

// Run subscribes to this instance's own channel until the context ends,
// resubscribing with a backoff if the connection drops - the same shape the
// presence watcher uses, for the same reason: a subscription that dies silently
// would turn every remote revoke into a 409.
func (b *SessionRevokeBus) Run(ctx context.Context) {
	if b == nil || b.rdb == nil || b.instanceID == "" {
		return
	}
	backoff := time.Second
	const maxBackoff = 30 * time.Second
	for {
		if ctx.Err() != nil {
			return
		}
		err := b.runOnce(ctx)
		if ctx.Err() != nil {
			return
		}
		log.Printf("[session-revoke] subscription to %s lost: %v (retry in %s)", b.channel(b.instanceID), err, backoff)
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		if backoff < maxBackoff {
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}
	}
}

func (b *SessionRevokeBus) runOnce(ctx context.Context) error {
	sub := b.rdb.Subscribe(ctx, b.channel(b.instanceID))
	defer sub.Close()
	// Wait for the subscription to be accepted before returning, so a publish
	// that happens right after this instance boots is not dropped.
	if _, err := sub.Receive(ctx); err != nil {
		return err
	}
	ch := sub.Channel()
	log.Printf("[session-revoke] subscribed to %s", b.channel(b.instanceID))
	for {
		select {
		case <-ctx.Done():
			return nil
		case msg, ok := <-ch:
			if !ok {
				return errors.New("pubsub channel closed")
			}
			if sessionID := strings.TrimSpace(msg.Payload); sessionID != "" && b.onRevoke != nil {
				b.onRevoke(sessionID)
			}
		}
	}
}
