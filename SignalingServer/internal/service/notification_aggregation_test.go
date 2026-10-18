package service

import (
	"context"
	"os"
	"testing"
	"time"

	"quickdesk/signaling/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// newNotificationAggregationTestDB gives the aggregation tests their own
// throwaway database, for the same reason the claim test has one: the merge is
// a single Postgres UPDATE with a subquery, which sqlite cannot prove.
func newNotificationAggregationTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("QD_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set QD_TEST_DATABASE_URL to a throwaway Postgres to run the aggregation tests")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.NotificationDelivery{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := db.Exec("TRUNCATE notification_deliveries").Error; err != nil {
		t.Fatalf("truncate: %v", err)
	}
	return db
}

func loadDeliveries(t *testing.T, db *gorm.DB, channelID uint) []models.NotificationDelivery {
	t.Helper()
	var rows []models.NotificationDelivery
	if err := db.Where("channel_id = ?", channelID).Order("id").Find(&rows).Error; err != nil {
		t.Fatalf("load deliveries: %v", err)
	}
	return rows
}

// A burst of the same event about the same device on the same channel has to
// become one delivery that carries how many occurrences it stands for.
func TestNotificationAggregationCollapsesBurst(t *testing.T) {
	db := newNotificationAggregationTestDB(t)
	svc := NewNotificationDeliveryService(db).WithAggregateWindow(30 * time.Second)
	ctx := context.Background()
	ch := models.UserNotificationChannel{ID: 1, UserID: 1}

	for i := 0; i < 5; i++ {
		if err := svc.Enqueue(ctx, ch, EventDeviceOnlineChanged, "126735214",
			map[string]interface{}{"event": EventDeviceOnlineChanged}); err != nil {
			t.Fatalf("enqueue %d: %v", i, err)
		}
	}

	rows := loadDeliveries(t, db, 1)
	if len(rows) != 1 {
		t.Fatalf("wrote %d rows for one burst, want 1", len(rows))
	}
	if rows[0].OccurrenceCount != 5 {
		t.Fatalf("occurrence_count = %d, want 5", rows[0].OccurrenceCount)
	}
	// The hold is what gives the burst time to collapse, so it has to be in the
	// future - and it is fixed, not extended by each merge.
	if !rows[0].NextAttemptAt.After(time.Now().Add(20 * time.Second)) {
		t.Fatalf("next_attempt_at = %s, want roughly 30s out so the burst has time",
			rows[0].NextAttemptAt)
	}
}

// Only identical (channel, event type, device) triples collapse; anything else
// is a different fact and gets its own row. Once the hold lapses a new event
// starts a new row rather than mutating one that is about to be sent.
func TestNotificationAggregationRespectsKeysAndWindow(t *testing.T) {
	db := newNotificationAggregationTestDB(t)
	svc := NewNotificationDeliveryService(db).WithAggregateWindow(30 * time.Second)
	ctx := context.Background()
	ch := models.UserNotificationChannel{ID: 1, UserID: 1}

	mustEnqueue := func(c models.UserNotificationChannel, evtType, deviceID string) {
		t.Helper()
		if err := svc.Enqueue(ctx, c, evtType, deviceID, map[string]interface{}{"event": evtType}); err != nil {
			t.Fatalf("enqueue %s/%s: %v", evtType, deviceID, err)
		}
	}

	mustEnqueue(ch, EventDeviceOnlineChanged, "device-a")
	mustEnqueue(ch, EventDeviceOnlineChanged, "device-a")                                               // merges
	mustEnqueue(ch, EventDeviceOnlineChanged, "device-b")                                               // different device
	mustEnqueue(ch, EventDeviceAccessDenied, "device-a")                                                // different event
	mustEnqueue(models.UserNotificationChannel{ID: 2, UserID: 1}, EventDeviceOnlineChanged, "device-a") // different channel

	if rows := loadDeliveries(t, db, 1); len(rows) != 3 {
		t.Fatalf("channel 1 has %d rows, want 3 (one merged + two distinct)", len(rows))
	}
	if rows := loadDeliveries(t, db, 2); len(rows) != 1 {
		t.Fatalf("channel 2 has %d rows, want its own single row", len(rows))
	}

	// Expire the hold on channel 1's merged row, then enqueue again: the row is
	// about to be delivered, so the new event must not mutate it.
	rows := loadDeliveries(t, db, 1)
	if err := db.Model(&models.NotificationDelivery{}).
		Where("id = ?", rows[0].ID).
		Update("next_attempt_at", time.Now().Add(-time.Second)).Error; err != nil {
		t.Fatalf("expire hold: %v", err)
	}
	mustEnqueue(ch, EventDeviceOnlineChanged, "device-a")
	after := loadDeliveries(t, db, 1)
	if len(after) != 4 {
		t.Fatalf("channel 1 has %d rows after the hold lapsed, want 4", len(after))
	}
	if after[0].OccurrenceCount != 2 {
		t.Fatalf("the expired row's count changed to %d; it must stay 2", after[0].OccurrenceCount)
	}
}

// The default has to stay the historical behaviour: window 0 means every event
// is its own row, delivered as soon as the worker ticks.
func TestNotificationAggregationDisabledByDefault(t *testing.T) {
	db := newNotificationAggregationTestDB(t)
	svc := NewNotificationDeliveryService(db)
	if svc.AggregateWindow() != 0 {
		t.Fatalf("default aggregate window = %v, want 0", svc.AggregateWindow())
	}
	ctx := context.Background()
	ch := models.UserNotificationChannel{ID: 1, UserID: 1}

	for i := 0; i < 3; i++ {
		if err := svc.Enqueue(ctx, ch, EventDeviceOnlineChanged, "device-a",
			map[string]interface{}{"event": EventDeviceOnlineChanged}); err != nil {
			t.Fatalf("enqueue %d: %v", i, err)
		}
	}
	rows := loadDeliveries(t, db, 1)
	if len(rows) != 3 {
		t.Fatalf("wrote %d rows with aggregation off, want 3", len(rows))
	}
	for _, row := range rows {
		if row.OccurrenceCount != 1 {
			t.Fatalf("occurrence_count = %d with aggregation off, want 1", row.OccurrenceCount)
		}
		if row.NextAttemptAt.After(time.Now().Add(time.Second)) {
			t.Fatalf("next_attempt_at = %s, want delivery due immediately", row.NextAttemptAt)
		}
	}
}
