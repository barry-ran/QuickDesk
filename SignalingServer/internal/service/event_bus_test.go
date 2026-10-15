package service

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

type captureSubscriber struct {
	events chan Event
}

func (s *captureSubscriber) Name() string { return "test-capture" }

func (s *captureSubscriber) HandleEvent(_ context.Context, evt Event) {
	s.events <- evt
}

func TestEventBusPublishesRecordingLifecycleEvent(t *testing.T) {
	redisServer := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	bus := NewEventBus(client)
	subscriber := &captureSubscriber{events: make(chan Event, 1)}
	bus.Subscribe(subscriber)

	bus.Publish(context.Background(), Event{
		Type:     EventRecordingCreated,
		UserID:   42,
		DeviceID: "123456789",
		Data: map[string]interface{}{
			"recording_id": 1,
			"session_id":   "session-1",
		},
	})

	select {
	case evt := <-subscriber.events:
		if evt.Type != EventRecordingCreated {
			t.Fatalf("type = %q, want %q", evt.Type, EventRecordingCreated)
		}
		if evt.UserID != 42 || evt.DeviceID != "123456789" {
			t.Fatalf("unexpected event routing: %+v", evt)
		}
		if evt.ServerRev == 0 {
			t.Fatal("expected server_rev to be assigned")
		}
	case <-time.After(time.Second):
		t.Fatal("subscriber did not receive event")
	}

	stream, err := client.XRange(context.Background(), "qd:events:user:42", "-", "+").Result()
	if err != nil {
		t.Fatalf("read user event stream: %v", err)
	}
	if len(stream) != 1 {
		t.Fatalf("stream entries = %d, want 1", len(stream))
	}
	if stream[0].Values["e"] == nil {
		t.Fatalf("stream entry missing payload: %+v", stream[0])
	}
}
