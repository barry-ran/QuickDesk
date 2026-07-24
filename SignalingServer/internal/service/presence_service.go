package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// PresenceService manages the two Redis signals that together define
// whether a device is "online":
//
//	qd:presence:device:{id}:hb           TTL 90s, refreshed by heartbeat
//	qd:presence:device:{id}:ws:{inst}    TTL 24h, one per host signaling WS
//	                                      connection; instance UUID prevents
//	                                      the old-process DEL from stomping
//	                                      a new process's key (§2.14)
//	qd:presence:instance:{inst}          TTL 15s, refreshed by each live
//	                                      SignalingServer process. Stale ws
//	                                      keys from crashed/restarted servers
//	                                      are ignored once their instance key
//	                                      expires.
//
// A device is online iff hb exists AND at least one ws:* key exists.
// Either signal alone isn't enough — see §2.4 rationale.
type PresenceService struct {
	rdb        *redis.Client
	instanceID string // unique per running SignalingServer process
}

// PresenceState is the composite returned by IsOnline.
type PresenceState struct {
	Heartbeat bool
	WSCount   int
	Online    bool
}

func (s PresenceState) String() string {
	return fmt.Sprintf("hb=%t ws_count=%d online=%t", s.Heartbeat, s.WSCount, s.Online)
}

// OfflineReason returns the authoritative reason a device is unavailable.
// It is empty while the composite presence state is online.
func (s PresenceState) OfflineReason() string {
	if s.Online {
		return ""
	}
	if !s.Heartbeat && s.WSCount == 0 {
		return "no_heartbeat_and_no_ws"
	}
	if !s.Heartbeat {
		return "heartbeat_missing"
	}
	return "ws_disconnected"
}

const (
	presenceHeartbeatTTL   = 90 * time.Second
	presenceWSTTL          = 24 * time.Hour
	presenceInstanceTTL    = 15 * time.Second
	presenceInstanceTick   = 5 * time.Second
	presenceKnownOnlineSet = "qd:presence:devices:known_online"
)

func NewPresenceService(rdb *redis.Client, instanceID string) *PresenceService {
	return &PresenceService{rdb: rdb, instanceID: instanceID}
}

func (p *PresenceService) hbKey(deviceID string) string {
	return fmt.Sprintf("qd:presence:device:%s:hb", deviceID)
}

func (p *PresenceService) wsKey(deviceID, instance string) string {
	return fmt.Sprintf("qd:presence:device:%s:ws:%s", deviceID, instance)
}

func (p *PresenceService) wsInstancesKey(deviceID string) string {
	return fmt.Sprintf("qd:presence:device:%s:ws_instances", deviceID)
}

func (p *PresenceService) instanceKey(instance string) string {
	return fmt.Sprintf("qd:presence:instance:%s", instance)
}

// Start keeps this server instance marked as live so stale WS presence keys
// from old processes don't keep devices falsely online after a restart.
func (p *PresenceService) Start(ctx context.Context) {
	if p == nil || p.rdb == nil {
		return
	}
	go func() {
		_ = p.markInstanceAlive(ctx)
		ticker := time.NewTicker(presenceInstanceTick)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_ = p.markInstanceAlive(ctx)
			}
		}
	}()
}

func (p *PresenceService) markInstanceAlive(ctx context.Context) error {
	return p.rdb.Set(ctx, p.instanceKey(p.instanceID), "1", presenceInstanceTTL).Err()
}

// Heartbeat refreshes the heartbeat TTL for deviceID. Called from
// POST /v1/devices/:id/heartbeat.
func (p *PresenceService) Heartbeat(ctx context.Context, deviceID string) error {
	return p.rdb.Set(ctx, p.hbKey(deviceID), "1", presenceHeartbeatTTL).Err()
}

// MarkWSConnected records that this instance holds a signaling WS for
// deviceID. Called when the host signaling WS auth succeeds.
func (p *PresenceService) MarkWSConnected(ctx context.Context, deviceID string) error {
	if err := p.markInstanceAlive(ctx); err != nil {
		return err
	}
	pipe := p.rdb.Pipeline()
	pipe.Set(ctx, p.wsKey(deviceID, p.instanceID), "1", presenceWSTTL)
	pipe.SAdd(ctx, p.wsInstancesKey(deviceID), p.instanceID)
	pipe.Expire(ctx, p.wsInstancesKey(deviceID), presenceWSTTL)
	_, err := pipe.Exec(ctx)
	return err
}

// RefreshWSPresence renews the WS presence lease for an already authenticated
// host signaling connection. A long-lived WebSocket must renew this lease so
// it remains online beyond presenceWSTTL.
func (p *PresenceService) RefreshWSPresence(ctx context.Context, deviceID string) error {
	return p.MarkWSConnected(ctx, deviceID)
}

// MarkWSDisconnected deletes this instance's WS presence key for deviceID.
// It's safe to call even when the key doesn't exist.
func (p *PresenceService) MarkWSDisconnected(ctx context.Context, deviceID string) error {
	pipe := p.rdb.Pipeline()
	pipe.Del(ctx, p.wsKey(deviceID, p.instanceID))
	pipe.SRem(ctx, p.wsInstancesKey(deviceID), p.instanceID)
	_, err := pipe.Exec(ctx)
	return err
}

// RememberOnlineCandidate records that deviceID has been observed online or
// close to online. It returns true only when this process won the transition
// from "not remembered" to "remembered", which callers use to publish a
// single online event instead of spamming on every heartbeat.
func (p *PresenceService) RememberOnlineCandidate(ctx context.Context, deviceID string) bool {
	return p.rdb.SAdd(ctx, presenceKnownOnlineSet, deviceID).Val() > 0
}

// ForgetOnlineCandidate removes deviceID from the remembered-online set. It
// returns true when the device was present, letting offline producers suppress
// duplicate offline events from keyspace notifications, WS close, and the
// fallback scanner.
func (p *PresenceService) ForgetOnlineCandidate(ctx context.Context, deviceID string) bool {
	return p.rdb.SRem(ctx, presenceKnownOnlineSet, deviceID).Val() > 0
}

func (p *PresenceService) KnownOnlineCandidates(ctx context.Context) []string {
	ids, err := p.rdb.SMembers(ctx, presenceKnownOnlineSet).Result()
	if err != nil {
		log.Printf("[Presence] smembers known online failed: %v", err)
		return nil
	}
	return ids
}

// State returns a consistent snapshot of the presence signals.
func (p *PresenceService) State(ctx context.Context, deviceID string) PresenceState {
	hb := p.rdb.Exists(ctx, p.hbKey(deviceID)).Val() > 0
	wsCount := p.liveWSCount(ctx, deviceID)

	return PresenceState{
		Heartbeat: hb,
		WSCount:   wsCount,
		Online:    hb && wsCount > 0,
	}
}

func (p *PresenceService) liveWSCount(ctx context.Context, deviceID string) int {
	instances, err := p.rdb.SMembers(ctx, p.wsInstancesKey(deviceID)).Result()
	if err != nil {
		log.Printf("[Presence] smembers ws instances failed for %s: %v", deviceID, err)
		return 0
	}
	if len(instances) == 0 {
		return 0
	}

	pipe := p.rdb.Pipeline()
	type instanceCheck struct {
		id  string
		cmd *redis.IntCmd
	}
	checks := make([]instanceCheck, 0, len(instances))
	for _, instance := range instances {
		if instance == "" {
			continue
		}
		checks = append(checks, instanceCheck{id: instance, cmd: pipe.Exists(ctx, p.instanceKey(instance))})
	}
	if len(checks) == 0 {
		return 0
	}
	if _, err := pipe.Exec(ctx); err != nil {
		log.Printf("[Presence] instance exists pipeline failed: %v", err)
	}
	count := 0
	staleInstances := make([]interface{}, 0)
	for _, check := range checks {
		if check.cmd.Val() > 0 {
			count++
		} else {
			staleInstances = append(staleInstances, check.id)
		}
	}
	if len(staleInstances) > 0 {
		_ = p.rdb.SRem(ctx, p.wsInstancesKey(deviceID), staleInstances...).Err()
	}
	return count
}

// IsOnline is a convenience for State(...).Online.
func (p *PresenceService) IsOnline(ctx context.Context, deviceID string) bool {
	return p.State(ctx, deviceID).Online
}

// BulkState looks up multiple device IDs at once and returns the full
// presence decision inputs. It is intentionally side-effect free except for
// stale ws_instances cleanup inside liveWSCount().
func (p *PresenceService) BulkState(ctx context.Context, deviceIDs []string) map[string]PresenceState {
	out := make(map[string]PresenceState, len(deviceIDs))
	if len(deviceIDs) == 0 {
		return out
	}

	pipe := p.rdb.Pipeline()
	cmds := make(map[string]*redis.IntCmd, len(deviceIDs))
	for _, id := range deviceIDs {
		cmds[id] = pipe.Exists(ctx, p.hbKey(id))
	}
	if _, err := pipe.Exec(ctx); err != nil {
		log.Printf("[Presence] pipeline exists failed: %v", err)
	}
	for _, id := range deviceIDs {
		hb := cmds[id].Val() > 0
		wsCount := 0
		if hb {
			wsCount = p.liveWSCount(ctx, id)
		}
		out[id] = PresenceState{
			Heartbeat: hb,
			WSCount:   wsCount,
			Online:    hb && wsCount > 0,
		}
	}
	return out
}

// BulkOnline looks up multiple device IDs at once, returning a map of
// deviceID → online. Used by list endpoints like GET /v1/me/devices so we
// don't do N round-trips.
func (p *PresenceService) BulkOnline(ctx context.Context, deviceIDs []string) map[string]bool {
	out := make(map[string]bool, len(deviceIDs))
	states := p.BulkState(ctx, deviceIDs)
	for id, state := range states {
		out[id] = state.Online
	}
	return out
}
