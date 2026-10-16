package service

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"quickdesk/signaling/internal/models"

	"gorm.io/gorm"
)

func seedAuditRowAt(t *testing.T, db *gorm.DB, action string, at time.Time) {
	t.Helper()
	if err := db.Create(&models.AuditLog{
		Action:    action,
		CreatedAt: at,
	}).Error; err != nil {
		t.Fatalf("seed audit row: %v", err)
	}
}

// Retention is the only thing that bounds `audit_logs`, so the window has to
// delete the expired rows and nothing else.
func TestAuditPurgeBeforeDeletesOnlyExpiredRows(t *testing.T) {
	db := newAuditTestDB(t)
	svc := NewAuditService(db)
	ctx := context.Background()

	now := time.Now()
	seedAuditRowAt(t, db, "old.action", now.AddDate(0, 0, -40))
	seedAuditRowAt(t, db, "recent.action", now.AddDate(0, 0, -1))

	purged, err := svc.PurgeBefore(ctx, now.AddDate(0, 0, -30))
	if err != nil {
		t.Fatalf("purge: %v", err)
	}
	if purged != 1 {
		t.Fatalf("purged %d rows, want the one outside the window", purged)
	}
	var remaining []models.AuditLog
	if err := db.Order("id").Find(&remaining).Error; err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(remaining) != 1 || remaining[0].Action != "recent.action" {
		t.Fatalf("remaining = %+v, want only the recent row", remaining)
	}
}

// Audit rows are evidence, so an archive directory has to hold the bytes before
// the delete happens.
func TestAuditPurgeBeforeArchivesBeforeDeleting(t *testing.T) {
	db := newAuditTestDB(t)
	archiveDir := t.TempDir()
	svc := NewAuditService(db).WithArchiveDir(archiveDir)
	ctx := context.Background()

	now := time.Now()
	seedAuditRowAt(t, db, "old.one", now.AddDate(0, 0, -90))
	seedAuditRowAt(t, db, "old.two", now.AddDate(0, 0, -60))
	seedAuditRowAt(t, db, "recent.action", now.AddDate(0, 0, -1))

	purged, err := svc.PurgeBefore(ctx, now.AddDate(0, 0, -30))
	if err != nil {
		t.Fatalf("purge: %v", err)
	}
	if purged != 2 {
		t.Fatalf("purged %d rows, want 2", purged)
	}

	name := "audit-logs-" + time.Now().UTC().Format("2006-01-02") + ".jsonl"
	body, err := os.ReadFile(filepath.Join(archiveDir, name))
	if err != nil {
		t.Fatalf("read archive: %v", err)
	}
	lines := strings.Count(strings.TrimSpace(string(body)), "\n") + 1
	if lines != 2 {
		t.Fatalf("archive holds %d lines, want one per purged row:\n%s", lines, body)
	}
	if !strings.Contains(string(body), "old.one") || !strings.Contains(string(body), "old.two") {
		t.Fatalf("archive is missing a purged row:\n%s", body)
	}
	if count := auditRowCount(t, db); count != 1 {
		t.Fatalf("table holds %d rows after purge, want only the recent one", count)
	}
}

// The archive is the point: if it cannot be written, the delete must not
// happen, because otherwise retention silently becomes the step that destroyed
// the only copy.
func TestAuditPurgeBeforeArchiveFailureDeletesNothing(t *testing.T) {
	db := newAuditTestDB(t)
	ctx := context.Background()

	now := time.Now()
	seedAuditRowAt(t, db, "old.action", now.AddDate(0, 0, -90))

	// A regular file standing where the archive directory should be makes
	// MkdirAll fail, which is the "archive is unusable" case.
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("seed blocker: %v", err)
	}
	svc := NewAuditService(db).WithArchiveDir(filepath.Join(blocker, "archive"))

	purged, err := svc.PurgeBefore(ctx, now.AddDate(0, 0, -30))
	if err == nil {
		t.Fatal("purge reported success even though the archive could not be written")
	}
	if purged != 0 {
		t.Fatalf("purged = %d, want 0 when the archive failed", purged)
	}
	if count := auditRowCount(t, db); count != 1 {
		t.Fatalf("table holds %d rows, want the row kept because the archive failed", count)
	}
}

// The row cap is the age-independent second bound: it must keep exactly the
// newest N rows (by id) and drop the oldest, whatever their timestamps say.
func TestAuditCapRowsKeepsNewest(t *testing.T) {
	db := newAuditTestDB(t)
	ctx := context.Background()
	svc := NewAuditService(db)

	for _, action := range []string{"cap.r1", "cap.r2", "cap.r3", "cap.r4", "cap.r5"} {
		seedAuditRowAt(t, db, action, time.Now())
	}

	removed, err := svc.CapRows(ctx, 3)
	if err != nil {
		t.Fatalf("cap: %v", err)
	}
	if removed != 2 {
		t.Fatalf("removed %d rows, want 2", removed)
	}
	var remaining []models.AuditLog
	if err := db.Order("id").Find(&remaining).Error; err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(remaining) != 3 {
		t.Fatalf("table holds %d rows, want 3", len(remaining))
	}
	if remaining[0].Action != "cap.r3" || remaining[2].Action != "cap.r5" {
		t.Fatalf("kept %q..%q, want the newest three rows",
			remaining[0].Action, remaining[2].Action)
	}
}

// A cap that is already satisfied, or a cap of 0 (disabled), must delete
// nothing - the "off" switch is not a "keep zero" switch.
func TestAuditCapRowsNoopWhenUnderOrUnset(t *testing.T) {
	db := newAuditTestDB(t)
	ctx := context.Background()
	svc := NewAuditService(db)
	seedAuditRowAt(t, db, "cap.a", time.Now())
	seedAuditRowAt(t, db, "cap.b", time.Now())

	if removed, err := svc.CapRows(ctx, 5); err != nil || removed != 0 {
		t.Fatalf("under-cap CapRows(5) = %d,%v, want 0,nil", removed, err)
	}
	if removed, err := svc.CapRows(ctx, 0); err != nil || removed != 0 {
		t.Fatalf("disabled CapRows(0) = %d,%v, want 0,nil", removed, err)
	}
	if count := auditRowCount(t, db); count != 2 {
		t.Fatalf("table holds %d rows, want both kept", count)
	}
}

// The row cap must respect the same archive-before-delete rule as age
// retention: the trimmed rows have to land on disk before they disappear.
func TestAuditCapRowsArchivesBeforeDeleting(t *testing.T) {
	db := newAuditTestDB(t)
	archiveDir := t.TempDir()
	svc := NewAuditService(db).WithArchiveDir(archiveDir)
	ctx := context.Background()

	for _, action := range []string{"cap.old1", "cap.old2", "cap.new"} {
		seedAuditRowAt(t, db, action, time.Now())
	}

	removed, err := svc.CapRows(ctx, 1)
	if err != nil {
		t.Fatalf("cap: %v", err)
	}
	if removed != 2 {
		t.Fatalf("removed %d rows, want 2", removed)
	}
	if count := auditRowCount(t, db); count != 1 {
		t.Fatalf("table holds %d rows after cap, want only the newest", count)
	}

	matches, _ := filepath.Glob(filepath.Join(archiveDir, "*.jsonl"))
	if len(matches) != 1 {
		t.Fatalf("archive files = %v, want exactly one daily file", matches)
	}
	body, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("read archive: %v", err)
	}
	if !strings.Contains(string(body), "cap.old1") || !strings.Contains(string(body), "cap.old2") {
		t.Fatalf("archive is missing a capped row:\n%s", body)
	}
	if strings.Contains(string(body), "cap.new") {
		t.Fatalf("archive holds a row that should have been kept:\n%s", body)
	}
}

// seedAuditActorRow writes one attributed row, which is what the per-actor cap
// works on. adminID and actorUserID are mutually exclusive by convention: an
// admin column wins when both are set.
func seedAuditActorRow(t *testing.T, db *gorm.DB, adminID, actorUserID uint, action string) {
	t.Helper()
	if err := db.Create(&models.AuditLog{
		AdminID:     adminID,
		ActorUserID: actorUserID,
		Action:      action,
		CreatedAt:   time.Now(),
	}).Error; err != nil {
		t.Fatalf("seed attributed audit row: %v", err)
	}
}

func auditRowsFor(t *testing.T, db *gorm.DB, where string, args ...interface{}) []models.AuditLog {
	t.Helper()
	var rows []models.AuditLog
	if err := db.Where(where, args...).Order("id").Find(&rows).Error; err != nil {
		t.Fatalf("list rows (%s): %v", where, err)
	}
	return rows
}

// The whole point of the per-actor cap: the noisy actor pays, everyone else
// keeps their trail. The table-wide cap cannot promise that.
func TestAuditCapRowsPerActorChargesOnlyTheNoisyActor(t *testing.T) {
	db := newAuditTestDB(t)
	svc := NewAuditService(db)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		seedAuditActorRow(t, db, 0, 7, "noisy.action")
	}
	for i := 0; i < 2; i++ {
		seedAuditActorRow(t, db, 0, 9, "quiet.action")
	}
	// Unattributed rows are the system's own; there is nobody to charge them
	// to, so the per-actor cap must leave them alone.
	seedAuditRowAt(t, db, "system.action", time.Now())

	removed, err := svc.CapRowsPerActor(ctx, 2)
	if err != nil {
		t.Fatalf("per-actor cap: %v", err)
	}
	if removed != 3 {
		t.Fatalf("removed %d rows, want the noisy actor's 3 oldest", removed)
	}
	if got := len(auditRowsFor(t, db, "actor_user_id = ?", 7)); got != 2 {
		t.Fatalf("noisy actor holds %d rows, want 2", got)
	}
	if got := len(auditRowsFor(t, db, "actor_user_id = ?", 9)); got != 2 {
		t.Fatalf("quiet actor holds %d rows, want its 2 untouched rows", got)
	}
	if got := len(auditRowsFor(t, db, "action = ?", "system.action")); got != 1 {
		t.Fatalf("unattributed rows = %d, want the system row left alone", got)
	}
	// The kept rows are the newest ones, which is what makes the cap "trim the
	// oldest" rather than "trim at random".
	kept := auditRowsFor(t, db, "actor_user_id = ?", 7)
	if kept[0].ID < 4 {
		t.Fatalf("kept rows start at id %d; the oldest rows should have gone first", kept[0].ID)
	}
}

// An admin with id 7 and an end user with id 7 are different people. Merging
// them would let an admin's work spend a user's budget (and the reverse).
func TestAuditCapRowsPerActorKeepsAdminsAndUsersApart(t *testing.T) {
	db := newAuditTestDB(t)
	svc := NewAuditService(db)
	ctx := context.Background()

	for i := 0; i < 4; i++ {
		seedAuditActorRow(t, db, 7, 0, "admin.action")
	}
	for i := 0; i < 4; i++ {
		seedAuditActorRow(t, db, 0, 7, "user.action")
	}

	removed, err := svc.CapRowsPerActor(ctx, 2)
	if err != nil {
		t.Fatalf("per-actor cap: %v", err)
	}
	if removed != 4 {
		t.Fatalf("removed %d rows, want 2 from each of the two actors", removed)
	}
	if got := len(auditRowsFor(t, db, "admin_id = ?", 7)); got != 2 {
		t.Fatalf("admin 7 holds %d rows, want 2", got)
	}
	if got := len(auditRowsFor(t, db, "admin_id = 0 AND actor_user_id = ?", 7)); got != 2 {
		t.Fatalf("user 7 holds %d rows, want 2", got)
	}
}

// Audit rows are evidence, so the per-actor cap has to archive the bytes before
// it deletes - the same rule the age window and the table-wide cap follow.
func TestAuditCapRowsPerActorArchivesBeforeDeleting(t *testing.T) {
	db := newAuditTestDB(t)
	archiveDir := t.TempDir()
	svc := NewAuditService(db).WithArchiveDir(archiveDir)
	ctx := context.Background()

	for _, action := range []string{"actor.old1", "actor.old2"} {
		seedAuditActorRow(t, db, 0, 3, action)
	}
	seedAuditActorRow(t, db, 0, 3, "actor.new")

	removed, err := svc.CapRowsPerActor(ctx, 1)
	if err != nil {
		t.Fatalf("per-actor cap: %v", err)
	}
	if removed != 2 {
		t.Fatalf("removed %d rows, want 2", removed)
	}
	matches, _ := filepath.Glob(filepath.Join(archiveDir, "*.jsonl"))
	if len(matches) != 1 {
		t.Fatalf("archive files = %v, want exactly one daily file", matches)
	}
	body, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("read archive: %v", err)
	}
	if !strings.Contains(string(body), "actor.old1") || !strings.Contains(string(body), "actor.old2") {
		t.Fatalf("archive is missing a capped row:\n%s", body)
	}
	if strings.Contains(string(body), "actor.new") {
		t.Fatalf("archive holds a row that should have been kept:\n%s", body)
	}
}

// An archive failure must leave the table untouched: a cap is never allowed to
// be the step that destroys the only copy of a trail.
func TestAuditCapRowsPerActorArchiveFailureDeletesNothing(t *testing.T) {
	db := newAuditTestDB(t)
	// A file where the archive directory should be makes MkdirAll fail.
	blocker := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatalf("write blocker: %v", err)
	}
	svc := NewAuditService(db).WithArchiveDir(blocker)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		seedAuditActorRow(t, db, 0, 5, "blocked.action")
	}
	if _, err := svc.CapRowsPerActor(ctx, 1); err == nil {
		t.Fatal("per-actor cap returned no error with an unusable archive directory")
	}
	if got := len(auditRowsFor(t, db, "actor_user_id = ?", 5)); got != 3 {
		t.Fatalf("rows after a failed archive = %d, want all 3 still present", got)
	}
}

// Disabled and under-limit calls are no-ops, and one pass never removes more
// than the batch limit - the worker loops instead.
func TestAuditCapRowsPerActorBatchesAndNoops(t *testing.T) {
	db := newAuditTestDB(t)
	svc := NewAuditService(db)
	ctx := context.Background()

	if removed, err := svc.CapRowsPerActor(ctx, 0); err != nil || removed != 0 {
		t.Fatalf("disabled cap = %d,%v, want 0,nil", removed, err)
	}

	rows := make([]models.AuditLog, 0, auditPurgeBatch+200)
	for i := 0; i < auditPurgeBatch+200; i++ {
		rows = append(rows, models.AuditLog{
			ActorUserID: 4, Action: "bulk.action", CreatedAt: time.Now(),
		})
	}
	if err := db.CreateInBatches(rows, 500).Error; err != nil {
		t.Fatalf("seed bulk rows: %v", err)
	}

	if removed, err := svc.CapRowsPerActor(ctx, 10_000); err != nil || removed != 0 {
		t.Fatalf("under-limit cap = %d,%v, want 0,nil", removed, err)
	}

	// One call removes at most auditPurgeBatch rows, so the caller can bound how
	// long it holds the database for.
	first, err := svc.CapRowsPerActor(ctx, 10)
	if err != nil {
		t.Fatalf("first capped pass: %v", err)
	}
	if first != auditPurgeBatch {
		t.Fatalf("first pass removed %d rows, want the batch limit %d", first, auditPurgeBatch)
	}
	second, err := svc.CapRowsPerActor(ctx, 10)
	if err != nil {
		t.Fatalf("second capped pass: %v", err)
	}
	if second != 190 {
		t.Fatalf("second pass removed %d rows, want the remaining 190", second)
	}
	if third, err := svc.CapRowsPerActor(ctx, 10); err != nil || third != 0 {
		t.Fatalf("third pass = %d,%v, want 0,nil", third, err)
	}
	if got := len(auditRowsFor(t, db, "actor_user_id = ?", 4)); got != 10 {
		t.Fatalf("actor holds %d rows after the cap, want 10", got)
	}
}

func seedAuditScopedRow(t *testing.T, db *gorm.DB, action, resourceType, resourceID string) {
	t.Helper()
	if err := db.Create(&models.AuditLog{
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		CreatedAt:    time.Now(),
	}).Error; err != nil {
		t.Fatalf("seed scoped audit row: %v", err)
	}
}

// A single action type can dominate the table even when every actor is quiet.
// The action cap must charge that action and leave unrelated actions alone.
func TestAuditCapRowsPerActionChargesOnlyTheNoisyAction(t *testing.T) {
	db := newAuditTestDB(t)
	svc := NewAuditService(db)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		seedAuditScopedRow(t, db, "noisy.action", "device", "d-noisy")
	}
	for i := 0; i < 2; i++ {
		seedAuditScopedRow(t, db, "quiet.action", "device", "d-quiet")
	}
	seedAuditScopedRow(t, db, "", "device", "d-actionless")

	removed, err := svc.CapRowsPerAction(ctx, 2)
	if err != nil {
		t.Fatalf("per-action cap: %v", err)
	}
	if removed != 3 {
		t.Fatalf("removed %d rows, want the noisy action's 3 oldest", removed)
	}
	if got := len(auditRowsFor(t, db, "action = ?", "noisy.action")); got != 2 {
		t.Fatalf("noisy action holds %d rows, want 2", got)
	}
	if got := len(auditRowsFor(t, db, "action = ?", "quiet.action")); got != 2 {
		t.Fatalf("quiet action holds %d rows, want its 2 untouched rows", got)
	}
	if got := len(auditRowsFor(t, db, "action = ?", "")); got != 1 {
		t.Fatalf("actionless rows = %d, want the unattributed row left alone", got)
	}
}

// Device rows live under two resource types, but both name the same device id.
// The cap must combine them, and must not mistake a recording id or a batch
// operation for a device.
func TestAuditCapRowsPerDeviceChargesOnlyTheNoisyDevice(t *testing.T) {
	db := newAuditTestDB(t)
	svc := NewAuditService(db)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		seedAuditScopedRow(t, db, "device.delete", "device", "d-noisy")
	}
	for i := 0; i < 5; i++ {
		seedAuditScopedRow(t, db, "device.access.denied", "event", "d-noisy")
	}
	for i := 0; i < 2; i++ {
		seedAuditScopedRow(t, db, "device.unbind", "device", "d-quiet")
	}
	seedAuditScopedRow(t, db, "device.batch", "device", "")
	seedAuditScopedRow(t, db, "recording.delete", "session_recording", "d-noisy")

	removed, err := svc.CapRowsPerDevice(ctx, 3)
	if err != nil {
		t.Fatalf("per-device cap: %v", err)
	}
	if removed != 7 {
		t.Fatalf("removed %d rows, want the noisy device's 7 oldest", removed)
	}
	if got := len(auditRowsFor(t, db, "resource_id = ? AND resource_type IN (?, ?)", "d-noisy", "device", "event")); got != 3 {
		t.Fatalf("noisy device holds %d device/event rows, want 3", got)
	}
	if got := len(auditRowsFor(t, db, "resource_id = ?", "d-quiet")); got != 2 {
		t.Fatalf("quiet device holds %d rows, want its 2 untouched rows", got)
	}
	if got := len(auditRowsFor(t, db, "action = ?", "device.batch")); got != 1 {
		t.Fatalf("unattributed batch rows = %d, want the batch row left alone", got)
	}
	if got := len(auditRowsFor(t, db, "action = ?", "recording.delete")); got != 1 {
		t.Fatalf("recording rows = %d, want the recording row left alone", got)
	}
}

// The new device cap must follow the same archive-before-delete rule as the
// older retention bounds.
func TestAuditCapRowsPerDeviceArchivesBeforeDeleting(t *testing.T) {
	db := newAuditTestDB(t)
	archiveDir := t.TempDir()
	svc := NewAuditService(db).WithArchiveDir(archiveDir)
	ctx := context.Background()

	for _, action := range []string{"device.old1", "device.old2", "device.new"} {
		seedAuditScopedRow(t, db, action, "device", "d-archive")
	}

	removed, err := svc.CapRowsPerDevice(ctx, 1)
	if err != nil {
		t.Fatalf("per-device cap: %v", err)
	}
	if removed != 2 {
		t.Fatalf("removed %d rows, want 2", removed)
	}
	matches, _ := filepath.Glob(filepath.Join(archiveDir, "*.jsonl"))
	if len(matches) != 1 {
		t.Fatalf("archive files = %v, want exactly one daily file", matches)
	}
	body, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("read archive: %v", err)
	}
	if !strings.Contains(string(body), "device.old1") || !strings.Contains(string(body), "device.old2") {
		t.Fatalf("archive is missing a capped row:\n%s", body)
	}
	if strings.Contains(string(body), "device.new") {
		t.Fatalf("archive holds a row that should have been kept:\n%s", body)
	}
}

// Disabled action and device caps are no-ops, just like the older bounds.
func TestAuditCapRowsPerActionAndDeviceNoopWhenDisabled(t *testing.T) {
	db := newAuditTestDB(t)
	svc := NewAuditService(db)
	ctx := context.Background()
	seedAuditScopedRow(t, db, "noop.action", "device", "d-noop")

	if removed, err := svc.CapRowsPerAction(ctx, 0); err != nil || removed != 0 {
		t.Fatalf("disabled action cap = %d,%v, want 0,nil", removed, err)
	}
	if removed, err := svc.CapRowsPerDevice(ctx, 0); err != nil || removed != 0 {
		t.Fatalf("disabled device cap = %d,%v, want 0,nil", removed, err)
	}
	if count := auditRowCount(t, db); count != 1 {
		t.Fatalf("table holds %d rows, want the row kept", count)
	}
}
