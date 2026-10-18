package service

import (
	"context"
	"os"
	"testing"

	"quickdesk/signaling/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func newAuditTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("QD_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set QD_TEST_DATABASE_URL to a throwaway Postgres to run the audit-trail tests")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.AuditLog{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := db.Exec("DELETE FROM audit_logs").Error; err != nil {
		t.Fatalf("clear audit logs: %v", err)
	}
	return db
}

func loadAuditRow(t *testing.T, db *gorm.DB) models.AuditLog {
	t.Helper()
	var row models.AuditLog
	if err := db.Order("id DESC").First(&row).Error; err != nil {
		t.Fatalf("load audit row: %v", err)
	}
	return row
}

// Deleting a recording destroys evidence, so the trail has to say what was
// deleted and who did it. The who is the point: admin_id means "which
// administrator", so a user-initiated deletion needs its own actor column
// rather than being filed as an admin action.
func TestAuditSubscriberRecordsUserRecordingDeletion(t *testing.T) {
	db := newAuditTestDB(t)
	sub := NewAuditSubscriber(NewAuditService(db))

	sub.HandleEvent(context.Background(), Event{
		Type:   EventRecordingDeleted,
		UserID: 7,
		Data:   map[string]interface{}{"recording_id": uint(4242)},
	})

	row := loadAuditRow(t, db)
	// The action has to match the administrator path's `recording.delete`, so a
	// single filter shows every deletion; the actor columns tell them apart.
	if row.Action != "recording.delete" {
		t.Fatalf("action = %q, want recording.delete", row.Action)
	}
	if row.ResourceType != "session_recording" {
		t.Fatalf("resource_type = %q, want session_recording", row.ResourceType)
	}
	if row.ResourceID != "4242" {
		t.Fatalf("resource_id = %q, want the recording id", row.ResourceID)
	}
	if row.ActorUserID != 7 {
		t.Fatalf("actor_user_id = %d, want 7", row.ActorUserID)
	}
	if row.AdminID != 0 || row.AdminUsername != "" {
		t.Fatalf("a user action must not be filed as admin activity: %+v", row)
	}

	var count int64
	if err := db.Model(&models.AuditLog{}).Count(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("wrote %d rows, want exactly 1", count)
	}
}

// The event's UserID is not always the actor. A rejected access attempt is
// routed to the device owner but caused by a stranger, so attributing it to the
// owner would file someone else's action under their history. Actions the user
// performs on their own resources are the ones that carry an actor.
func TestAuditSubscriberOnlyAttributesRealActors(t *testing.T) {
	db := newAuditTestDB(t)
	sub := NewAuditSubscriber(NewAuditService(db))
	ctx := context.Background()

	sub.HandleEvent(ctx, Event{Type: EventDeviceAccessDenied, UserID: 7, DeviceID: "123456789"})
	if row := loadAuditRow(t, db); row.ActorUserID != 0 {
		t.Fatalf("denied access must not be attributed to the device owner, got actor %d", row.ActorUserID)
	}

	sub.HandleEvent(ctx, Event{Type: EventDevicePolicyChanged, UserID: 7, DeviceID: "123456789"})
	if row := loadAuditRow(t, db); row.ActorUserID != 7 {
		t.Fatalf("the owner changing their own policy is the actor, got actor %d", row.ActorUserID)
	}
}

// Creating a recording is already recorded by the recording index; a row per
// session in the audit trail would only dilute it. The same goes for the
// high-frequency user-scoped events the subscriber has always ignored.
func TestAuditSubscriberIgnoresNonAuditedEvents(t *testing.T) {
	db := newAuditTestDB(t)
	sub := NewAuditSubscriber(NewAuditService(db))
	ctx := context.Background()

	for _, evtType := range []string{EventRecordingCreated, EventRecordingUpdated, EventFavoriteAdded, EventDeviceOnlineChanged} {
		sub.HandleEvent(ctx, Event{Type: evtType, UserID: 7, DeviceID: "123456789"})
	}

	var count int64
	if err := db.Model(&models.AuditLog{}).Count(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Fatalf("wrote %d audit rows for non-audited events, want 0", count)
	}
}

// A user-side row has to answer "from where" the same way the admin path does:
// the producer puts the observed address in the event data and the subscriber
// lifts it onto the row. A missing or non-string value must stay empty rather
// than being stringified into something meaningless.
func TestAuditSubscriberRecordsActorIP(t *testing.T) {
	db := newAuditTestDB(t)
	sub := NewAuditSubscriber(NewAuditService(db))
	ctx := context.Background()

	sub.HandleEvent(ctx, Event{
		Type:   EventRecordingDeleted,
		UserID: 7,
		Data:   map[string]interface{}{"recording_id": uint(1), "ip": "203.0.113.9"},
	})
	if row := loadAuditRow(t, db); row.IP != "203.0.113.9" {
		t.Fatalf("ip = %q, want the actor's observed address", row.IP)
	}

	// No ip in the payload: the column stays empty, and the row is still written.
	sub.HandleEvent(ctx, Event{
		Type:   EventRecordingDeleted,
		UserID: 7,
		Data:   map[string]interface{}{"recording_id": uint(2)},
	})
	if row := loadAuditRow(t, db); row.IP != "" {
		t.Fatalf("ip = %q, want empty when the producer had no address", row.IP)
	}

	// A non-string ip must not be stringified into `map[]`/`7`.
	sub.HandleEvent(ctx, Event{
		Type:   EventRecordingDeleted,
		UserID: 7,
		Data:   map[string]interface{}{"recording_id": uint(3), "ip": 7},
	})
	if row := loadAuditRow(t, db); row.IP != "" {
		t.Fatalf("ip = %q, want empty for a non-string value", row.IP)
	}
}

// Both columns exist so "which admin" and "which user" stay answerable; the
// filter has to be able to ask the second question.
func TestAuditListFiltersByActorUserID(t *testing.T) {
	db := newAuditTestDB(t)
	svc := NewAuditService(db)
	ctx := context.Background()

	svc.LogActor(ctx, 7, "recording.delete", "session_recording", "11", "size=1", "198.51.100.7")
	svc.LogActor(ctx, 8, "recording.delete", "session_recording", "12", "size=2", "198.51.100.8")
	// An admin action has actor_user_id 0 and must never match a user filter.
	svc.Log(ctx, 1, "admin", "recording.delete", "session_recording", "13", "", "198.51.100.1")

	rows, total, err := svc.List(ctx, AuditListParams{ActorUserID: 7, Limit: 10})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 1 || len(rows) != 1 {
		t.Fatalf("filter returned total=%d rows=%d, want exactly the one row for user 7", total, len(rows))
	}
	if rows[0].ActorUserID != 7 || rows[0].ResourceID != "11" {
		t.Fatalf("wrong row for the actor filter: %+v", rows[0])
	}
	if rows[0].IP != "198.51.100.7" {
		t.Fatalf("ip = %q, want the address stored by LogActor", rows[0].IP)
	}

	// No filter still returns everything, so the admin activity view is unchanged.
	if _, total, err = svc.List(ctx, AuditListParams{Limit: 10}); err != nil {
		t.Fatalf("unfiltered list: %v", err)
	}
	if total != 3 {
		t.Fatalf("unfiltered total = %d, want 3", total)
	}
}
