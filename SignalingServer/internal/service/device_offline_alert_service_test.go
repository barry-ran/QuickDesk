package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"quickdesk/signaling/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// fakePresence is the evaluator's view of Redis: a plain map the test drives.
// The interesting behaviour in this service is the episode bookkeeping, so the
// presence lookup is the one thing worth faking.
type fakePresence struct {
	online map[string]bool
}

func (f *fakePresence) IsOnline(_ context.Context, deviceID string) bool {
	return f.online[deviceID]
}

// newOfflineAlertTestDB mirrors newPurgeTestDB but also owns `devices`, because
// the evaluator skips devices the owner has logged out of.
func newOfflineAlertTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("QD_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set QD_TEST_DATABASE_URL to a throwaway Postgres to run the offline-alert tests")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.DeviceOfflineAlertRule{}, &models.Device{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := db.Exec("DELETE FROM device_offline_alert_rules").Error; err != nil {
		t.Fatalf("clear rules: %v", err)
	}
	return db
}

func seedOfflineAlertDevice(t *testing.T, db *gorm.DB, deviceID string, userID uint, loggedIn bool) {
	t.Helper()
	ensureOfflineAlertUser(t, db, userID)
	if err := db.Exec("DELETE FROM devices WHERE device_id = ?", deviceID).Error; err != nil {
		t.Fatalf("clear device: %v", err)
	}
	owner := userID
	device := models.Device{
		DeviceID:   deviceID,
		DeviceUUID: "uuid-" + deviceID,
		DeviceName: "workshop-" + deviceID,
		UserID:     &owner,
		LoggedIn:   loggedIn,
	}
	if err := db.Create(&device).Error; err != nil {
		t.Fatalf("seed device: %v", err)
	}
}

// ensureOfflineAlertUser creates the owner row the devices FK requires. The
// id is pinned so the tests can assert on event routing.
func ensureOfflineAlertUser(t *testing.T, db *gorm.DB, userID uint) {
	t.Helper()
	username := fmt.Sprintf("qa-offline-%d", userID)
	// A previous run may have left the same username on another id; clear it
	// so the insert below cannot hit the unique index.
	if err := db.Exec("DELETE FROM users WHERE username = ? AND id <> ?", username, userID).Error; err != nil {
		t.Fatalf("clear user: %v", err)
	}
	if err := db.Exec(
		"INSERT INTO users (id, username, password) VALUES (?, ?, 'test-hash') ON CONFLICT (id) DO NOTHING",
		userID, username,
	).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
}

// TestOfflineAlertStateMachine walks the whole episode lifecycle: an outage has
// to last past the threshold before it alerts, it alerts once, and a later
// outage alerts again rather than being suppressed forever by the first one.
func TestOfflineAlertStateMachine(t *testing.T) {
	db := newOfflineAlertTestDB(t)
	const deviceID = "900000001"
	seedOfflineAlertDevice(t, db, deviceID, 1, true)

	presence := &fakePresence{online: map[string]bool{deviceID: false}}
	bus := NewEventBus(nil)
	captured := &captureSubscriber{events: make(chan Event, 4)}
	bus.Subscribe(captured)

	svc := NewDeviceOfflineAlertService(db, presence, bus)
	ctx := context.Background()
	// Truncated to microseconds: these timestamps round-trip through Postgres,
	// which keeps microseconds, so a nanosecond-precision now() would never
	// compare equal to what comes back.
	start := time.Now().UTC().Truncate(time.Microsecond)

	if _, err := svc.Upsert(ctx, deviceID, 1, UpsertOfflineAlertInput{Enabled: true, AfterMinutes: 10}); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	// 1. First offline observation only starts the episode.
	fired, err := svc.EvaluateOnce(ctx, start)
	if err != nil {
		t.Fatalf("evaluate 1: %v", err)
	}
	if fired != 0 {
		t.Fatalf("first observation fired %d alerts, want 0", fired)
	}
	rule := loadOfflineAlertRule(t, db, deviceID)
	if rule.OfflineSince == nil {
		t.Fatal("offline_since was not recorded; the episode never starts")
	}
	if rule.AlertedAt != nil {
		t.Fatal("alerted_at set before the threshold was crossed")
	}

	// 2. Still inside the threshold: quiet.
	if fired, err = svc.EvaluateOnce(ctx, start.Add(5*time.Minute)); err != nil {
		t.Fatalf("evaluate 2: %v", err)
	}
	if fired != 0 {
		t.Fatalf("fired %d alerts inside the threshold, want 0", fired)
	}

	// 3. Past the threshold: exactly one alert.
	if fired, err = svc.EvaluateOnce(ctx, start.Add(11*time.Minute)); err != nil {
		t.Fatalf("evaluate 3: %v", err)
	}
	if fired != 1 {
		t.Fatalf("fired %d alerts past the threshold, want 1", fired)
	}
	select {
	case evt := <-captured.events:
		if evt.Type != EventDeviceOfflineSustained {
			t.Fatalf("event type = %q, want %q", evt.Type, EventDeviceOfflineSustained)
		}
		if evt.UserID != 1 || evt.DeviceID != deviceID {
			t.Fatalf("event routing = %+v, want user 1 / device %s", evt, deviceID)
		}
		if got := evt.Data["threshold_minutes"]; got != 10 {
			t.Fatalf("threshold_minutes = %v, want 10", got)
		}
		if _, ok := evt.Data["device_name"]; !ok {
			t.Fatal("alert payload should name the device so the owner knows which one")
		}
	case <-time.After(time.Second):
		t.Fatal("no alert event was published")
	}

	// 4. Same episode again: no second alert.
	if fired, err = svc.EvaluateOnce(ctx, start.Add(12*time.Minute)); err != nil {
		t.Fatalf("evaluate 4: %v", err)
	}
	if fired != 0 {
		t.Fatalf("re-fired %d alerts for the same episode, want 0", fired)
	}

	// 5. Back online clears the episode.
	presence.online[deviceID] = true
	if _, err = svc.EvaluateOnce(ctx, start.Add(13*time.Minute)); err != nil {
		t.Fatalf("evaluate 5: %v", err)
	}
	rule = loadOfflineAlertRule(t, db, deviceID)
	if rule.OfflineSince != nil || rule.AlertedAt != nil {
		t.Fatalf("episode state survived a recovery: %+v", rule)
	}

	// 6. A second outage is a new episode and alerts on its own.
	presence.online[deviceID] = false
	if _, err = svc.EvaluateOnce(ctx, start.Add(14*time.Minute)); err != nil {
		t.Fatalf("evaluate 6: %v", err)
	}
	if fired, err = svc.EvaluateOnce(ctx, start.Add(25*time.Minute)); err != nil {
		t.Fatalf("evaluate 7: %v", err)
	}
	if fired != 1 {
		t.Fatalf("second outage fired %d alerts, want 1", fired)
	}
}

// A device the owner logged out of is not "down" - it is parked. Alerting about
// it would train the owner to ignore the alerts.
func TestOfflineAlertSkipsLoggedOutDevices(t *testing.T) {
	db := newOfflineAlertTestDB(t)
	const deviceID = "900000002"
	seedOfflineAlertDevice(t, db, deviceID, 1, false)

	presence := &fakePresence{online: map[string]bool{deviceID: false}}
	svc := NewDeviceOfflineAlertService(db, presence, NewEventBus(nil))
	ctx := context.Background()
	// Truncated to microseconds: these timestamps round-trip through Postgres,
	// which keeps microseconds, so a nanosecond-precision now() would never
	// compare equal to what comes back.
	start := time.Now().UTC().Truncate(time.Microsecond)

	if _, err := svc.Upsert(ctx, deviceID, 1, UpsertOfflineAlertInput{Enabled: true, AfterMinutes: 1}); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	for i := 0; i < 3; i++ {
		fired, err := svc.EvaluateOnce(ctx, start.Add(time.Duration(i)*time.Hour))
		if err != nil {
			t.Fatalf("evaluate %d: %v", i, err)
		}
		if fired != 0 {
			t.Fatalf("fired %d alerts for a logged-out device, want 0", fired)
		}
	}
	rule := loadOfflineAlertRule(t, db, deviceID)
	if rule.OfflineSince != nil {
		t.Fatal("a logged-out device should not start an outage episode")
	}
}

// TestOfflineAlertUpsertValidationAndDisable checks the write path: the
// threshold is bounded, and turning the rule off clears the episode so the next
// time it is switched on the owner does not get an alert about an outage that
// happened while the rule was off.
func TestOfflineAlertUpsertValidationAndDisable(t *testing.T) {
	db := newOfflineAlertTestDB(t)
	const deviceID = "900000003"
	seedOfflineAlertDevice(t, db, deviceID, 7, true)

	presence := &fakePresence{online: map[string]bool{deviceID: false}}
	svc := NewDeviceOfflineAlertService(db, presence, NewEventBus(nil))
	ctx := context.Background()

	for _, bad := range []int{0, -1, MaxOfflineAlertAfterMinutes + 1} {
		_, err := svc.Upsert(ctx, deviceID, 7, UpsertOfflineAlertInput{Enabled: true, AfterMinutes: bad})
		if !errors.Is(err, ErrInvalidOfflineAlertThreshold) {
			t.Fatalf("after_minutes=%d: err = %v, want ErrInvalidOfflineAlertThreshold", bad, err)
		}
	}
	for _, bad := range []int{-1, MaxOfflineAlertRepeatMinutes + 1} {
		_, err := svc.Upsert(ctx, deviceID, 7, UpsertOfflineAlertInput{
			Enabled: true, AfterMinutes: 10, RepeatMinutes: bad,
		})
		if !errors.Is(err, ErrInvalidOfflineAlertRepeat) {
			t.Fatalf("repeat_minutes=%d: err = %v, want ErrInvalidOfflineAlertRepeat", bad, err)
		}
	}
	for _, ok := range []int{MinOfflineAlertAfterMinutes, MaxOfflineAlertAfterMinutes} {
		rule, err := svc.Upsert(ctx, deviceID, 7, UpsertOfflineAlertInput{Enabled: true, AfterMinutes: ok})
		if err != nil {
			t.Fatalf("after_minutes=%d should be accepted: %v", ok, err)
		}
		if rule.AfterMinutes != ok {
			t.Fatalf("after_minutes = %d, want %d", rule.AfterMinutes, ok)
		}
	}

	// Start an episode, then switch the rule off.
	// Truncated to microseconds: these timestamps round-trip through Postgres,
	// which keeps microseconds, so a nanosecond-precision now() would never
	// compare equal to what comes back.
	start := time.Now().UTC().Truncate(time.Microsecond)
	if _, err := svc.EvaluateOnce(ctx, start); err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if rule := loadOfflineAlertRule(t, db, deviceID); rule.OfflineSince == nil {
		t.Fatal("expected an episode to be in progress before disabling")
	}
	if _, err := svc.Upsert(ctx, deviceID, 7, UpsertOfflineAlertInput{Enabled: false, AfterMinutes: 10}); err != nil {
		t.Fatalf("disable: %v", err)
	}
	rule := loadOfflineAlertRule(t, db, deviceID)
	if rule.Enabled {
		t.Fatal("rule should be disabled")
	}
	if rule.OfflineSince != nil || rule.AlertedAt != nil {
		t.Fatalf("disabling must clear the episode state, got %+v", rule)
	}

	// A disabled rule is invisible to the evaluator even though the device is
	// still offline.
	if fired, err := svc.EvaluateOnce(ctx, start.Add(time.Hour)); err != nil || fired != 0 {
		t.Fatalf("disabled rule fired %d alerts (err=%v), want 0", fired, err)
	}

	// Delete is idempotent.
	if err := svc.Delete(ctx, deviceID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if err := svc.Delete(ctx, deviceID); err != nil {
		t.Fatalf("second delete should be a no-op: %v", err)
	}
	if rule, err := svc.Get(ctx, deviceID); err != nil || rule != nil {
		t.Fatalf("Get after delete = %+v (err=%v), want nil", rule, err)
	}
}

func loadOfflineAlertRule(t *testing.T, db *gorm.DB, deviceID string) models.DeviceOfflineAlertRule {
	t.Helper()
	var rule models.DeviceOfflineAlertRule
	if err := db.Where("device_id = ?", deviceID).First(&rule).Error; err != nil {
		t.Fatalf("load rule: %v", err)
	}
	return rule
}

// TestOfflineAlertRepeatAndSnooze covers the two optional behaviours: a long
// outage keeps reminding the owner instead of producing one easily-missed
// message, and a maintenance snooze silences publishing without pausing the
// episode clock.
func TestOfflineAlertRepeatAndSnooze(t *testing.T) {
	db := newOfflineAlertTestDB(t)
	const deviceID = "900000004"
	seedOfflineAlertDevice(t, db, deviceID, 1, true)

	presence := &fakePresence{online: map[string]bool{deviceID: false}}
	bus := NewEventBus(nil)
	captured := &captureSubscriber{events: make(chan Event, 8)}
	bus.Subscribe(captured)

	svc := NewDeviceOfflineAlertService(db, presence, bus)
	ctx := context.Background()
	// Truncated to microseconds: these timestamps round-trip through Postgres,
	// which keeps microseconds, so a nanosecond-precision now() would never
	// compare equal to what comes back.
	start := time.Now().UTC().Truncate(time.Microsecond)

	if _, err := svc.Upsert(ctx, deviceID, 1, UpsertOfflineAlertInput{
		Enabled: true, AfterMinutes: 1, RepeatMinutes: 5,
	}); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	// Episode starts, then the first alert fires once the threshold passes.
	if _, err := svc.EvaluateOnce(ctx, start); err != nil {
		t.Fatalf("evaluate start: %v", err)
	}
	if fired, err := svc.EvaluateOnce(ctx, start.Add(2*time.Minute)); err != nil || fired != 1 {
		t.Fatalf("first alert: fired=%d err=%v, want 1", fired, err)
	}
	assertAlertKind(t, captured, false)

	// Inside the repeat interval: quiet.
	if fired, err := svc.EvaluateOnce(ctx, start.Add(4*time.Minute)); err != nil || fired != 0 {
		t.Fatalf("inside repeat interval: fired=%d err=%v, want 0", fired, err)
	}
	// Past it: a reminder, flagged as such so the receiver can tell the two
	// apart without diffing timestamps.
	if fired, err := svc.EvaluateOnce(ctx, start.Add(8*time.Minute)); err != nil || fired != 1 {
		t.Fatalf("first reminder: fired=%d err=%v, want 1", fired, err)
	}
	assertAlertKind(t, captured, true)

	// A snooze suppresses the next reminder but keeps the episode running.
	snoozeUntil := start.Add(20 * time.Minute)
	if _, err := svc.Upsert(ctx, deviceID, 1, UpsertOfflineAlertInput{
		Enabled: true, AfterMinutes: 1, RepeatMinutes: 5, SnoozedUntil: &snoozeUntil,
	}); err != nil {
		t.Fatalf("snooze: %v", err)
	}
	if fired, err := svc.EvaluateOnce(ctx, start.Add(15*time.Minute)); err != nil || fired != 0 {
		t.Fatalf("snoozed evaluation: fired=%d err=%v, want 0", fired, err)
	}
	rule := loadOfflineAlertRule(t, db, deviceID)
	if rule.OfflineSince == nil || !rule.OfflineSince.Equal(start) {
		t.Fatalf("snooze must not pause the episode clock: offline_since = %v, want %v",
			rule.OfflineSince, start)
	}

	// Once the snooze expires the reminder goes out.
	if fired, err := svc.EvaluateOnce(ctx, start.Add(22*time.Minute)); err != nil || fired != 1 {
		t.Fatalf("after snooze: fired=%d err=%v, want 1", fired, err)
	}
	assertAlertKind(t, captured, true)

	// Clearing the snooze is just an upsert without the field.
	if _, err := svc.Upsert(ctx, deviceID, 1, UpsertOfflineAlertInput{
		Enabled: true, AfterMinutes: 1, RepeatMinutes: 5,
	}); err != nil {
		t.Fatalf("clear snooze: %v", err)
	}
	if rule := loadOfflineAlertRule(t, db, deviceID); rule.SnoozedUntil != nil {
		t.Fatalf("snoozed_until = %v, want cleared", rule.SnoozedUntil)
	}
}

// assertAlertKind drains one event and checks its reminder flag.
func assertAlertKind(t *testing.T, captured *captureSubscriber, wantReminder bool) {
	t.Helper()
	select {
	case evt := <-captured.events:
		if evt.Type != EventDeviceOfflineSustained {
			t.Fatalf("event type = %q, want %q", evt.Type, EventDeviceOfflineSustained)
		}
		if got := evt.Data["reminder"]; got != wantReminder {
			t.Fatalf("reminder = %v, want %v", got, wantReminder)
		}
	case <-time.After(time.Second):
		t.Fatal("expected an offline-alert event")
	}
}
