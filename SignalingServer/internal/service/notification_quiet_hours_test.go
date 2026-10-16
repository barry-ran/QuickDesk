package service

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"quickdesk/signaling/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func newQuietHoursTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("QD_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set QD_TEST_DATABASE_URL to a throwaway Postgres to run the quiet-hours tests")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.NotificationQuietHours{}, &models.NotificationDelivery{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := db.Exec("TRUNCATE notification_quiet_hours, notification_deliveries").Error; err != nil {
		t.Fatalf("truncate: %v", err)
	}
	return db
}

// A snooze is an exact instant, so a held alert can be pointed straight at the
// moment it lifts rather than polling until the window happens to be over.
func TestQuietHoursUntilUsesTheSnoozeDeadline(t *testing.T) {
	now := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	deadline := now.Add(30 * time.Minute)

	if got := QuietHoursUntil(models.NotificationQuietHours{SnoozedUntil: &deadline}, now); !got.Equal(deadline) {
		t.Fatalf("snooze until %s resolved to %s", deadline, got)
	}
	past := now.Add(-time.Minute)
	if got := QuietHoursUntil(models.NotificationQuietHours{SnoozedUntil: &past}, now); !got.IsZero() {
		t.Fatalf("a lapsed snooze still reported quiet until %s", got)
	}
	if got := QuietHoursUntil(models.NotificationQuietHours{}, now); !got.IsZero() {
		t.Fatalf("an empty setting reported quiet until %s", got)
	}
}

// Recurring windows are evaluated in the setting's own timezone, and the
// returned instant is the next re-check rather than a computed boundary: the
// window evaluator owns "is it quiet now", and a second implementation of the
// boundary is how the two start disagreeing.
func TestQuietHoursUntilHonoursRecurringWindows(t *testing.T) {
	shanghai, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatalf("load timezone: %v", err)
	}
	row := models.NotificationQuietHours{
		Windows:  `[{"days":[0,1,2,3,4,5,6],"start":"22:00","end":"07:00"}]`,
		Timezone: "Asia/Shanghai",
	}

	night := time.Date(2026, 10, 16, 23, 30, 0, 0, shanghai)
	until := QuietHoursUntil(row, night)
	if until.IsZero() {
		t.Fatal("23:30 local was not treated as quiet")
	}
	if got := until.Sub(night.UTC()); got != quietHoursRecheck {
		t.Fatalf("held until %s after the observation, want the %s re-check", got, quietHoursRecheck)
	}

	noon := time.Date(2026, 10, 16, 12, 0, 0, 0, shanghai)
	if got := QuietHoursUntil(row, noon); !got.IsZero() {
		t.Fatalf("noon local was treated as quiet until %s", got)
	}

	// The same wall clock in a different zone is not the same instant: 23:30
	// UTC is 07:30 in Shanghai and therefore outside the window.
	if got := QuietHoursUntil(row, time.Date(2026, 10, 16, 23, 30, 0, 0, time.UTC)); !got.IsZero() {
		t.Fatalf("the window was evaluated in the wrong timezone (quiet until %s)", got)
	}
}

// A malformed stored row must not hold every alert forever.
func TestQuietHoursWindowsDecodesGarbageAsEmpty(t *testing.T) {
	if got := QuietHoursWindows(`{"not":"a list"}`); len(got) != 0 {
		t.Fatalf("malformed windows decoded to %+v", got)
	}
	if got := QuietHoursWindows(""); len(got) != 0 {
		t.Fatalf("empty windows decoded to %+v", got)
	}
}

// Validation runs before the first write, so a rejected body cannot
// half-apply. A nil database proves it: nothing may be touched on the error
// path.
func TestQuietHoursPutValidatesBeforeWriting(t *testing.T) {
	svc := NewNotificationQuietHoursService(nil)
	ctx := context.Background()
	cases := []struct {
		name string
		in   NotificationQuietHoursInput
	}{
		{"day out of range", NotificationQuietHoursInput{
			Windows: []AccessWindow{{Days: []int{7}, Start: "22:00", End: "07:00"}},
		}},
		{"malformed time", NotificationQuietHoursInput{
			Windows: []AccessWindow{{Days: []int{1}, Start: "10pm", End: "07:00"}},
		}},
		{"unknown timezone", NotificationQuietHoursInput{
			Windows:  []AccessWindow{{Days: []int{1}, Start: "22:00", End: "07:00"}},
			Timezone: "Mars/Olympus",
		}},
		{"too many windows", NotificationQuietHoursInput{
			Windows: repeatWindows(MaxQuietHoursWindows + 1),
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := svc.Put(ctx, 1, tc.in); !errors.Is(err, ErrInvalidQuietHours) {
				t.Fatalf("Put returned %v, want ErrInvalidQuietHours", err)
			}
		})
	}
}

func repeatWindows(n int) []AccessWindow {
	windows := make([]AccessWindow, 0, n)
	for i := 0; i < n; i++ {
		windows = append(windows, AccessWindow{Days: []int{i % 7}, Start: "22:00", End: "07:00"})
	}
	return windows
}

// The endpoint reports the documented defaults for a user who never saved a
// setting, so a client never has to interpret a null.
func TestQuietHoursViewReportsDefaultsWithoutARow(t *testing.T) {
	view := QuietHoursView(nil, time.Now())
	if view.Active || view.ActiveUntil != nil || view.SnoozedUntil != nil {
		t.Fatalf("default view = %+v, want inactive and empty", view)
	}
	if view.Windows == nil || len(view.Windows) != 0 {
		t.Fatalf("default windows = %#v, want an empty array", view.Windows)
	}
}

// The setting survives a round trip through the database, including the
// "no windows" representation that has exactly one spelling.
func TestQuietHoursRoundTrip(t *testing.T) {
	db := newQuietHoursTestDB(t)
	svc := NewNotificationQuietHoursService(db)
	ctx := context.Background()

	if row, err := svc.Get(ctx, 7); err != nil || row != nil {
		t.Fatalf("Get before any write = (%+v, %v), want (nil, nil)", row, err)
	}
	deadline := time.Now().UTC().Add(time.Hour).Truncate(time.Second)
	row, err := svc.Put(ctx, 7, NotificationQuietHoursInput{
		SnoozedUntil: &deadline,
		Windows:      []AccessWindow{{Days: []int{0, 6}, Start: "22:00", End: "07:00"}},
		Timezone:     "Asia/Shanghai",
	})
	if err != nil {
		t.Fatalf("put: %v", err)
	}
	if row.SnoozedUntil == nil || !row.SnoozedUntil.Equal(deadline) {
		t.Fatalf("stored snooze = %v, want %s", row.SnoozedUntil, deadline)
	}
	if got := QuietHoursWindows(row.Windows); len(got) != 1 || got[0].Start != "22:00" {
		t.Fatalf("stored windows = %+v", got)
	}

	// Replacing it with nothing clears what was there rather than leaving a
	// stale snooze behind.
	cleared, err := svc.Put(ctx, 7, NotificationQuietHoursInput{})
	if err != nil {
		t.Fatalf("clear: %v", err)
	}
	if cleared.SnoozedUntil != nil || len(QuietHoursWindows(cleared.Windows)) != 0 || cleared.Timezone != "" {
		t.Fatalf("cleared row = %+v, want everything empty", cleared)
	}
}

// Holding is not a delivery attempt: the endpoint never saw the POST, so
// counting it would let a quiet night park a healthy channel's rows as failed.
func TestQuietHoursHoldKeepsTheDeliveryPending(t *testing.T) {
	db := newQuietHoursTestDB(t)
	ctx := context.Background()
	quiet := NewNotificationQuietHoursService(db)
	deliveries := NewNotificationDeliveryService(db).WithQuietHours(quiet)

	deadline := time.Now().UTC().Add(2 * time.Minute).Truncate(time.Second)
	if _, err := quiet.Put(ctx, 42, NotificationQuietHoursInput{SnoozedUntil: &deadline}); err != nil {
		t.Fatalf("quiet hours: %v", err)
	}
	row := seedDelivery(t, db, models.NotificationDelivery{
		ChannelID: 5, UserID: 42, Status: DeliveryStatusPending,
		NextAttemptAt: time.Now().UTC().Add(-time.Minute),
	})

	deliveries.DrainOnce(ctx)

	var stored models.NotificationDelivery
	if err := db.First(&stored, row.ID).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if stored.Status != DeliveryStatusPending {
		t.Fatalf("held delivery status = %q, want it back to %q", stored.Status, DeliveryStatusPending)
	}
	if stored.Attempts != 0 {
		t.Fatalf("held delivery counted %d attempts, want 0", stored.Attempts)
	}
	if stored.LastStatus != 0 {
		t.Fatalf("held delivery recorded last_status %d, want 0", stored.LastStatus)
	}
	if stored.NextAttemptAt.Before(deadline.Add(-time.Second)) {
		t.Fatalf("held until %s, want it at or after the snooze end %s", stored.NextAttemptAt, deadline)
	}
	if stored.HeldUntil == nil {
		t.Fatal("a held delivery does not say it is held")
	}

	// Once the quiet period is gone the row is attempted again. The channel it
	// points at does not exist in this fixture, so the attempt itself is
	// visible as the row leaving `pending`.
	if _, err := quiet.Put(ctx, 42, NotificationQuietHoursInput{}); err != nil {
		t.Fatalf("clear quiet hours: %v", err)
	}
	if err := db.Model(&models.NotificationDelivery{}).Where("id = ?", row.ID).
		Update("next_attempt_at", time.Now().UTC().Add(-time.Second)).Error; err != nil {
		t.Fatalf("make it due: %v", err)
	}
	deliveries.DrainOnce(ctx)

	if err := db.First(&stored, row.ID).Error; err != nil {
		t.Fatalf("reload after the window: %v", err)
	}
	if stored.Status == DeliveryStatusPending {
		t.Fatal("the delivery was still pending after the quiet period ended")
	}
}

// Changing the setting has to move the rows it was holding - otherwise
// clearing a snooze early would leave the alerts asleep until the deadline that
// no longer exists.
func TestReclockHeldReleasesOnlyHeldRows(t *testing.T) {
	db := newQuietHoursTestDB(t)
	ctx := context.Background()
	deliveries := NewNotificationDeliveryService(db)
	future := time.Now().UTC().Add(time.Hour).Truncate(time.Second)

	held := seedDelivery(t, db, models.NotificationDelivery{
		ChannelID: 6, UserID: 43, Status: DeliveryStatusPending,
		NextAttemptAt: future, HeldUntil: &future,
	})
	// A row waiting out a delivery backoff: same shape, no held_until.
	backoff := seedDelivery(t, db, models.NotificationDelivery{
		ChannelID: 6, UserID: 43, Status: DeliveryStatusPending,
		NextAttemptAt: future,
	})

	moved, err := deliveries.ReclockHeld(ctx, 43, time.Now().UTC(), false)
	if err != nil {
		t.Fatalf("release: %v", err)
	}
	if moved != 1 {
		t.Fatalf("released %d rows, want exactly the held one", moved)
	}

	var storedHeld, storedBackoff models.NotificationDelivery
	if err := db.First(&storedHeld, held.ID).Error; err != nil {
		t.Fatalf("reload held: %v", err)
	}
	if storedHeld.HeldUntil != nil {
		t.Fatalf("released row still reports held_until %v", storedHeld.HeldUntil)
	}
	if !storedHeld.NextAttemptAt.Before(future) {
		t.Fatalf("released row is still due at %s", storedHeld.NextAttemptAt)
	}
	if err := db.First(&storedBackoff, backoff.ID).Error; err != nil {
		t.Fatalf("reload backoff: %v", err)
	}
	if !storedBackoff.NextAttemptAt.Equal(future) {
		t.Fatalf("a backoff row was moved from %s to %s", future, storedBackoff.NextAttemptAt)
	}

	// The other direction: a shortened quiet period pulls held rows forward and
	// keeps them marked as held, so the list still explains why they wait.
	sooner := time.Now().UTC().Add(5 * time.Minute).Truncate(time.Second)
	if err := db.Model(&models.NotificationDelivery{}).Where("id = ?", held.ID).
		Updates(map[string]interface{}{"held_until": future, "next_attempt_at": future}).Error; err != nil {
		t.Fatalf("re-hold: %v", err)
	}
	moved, err = deliveries.ReclockHeld(ctx, 43, sooner, true)
	if err != nil {
		t.Fatalf("re-hold reclock: %v", err)
	}
	if moved != 1 {
		t.Fatalf("re-pointed %d rows, want 1", moved)
	}
	if err := db.First(&storedHeld, held.ID).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if storedHeld.HeldUntil == nil || !storedHeld.HeldUntil.Equal(sooner) {
		t.Fatalf("held_until = %v, want the new quiet end %s", storedHeld.HeldUntil, sooner)
	}
	if !storedHeld.NextAttemptAt.Equal(sooner) {
		t.Fatalf("next_attempt_at = %s, want %s", storedHeld.NextAttemptAt, sooner)
	}
}
