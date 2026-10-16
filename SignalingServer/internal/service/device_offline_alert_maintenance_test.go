package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"quickdesk/signaling/internal/models"
)

// 2026-10-19 is a Monday; the wrap-midnight and timezone cases are pinned to
// concrete instants so the assertions do not depend on when the test runs.
var monday = time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC)

func TestOfflineAlertMaintenanceActive(t *testing.T) {
	tests := []struct {
		name    string
		windows string
		tz      string
		now     time.Time
		want    bool
	}{
		{
			name: "no windows is never active",
			now:  monday.Add(3 * time.Hour),
			want: false,
		},
		{
			name:    "inside a daytime window",
			windows: `[{"days":[1],"start":"02:00","end":"04:00"}]`,
			tz:      "UTC",
			now:     monday.Add(3 * time.Hour),
			want:    true,
		},
		{
			name:    "after the window ends",
			windows: `[{"days":[1],"start":"02:00","end":"04:00"}]`,
			tz:      "UTC",
			now:     monday.Add(5 * time.Hour),
			want:    false,
		},
		{
			name:    "wrap past midnight uses the start day",
			windows: `[{"days":[1],"start":"23:00","end":"01:00"}]`,
			tz:      "UTC",
			now:     time.Date(2026, 10, 20, 0, 30, 0, 0, time.UTC),
			want:    true,
		},
		{
			name:    "same start and end means all day for the selected days",
			windows: `[{"days":[1],"start":"00:00","end":"00:00"}]`,
			tz:      "UTC",
			now:     monday.Add(13 * time.Hour),
			want:    true,
		},
		{
			name:    "timezone moves the window",
			windows: `[{"days":[1],"start":"09:00","end":"10:00"}]`,
			tz:      "Asia/Shanghai",
			now:     monday.Add(1*time.Hour + 30*time.Minute),
			want:    true,
		},
		{
			name:    "the same instant is outside the window in UTC",
			windows: `[{"days":[1],"start":"09:00","end":"10:00"}]`,
			tz:      "UTC",
			now:     monday.Add(1*time.Hour + 30*time.Minute),
			want:    false,
		},
		{
			// Empty timezone means the server's local zone, matching device
			// access windows. The instant is built in time.Local so the case
			// holds wherever the test runs.
			name:    "empty timezone uses the server local zone",
			windows: `[{"days":[1],"start":"09:00","end":"10:00"}]`,
			now:     time.Date(2026, 10, 19, 9, 30, 0, 0, time.Local),
			want:    true,
		},
		{
			name:    "malformed JSON decodes to no windows",
			windows: `{not json`,
			now:     monday.Add(3 * time.Hour),
			want:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := models.DeviceOfflineAlertRule{
				MaintenanceWindows:  tt.windows,
				MaintenanceTimezone: tt.tz,
			}
			if got := OfflineAlertMaintenanceActive(rule, tt.now); got != tt.want {
				t.Errorf("OfflineAlertMaintenanceActive(%s, %s) = %v, want %v", tt.windows, tt.now, got, tt.want)
			}
		})
	}
}

func TestValidateOfflineAlertMaintenance(t *testing.T) {
	good := []AccessWindow{{Days: []int{1, 2}, Start: "02:00", End: "04:00"}}
	if err := validateOfflineAlertMaintenance(good, "Asia/Shanghai"); err != nil {
		t.Fatalf("a valid window was rejected: %v", err)
	}
	if err := validateOfflineAlertMaintenance(nil, ""); err != nil {
		t.Fatalf("an empty window list must be valid: %v", err)
	}

	tooMany := make([]AccessWindow, MaxOfflineAlertMaintenanceWindows+1)
	for i := range tooMany {
		tooMany[i] = AccessWindow{Start: "01:00", End: "02:00"}
	}
	cases := []struct {
		name    string
		windows []AccessWindow
		tz      string
	}{
		{"too many windows", tooMany, ""},
		{"day out of range", []AccessWindow{{Days: []int{7}, Start: "01:00", End: "02:00"}}, ""},
		{"bad clock", []AccessWindow{{Start: "25:00", End: "02:00"}}, ""},
		{"bad timezone", good, "Mars/Olympus"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateOfflineAlertMaintenance(tc.windows, tc.tz)
			if !errors.Is(err, ErrInvalidOfflineAlertMaintenance) {
				t.Fatalf("err = %v, want ErrInvalidOfflineAlertMaintenance", err)
			}
		})
	}
}

// TestOfflineAlertMaintenanceSuppressesAndResumes is the end-to-end check on
// the evaluator: a window that covers "now" keeps a threshold-crossing outage
// quiet, and the alert goes out on the next tick once the windows are cleared -
// with the episode clock never having been paused.
func TestOfflineAlertMaintenanceSuppressesAndResumes(t *testing.T) {
	db := newOfflineAlertTestDB(t)
	const deviceID = "900000777"
	seedOfflineAlertDevice(t, db, deviceID, 1, true)

	presence := &fakePresence{online: map[string]bool{deviceID: false}}
	bus := NewEventBus(nil)
	captured := &captureSubscriber{events: make(chan Event, 4)}
	bus.Subscribe(captured)

	svc := NewDeviceOfflineAlertService(db, presence, bus)
	ctx := context.Background()
	start := time.Now().UTC()

	// start/end equal with no days reads as "always quiet" (see windowAllows),
	// which makes this case independent of the wall clock.
	alwaysQuiet := []AccessWindow{{Start: "00:00", End: "00:00"}}
	if _, err := svc.Upsert(ctx, deviceID, 1, UpsertOfflineAlertInput{
		Enabled:            true,
		AfterMinutes:       10,
		MaintenanceWindows: alwaysQuiet,
	}); err != nil {
		t.Fatalf("upsert with windows: %v", err)
	}

	if _, err := svc.EvaluateOnce(ctx, start); err != nil {
		t.Fatalf("evaluate 1: %v", err)
	}
	fired, err := svc.EvaluateOnce(ctx, start.Add(11*time.Minute))
	if err != nil {
		t.Fatalf("evaluate 2: %v", err)
	}
	if fired != 0 {
		t.Fatalf("fired %d alerts inside a maintenance window, want 0", fired)
	}
	rule := loadOfflineAlertRule(t, db, deviceID)
	if rule.OfflineSince == nil {
		t.Fatal("the maintenance window paused the episode clock; it must not")
	}
	if rule.AlertedAt != nil {
		t.Fatal("alerted_at was written inside the maintenance window")
	}

	// Clearing the windows lets the already-due alert through on the next tick.
	if _, err := svc.Upsert(ctx, deviceID, 1, UpsertOfflineAlertInput{
		Enabled:      true,
		AfterMinutes: 10,
	}); err != nil {
		t.Fatalf("upsert without windows: %v", err)
	}
	if fired, err = svc.EvaluateOnce(ctx, start.Add(12*time.Minute)); err != nil {
		t.Fatalf("evaluate 3: %v", err)
	}
	if fired != 1 {
		t.Fatalf("fired %d alerts after the window ended, want 1", fired)
	}
	select {
	case evt := <-captured.events:
		if evt.Type != EventDeviceOfflineSustained {
			t.Fatalf("event type = %q, want %q", evt.Type, EventDeviceOfflineSustained)
		}
	case <-time.After(time.Second):
		t.Fatal("no alert event was published after the maintenance window")
	}
}
