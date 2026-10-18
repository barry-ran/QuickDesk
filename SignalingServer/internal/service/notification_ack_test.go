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

func newNotificationAckTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("QD_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set QD_TEST_DATABASE_URL to a throwaway Postgres to run the acknowledgement tests")
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

func seedDelivery(t *testing.T, db *gorm.DB, row models.NotificationDelivery) models.NotificationDelivery {
	t.Helper()
	now := time.Now().UTC()
	if row.CreatedAt.IsZero() {
		row.CreatedAt = now
	}
	row.UpdatedAt = now
	if row.Payload == "" {
		row.Payload = `{"event":"device.offline.sustained"}`
	}
	if row.EventType == "" {
		row.EventType = EventDeviceOnlineChanged
	}
	if row.NextAttemptAt.IsZero() {
		row.NextAttemptAt = now
	}
	if err := db.Create(&row).Error; err != nil {
		t.Fatalf("seed delivery: %v", err)
	}
	return row
}

// "The endpoint answered 200" and "a person saw this" are different facts, and
// acking is how the second one gets recorded.
func TestAcknowledgeMarksDeliveredDeliveryOnce(t *testing.T) {
	db := newNotificationAckTestDB(t)
	svc := NewNotificationDeliveryService(db)
	ctx := context.Background()
	row := seedDelivery(t, db, models.NotificationDelivery{
		ChannelID: 7, UserID: 1, Status: DeliveryStatusDelivered, Attempts: 1, LastStatus: 200,
	})

	acked, err := svc.Acknowledge(ctx, 7, row.ID)
	if err != nil {
		t.Fatalf("acknowledge: %v", err)
	}
	if acked.AckedAt == nil {
		t.Fatal("acknowledge returned no acked_at")
	}
	if acked.Status != DeliveryStatusDelivered {
		t.Fatalf("status = %q, want it untouched at %q", acked.Status, DeliveryStatusDelivered)
	}
	first := *acked.AckedAt

	// Idempotent: the second call must not move the timestamp, because the
	// first one is the record of when a human confirmed it.
	again, err := svc.Acknowledge(ctx, 7, row.ID)
	if err != nil {
		t.Fatalf("second acknowledge: %v", err)
	}
	if again.AckedAt == nil || !again.AckedAt.Equal(first) {
		t.Fatalf("second acknowledge moved acked_at from %s to %v", first, again.AckedAt)
	}

	var stored models.NotificationDelivery
	if err := db.First(&stored, row.ID).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if stored.AckedAt == nil || !stored.AckedAt.Equal(first) {
		t.Fatalf("stored acked_at = %v, want %s", stored.AckedAt, first)
	}
}

// A delivery that has not reached the endpoint yet cannot be confirmed: the
// claim would be unsupported, so it is refused rather than recorded as false.
func TestAcknowledgeRefusesUnfinishedDeliveries(t *testing.T) {
	db := newNotificationAckTestDB(t)
	svc := NewNotificationDeliveryService(db)
	ctx := context.Background()

	for _, status := range []string{DeliveryStatusPending, DeliveryStatusDelivering} {
		row := seedDelivery(t, db, models.NotificationDelivery{
			ChannelID: 8, UserID: 1, Status: status,
		})
		if _, err := svc.Acknowledge(ctx, 8, row.ID); !errors.Is(err, ErrDeliveryNotConfirmable) {
			t.Fatalf("acknowledging a %s delivery returned %v, want ErrDeliveryNotConfirmable", status, err)
		}
	}
}

// A parked failure is still a delivery somebody may have seen (the alert is in
// the operator's list even though the endpoint rejected it), so it can be
// acknowledged - the point of the column is "was this seen", not "did the
// network cooperate".
func TestAcknowledgeAcceptsFailedDelivery(t *testing.T) {
	db := newNotificationAckTestDB(t)
	svc := NewNotificationDeliveryService(db)
	ctx := context.Background()
	row := seedDelivery(t, db, models.NotificationDelivery{
		ChannelID: 9, UserID: 1, Status: DeliveryStatusFailed, Attempts: 8, LastStatus: 500,
	})
	acked, err := svc.Acknowledge(ctx, 9, row.ID)
	if err != nil {
		t.Fatalf("acknowledge a failed delivery: %v", err)
	}
	if acked.AckedAt == nil {
		t.Fatal("a failed delivery was not acknowledged")
	}
}

// The delivery id alone is not enough: it has to sit under the channel the
// caller asked about, or one user could confirm another user's alerts by
// guessing ids.
func TestAcknowledgeIsScopedToItsChannel(t *testing.T) {
	db := newNotificationAckTestDB(t)
	svc := NewNotificationDeliveryService(db)
	ctx := context.Background()
	row := seedDelivery(t, db, models.NotificationDelivery{
		ChannelID: 11, UserID: 1, Status: DeliveryStatusDelivered,
	})
	if _, err := svc.Acknowledge(ctx, 12, row.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("cross-channel acknowledge returned %v, want ErrRecordNotFound", err)
	}
}

// The two numbers the loop is watched with: how many deliveries carry a human
// confirmation, and how many succeeded without one.
func TestAcknowledgementCounters(t *testing.T) {
	db := newNotificationAckTestDB(t)
	svc := NewNotificationDeliveryService(db)
	ctx := context.Background()
	when := time.Now().UTC().Add(-time.Minute)

	seedDelivery(t, db, models.NotificationDelivery{
		ChannelID: 21, UserID: 1, Status: DeliveryStatusDelivered, AckedAt: &when,
	})
	seedDelivery(t, db, models.NotificationDelivery{
		ChannelID: 21, UserID: 1, Status: DeliveryStatusDelivered,
	})
	seedDelivery(t, db, models.NotificationDelivery{
		ChannelID: 21, UserID: 1, Status: DeliveryStatusFailed, AckedAt: &when,
	})
	seedDelivery(t, db, models.NotificationDelivery{
		ChannelID: 22, UserID: 1, Status: DeliveryStatusDelivered,
	})
	seedDelivery(t, db, models.NotificationDelivery{
		ChannelID: 22, UserID: 1, Status: DeliveryStatusPending,
	})

	acked, unacked, err := svc.CountAcknowledgements(ctx)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if acked != 2 {
		t.Fatalf("acked = %d, want 2", acked)
	}
	if unacked != 2 {
		t.Fatalf("unacked = %d, want 2 (a pending row is not an accepted-but-unconfirmed one)", unacked)
	}

	channelUnacked, err := svc.UnackedDelivered(ctx, 22)
	if err != nil {
		t.Fatalf("channel count: %v", err)
	}
	if channelUnacked != 1 {
		t.Fatalf("channel 22 unacked = %d, want 1", channelUnacked)
	}
}
