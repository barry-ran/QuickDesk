package service

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestRefreshWSPresenceRenewsDeviceLease(t *testing.T) {
	redisServer := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })

	presence := NewPresenceService(redisClient, "instance-a")
	deviceID := "device-a"
	ctx := context.Background()

	if err := presence.MarkWSConnected(ctx, deviceID); err != nil {
		t.Fatalf("MarkWSConnected() error = %v", err)
	}

	redisServer.FastForward(time.Hour)

	if err := presence.RefreshWSPresence(ctx, deviceID); err != nil {
		t.Fatalf("RefreshWSPresence() error = %v", err)
	}

	if ttl := redisServer.TTL(presence.wsKey(deviceID, presence.instanceID)); ttl <= 23*time.Hour {
		t.Fatalf("WS presence TTL = %v, want renewed near %v", ttl, presenceWSTTL)
	}
	if ttl := redisServer.TTL(presence.wsInstancesKey(deviceID)); ttl <= 23*time.Hour {
		t.Fatalf("WS instance-set TTL = %v, want renewed near %v", ttl, presenceWSTTL)
	}
}

func TestPresenceStateOfflineReason(t *testing.T) {
	tests := []struct {
		name  string
		state PresenceState
		want  string
	}{
		{"online", PresenceState{Heartbeat: true, WSCount: 1, Online: true}, ""},
		{"heartbeat missing", PresenceState{Heartbeat: false, WSCount: 1}, "heartbeat_missing"},
		{"ws disconnected", PresenceState{Heartbeat: true, WSCount: 0}, "ws_disconnected"},
		{"both missing", PresenceState{}, "no_heartbeat_and_no_ws"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.state.OfflineReason(); got != tt.want {
				t.Fatalf("OfflineReason() = %q, want %q", got, tt.want)
			}
		})
	}
}
