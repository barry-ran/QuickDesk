package service

import (
	"context"
	"testing"
	"time"

	"quickdesk/signaling/internal/models"

	"gorm.io/gorm"
)

// A recording is fetched in many range requests and its thumbnail re-renders on
// every scroll, so the media-read audit has to collapse a burst into one row
// without collapsing genuinely different reads.
func TestLogActorDedupedCollapsesRepeatsWithinTheWindow(t *testing.T) {
	db := newAuditTestDB(t)
	svc := NewAuditService(db)
	ctx := context.Background()
	const window = 5 * time.Minute

	if wrote := svc.LogActorDeduped(ctx, 7, "recording.download", "session_recording", "42", "size_bytes=10", "203.0.113.7", window); !wrote {
		t.Fatal("the first read should write a row")
	}
	if wrote := svc.LogActorDeduped(ctx, 7, "recording.download", "session_recording", "42", "size_bytes=10", "203.0.113.7", window); wrote {
		t.Fatal("a repeat inside the window must not write a second row")
	}
	if count := auditRowCount(t, db); count != 1 {
		t.Fatalf("wrote %d rows for one read plus a repeat, want 1", count)
	}

	// A different actor, action or recording is a different fact and must land.
	svc.LogActorDeduped(ctx, 8, "recording.download", "session_recording", "42", "", "", window)
	svc.LogActorDeduped(ctx, 7, "recording.thumbnail.download", "session_recording", "42", "", "", window)
	svc.LogActorDeduped(ctx, 7, "recording.download", "session_recording", "43", "", "", window)
	if count := auditRowCount(t, db); count != 4 {
		t.Fatalf("wrote %d rows, want one per distinct (actor, action, recording)", count)
	}

	// window <= 0 means "no dedupe at all".
	svc.LogActorDeduped(ctx, 7, "recording.download", "session_recording", "42", "", "", 0)
	svc.LogActorDeduped(ctx, 7, "recording.download", "session_recording", "42", "", "", 0)
	if count := auditRowCount(t, db); count != 6 {
		t.Fatalf("wrote %d rows with dedupe disabled, want 6", count)
	}
}

// The window is a window, not a permanent "seen this once" latch: a read after
// the window has passed is a new access and has to be recorded.
func TestLogActorDedupedWritesAgainAfterTheWindow(t *testing.T) {
	db := newAuditTestDB(t)
	svc := NewAuditService(db)
	ctx := context.Background()

	// An existing row from ten minutes ago is outside a five-minute window.
	if err := db.Create(&models.AuditLog{
		ActorUserID:  7,
		Action:       "recording.download",
		ResourceType: "session_recording",
		ResourceID:   "42",
		CreatedAt:    time.Now().Add(-10 * time.Minute),
	}).Error; err != nil {
		t.Fatalf("seed old row: %v", err)
	}
	if wrote := svc.LogActorDeduped(ctx, 7, "recording.download", "session_recording", "42", "", "", 5*time.Minute); !wrote {
		t.Fatal("a read after the window expired should write a new row")
	}
	if count := auditRowCount(t, db); count != 2 {
		t.Fatalf("wrote %d rows, want the old one plus the new one", count)
	}
}

func auditRowCount(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	var count int64
	if err := db.Model(&models.AuditLog{}).Count(&count).Error; err != nil {
		t.Fatalf("count audit rows: %v", err)
	}
	return count
}
