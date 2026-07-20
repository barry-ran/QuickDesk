package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"quickdesk/signaling/internal/observability"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// Event is the canonical in-process event envelope. All handler code
// publishes via EventBus.Publish; subscribers (realtime, webhook, audit,
// connection history) receive a copy and process it independently.
//
// UserID is the *intended recipient* for user-scoped events (0 means system
// event, fan-out to admin/webhook only — see docs §2.17 M4).
//
// Data is a JSON-serialisable payload; it MUST NOT contain sensitive fields
// (access_code plaintext, device_secret, refresh_token etc.) per §2.16.
type Event struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	UserID    uint                   `json:"user_id,omitempty"`
	DeviceID  string                 `json:"device_id,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Ts        time.Time              `json:"ts"`
	ServerRev int64                  `json:"server_rev,omitempty"`
}

// Canonical event type strings. Add new ones here so grep can find every
// producer at a glance.
const (
	EventDeviceBound             = "device.bound"
	EventDeviceAdded             = "device.added"
	EventDeviceUnbound           = "device.unbound"
	EventDeviceOwnershipLost     = "device.ownership.lost"
	EventDeviceOnlineChanged     = "device.online.changed"
	EventDeviceSessionUpdated    = "device.session.updated"
	EventDeviceAccessCodeChanged = "device.access_code.changed"
	EventDeviceRemarkChanged     = "device.remark.changed"
	EventDeviceNameChanged       = "device.device_name.changed"
	EventDeviceSecretRotated     = "device.secret.rotated"

	EventFavoriteAdded   = "favorite.added"
	EventFavoriteUpdated = "favorite.updated"
	EventFavoriteRemoved = "favorite.removed"

	EventSessionRevoked  = "session.revoked"
	EventTurnConfigChang = "turn.config.changed"
)

// EventSubscriber is the in-process interface for anything that wants to
// react to events (real-time WS push, webhook dispatcher, audit log,
// connection-history writer). Subscribers are invoked through EventBus'
// bounded worker queue and MUST NOT block indefinitely.
type EventSubscriber interface {
	HandleEvent(ctx context.Context, evt Event)
	Name() string
}

// EventBus fans out events to local subscribers and (for user-scoped
// events) appends them to the per-user Redis stream used by realtime WS
// `resume` 鈥?see 搂2.8.
type EventBus struct {
	rdb       *redis.Client
	mu        sync.RWMutex
	publishMu sync.Mutex
	subs      []EventSubscriber
	stream    time.Duration // stream retention (TTL hint; actual MAXLEN handles size)
	workQueue chan subscriberWork
}

type subscriberWork struct {
	sub EventSubscriber
	evt Event
}

// NewEventBus wires a bus to Redis. The stream TTL is a rough hint; retention
// is primarily capped by XADD MAXLEN.
func NewEventBus(rdb *redis.Client) *EventBus {
	b := &EventBus{rdb: rdb, stream: 5 * time.Minute, workQueue: make(chan subscriberWork, 1024)}
	go b.subscriberWorker()
	return b
}

func (b *EventBus) subscriberWorker() {
	for work := range b.workQueue {
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("[EventBus] subscriber %s panicked on %s: %v", work.sub.Name(), work.evt.Type, r)
				}
			}()
			work.sub.HandleEvent(context.Background(), work.evt)
		}()
	}
}

// Subscribe adds an in-process subscriber. Order of registration determines
// fan-out order. EventBus uses a single worker so all subscribers observe
// events in the same order as EventBus.Publish stamped server_rev.
func (b *EventBus) Subscribe(s EventSubscriber) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subs = append(b.subs, s)
}

// Publish enqueues the event for in-process subscribers and, when UserID is non-zero, appends it to the
// per-user Redis stream that powers realtime resume/snapshot recovery.
//
// NOTE: Callers must Publish *after* the DB transaction that produced the
// event has committed (lightweight outbox-style — see §2.17 decision note:
// QuickDesk single-instance deployment accepts the nanosecond-wide
// "commit ok / publish lost" window in exchange for not maintaining an
// outbox table; the §2.8 snapshot-on-reconnect path is authoritative and
// self-heals any lost delta).
func (b *EventBus) Publish(ctx context.Context, evt Event) {
	b.publishMu.Lock()
	defer b.publishMu.Unlock()

	if evt.ID == "" {
		evt.ID = "evt_" + uuid.New().String()
	}
	if evt.Ts.IsZero() {
		evt.Ts = time.Now().UTC()
	}
	// server_rev is stamped exactly once here so both the in-memory
	// fanout and the stream replay see identical values. INCR on Redis
	// guarantees global monotonicity across every process instance
	// (§2.7).
	if evt.ServerRev == 0 && b.rdb != nil {
		if v, err := b.rdb.Incr(ctx, "qd:events:server_rev").Result(); err == nil {
			evt.ServerRev = v
		}
	}
	if evt.Type == EventSessionRevoked {
		observability.Event("event_bus", "session_revoked", map[string]interface{}{
			"event_id": evt.ID, "family_id": stringField(evt.Data, "family_id"),
			"reason": stringField(evt.Data, "reason"), "server_rev": evt.ServerRev, "user_id": evt.UserID,
		})
	}

	// Append to Redis stream for resume/snapshot replay (§2.8). If the
	// stream write fails, enqueue the event onto the retry list so the
	// RetryWorker can replay it.
	streamOK := true
	if evt.UserID != 0 && b.rdb != nil {
		if payload, err := json.Marshal(evt); err == nil {
			streamKey := fmt.Sprintf("qd:events:user:%d", evt.UserID)
			args := &redis.XAddArgs{
				Stream: streamKey,
				MaxLen: 1000,
				Approx: true,
				Values: map[string]interface{}{"e": string(payload)},
			}
			if cmd := b.rdb.XAdd(ctx, args); cmd.Err() != nil {
				log.Printf("[EventBus] XADD %s failed: %v", streamKey, cmd.Err())
				streamOK = false
			} else {
				// Use a generous TTL (1 hour) so idle users still have
				// replay data after brief disconnects. MAXLEN~1000 handles
				// active-user trimming; TTL handles abandoned streams.
				b.rdb.Expire(ctx, streamKey, 1*time.Hour)
			}
		}
	}

	// Fan out to in-process subscribers through a bounded worker queue.
	// This prevents high-frequency events from spawning unbounded goroutines.
	b.mu.RLock()
	subs := append([]EventSubscriber(nil), b.subs...)
	b.mu.RUnlock()
	for _, s := range subs {
		select {
		case b.workQueue <- subscriberWork{sub: s, evt: evt}:
		default:
			log.Printf("[EventBus] subscriber queue full; dropping %s for %s", evt.Type, s.Name())
		}
	}

	// §2.17 outbox retry list: if the authoritative stream write failed,
	// push the event onto qd:events:retry so a background worker can
	// replay it. We don't block the caller on this.
	if !streamOK && b.rdb != nil {
		if payload, err := json.Marshal(evt); err == nil {
			if lerr := b.rdb.LPush(ctx, eventRetryKey, string(payload)).Err(); lerr != nil {
				log.Printf("[EventBus] retry enqueue failed: %v", lerr)
			} else {
				b.rdb.Expire(ctx, eventRetryKey, 24*time.Hour)
			}
		}
	}
}

func stringField(fields map[string]interface{}, key string) string {
	if fields == nil {
		return ""
	}
	value, _ := fields[key].(string)
	return value
}

// eventRetryKey is the Redis list where events land if their stream write
// or (future) outbox commit fails. A background worker drains it.
const eventRetryKey = "qd:events:retry"

// StartRetryWorker launches a background goroutine that drains the retry
// list every few seconds. Call once at server startup; it stops when ctx
// is cancelled.
func (b *EventBus) StartRetryWorker(ctx context.Context) {
	if b.rdb == nil {
		return
	}
	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				b.drainRetry(ctx)
			}
		}
	}()
}

func (b *EventBus) drainRetry(ctx context.Context) {
	// Process up to 100 pending events per tick so a massive backlog
	// can't starve other Redis traffic.
	for i := 0; i < 100; i++ {
		val, err := b.rdb.RPop(ctx, eventRetryKey).Result()
		if err == redis.Nil {
			return // queue empty
		}
		if err != nil {
			log.Printf("[EventBus] retry RPOP failed: %v", err)
			return
		}
		var evt Event
		if err := json.Unmarshal([]byte(val), &evt); err != nil {
			log.Printf("[EventBus] retry payload unmarshal failed: %v", err)
			continue
		}
		// Only the stream write needs retry — subscribers already fired
		// on the original publish.
		if evt.UserID == 0 {
			continue
		}
		streamKey := fmt.Sprintf("qd:events:user:%d", evt.UserID)
		cmd := b.rdb.XAdd(ctx, &redis.XAddArgs{
			Stream: streamKey,
			MaxLen: 1000,
			Approx: true,
			Values: map[string]interface{}{"e": val},
		})
		if cmd.Err() != nil {
			// Re-queue at the *back* so we don't thrash on a broken key.
			_ = b.rdb.LPush(ctx, eventRetryKey, val).Err()
			log.Printf("[EventBus] retry XADD %s failed: %v", streamKey, cmd.Err())
			return
		}
		b.rdb.Expire(ctx, streamKey, b.stream)
	}
}

// ReadStreamSince returns events for `userID` with stream entry IDs strictly
// greater than `lastID` (pass "0" to fetch from the beginning of whatever
// still lives in the stream). Result is oldest-first.
//
// If the caller's `lastID` is older than whatever the stream currently holds
// (we'd have to return a truncated result), ReadStreamSince still returns
// what it can 鈥?the handler decides whether to fall back to a full snapshot
// by comparing the received count against the expected gap.
// ResumeResult is what callers get back when they ask for events newer
// than a given server_rev. `Truncated` is true when the stream no longer
// holds anything as old as `sinceRev` — the caller should respond with
// `{type:"snapshot_required"}` per §2.8 instead of trying to replay.
type ResumeResult struct {
	Events    []Event
	Truncated bool
}

// EventsSinceRev returns events for `userID` with ServerRev strictly
// greater than `sinceRev`. Result is oldest-first.
//
// Truncation detection: if the stream holds at least one entry but its
// oldest entry's ServerRev is greater than `sinceRev`, we know we lost
// the (sinceRev, oldestRev] window — the caller should fall back to a
// full snapshot rather than serve a partial replay.
func (b *EventBus) EventsSinceRev(ctx context.Context, userID uint, sinceRev int64) (ResumeResult, error) {
	if b.rdb == nil {
		return ResumeResult{}, nil
	}
	key := fmt.Sprintf("qd:events:user:%d", userID)
	res, err := b.rdb.XRevRangeN(ctx, key, "+", "-", 1000).Result()
	if err != nil {
		return ResumeResult{}, err
	}
	for left, right := 0, len(res)-1; left < right; left, right = left+1, right-1 {
		res[left], res[right] = res[right], res[left]
	}

	out := make([]Event, 0, len(res))
	var oldestRev int64 = -1
	for _, msg := range res {
		raw, ok := msg.Values["e"].(string)
		if !ok {
			continue
		}
		var evt Event
		if json.Unmarshal([]byte(raw), &evt) != nil {
			continue
		}
		if oldestRev < 0 {
			oldestRev = evt.ServerRev
		}
		if evt.ServerRev > sinceRev {
			out = append(out, evt)
		}
	}
	// Truncation detection:
	// 1. Stream non-empty AND oldest entry newer than checkpoint → gap trimmed.
	// 2. Stream empty (expired/deleted) AND client has a non-zero checkpoint
	//    → all events were lost, client needs a full snapshot.
	truncated := (oldestRev > 0 && oldestRev > sinceRev+1) ||
		(len(res) == 0 && sinceRev > 0)
	return ResumeResult{Events: out, Truncated: truncated}, nil
}

// ReadStreamSince is kept for backwards compatibility with callers that
// pass a stream entry ID. New code should prefer EventsSinceRev which
// works against the monotonic server_rev surfaced to clients (§2.7).
func (b *EventBus) ReadStreamSince(ctx context.Context, userID uint, lastID string) ([]Event, error) {
	if b.rdb == nil {
		return nil, nil
	}
	if lastID == "" {
		lastID = "0"
	}
	key := fmt.Sprintf("qd:events:user:%d", userID)
	res, err := b.rdb.XRange(ctx, key, "("+lastID, "+").Result()
	if err != nil {
		return nil, err
	}
	out := make([]Event, 0, len(res))
	for _, msg := range res {
		if raw, ok := msg.Values["e"].(string); ok {
			var evt Event
			if err := json.Unmarshal([]byte(raw), &evt); err == nil {
				out = append(out, evt)
			}
		}
	}
	return out, nil
}
