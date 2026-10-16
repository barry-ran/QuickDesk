package service

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"quickdesk/signaling/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// TestClaimDueIsExclusiveAcrossWorkers proves the outbox is safe to run from
// more than one worker (or more than one server instance): the
// `SELECT ... FOR UPDATE SKIP LOCKED` claim must hand each pending row to
// exactly one claimer, never twice.
//
// SKIP LOCKED is a Postgres feature, so the test needs a real database and is
// skipped unless QD_TEST_DATABASE_URL points at a *throwaway* database — it
// truncates notification_deliveries first.
func TestClaimDueIsExclusiveAcrossWorkers(t *testing.T) {
	dsn := os.Getenv("QD_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set QD_TEST_DATABASE_URL to a throwaway Postgres to run the claim-safety test")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.NotificationDelivery{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	// The database is ours alone; start from a known empty table.
	if err := db.Exec("TRUNCATE notification_deliveries").Error; err != nil {
		t.Fatalf("truncate: %v", err)
	}

	const (
		rows    = 500
		batch   = 20
		workers = 8
	)

	now := time.Now().UTC()
	seed := make([]models.NotificationDelivery, 0, rows)
	for i := 0; i < rows; i++ {
		seed = append(seed, models.NotificationDelivery{
			ChannelID:     1,
			UserID:        1,
			EventType:     "test",
			Payload:       `{"event":"test"}`,
			Status:        DeliveryStatusPending,
			NextAttemptAt: now,
		})
	}
	if err := db.CreateInBatches(seed, 100).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	svc := NewNotificationDeliveryService(db)

	var mu sync.Mutex
	seen := make(map[uint]int)
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				claimed, err := svc.claimDue(context.Background(), batch)
				if err != nil {
					t.Errorf("claim: %v", err)
					return
				}
				if len(claimed) == 0 {
					return
				}
				mu.Lock()
				for _, r := range claimed {
					seen[r.ID]++
				}
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	if len(seen) != rows {
		t.Errorf("claimed %d distinct rows, want %d", len(seen), rows)
	}
	duplicates := 0
	for _, n := range seen {
		if n != 1 {
			duplicates++
		}
	}
	if duplicates != 0 {
		t.Errorf("%d rows were claimed more than once; SKIP LOCKED is not exclusive", duplicates)
	}

	// Every row must have moved out of pending.
	var pending int64
	if err := db.Model(&models.NotificationDelivery{}).
		Where("status = ?", DeliveryStatusPending).
		Count(&pending).Error; err != nil {
		t.Fatalf("count pending: %v", err)
	}
	if pending != 0 {
		t.Errorf("%d rows still pending; a claimer gave up early", pending)
	}
}

// TestRetryRearmsFailedDelivery covers the dead-letter replay path: only a
// parked `failed` row is retryable, and replaying it resets the attempt budget
// and returns it to `pending` so the worker picks it up on its next pass.
func TestRetryRearmsFailedDelivery(t *testing.T) {
	dsn := os.Getenv("QD_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set QD_TEST_DATABASE_URL to a throwaway Postgres to run the retry test")
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
	svc := NewNotificationDeliveryService(db)
	ctx := context.Background()

	failed := &models.NotificationDelivery{
		ChannelID: 42, UserID: 7, EventType: "test", Payload: `{"event":"test"}`,
		Status: DeliveryStatusFailed, Attempts: notificationMaxAttempts,
		NextAttemptAt: time.Now().UTC(),
	}
	if err := db.Create(failed).Error; err != nil {
		t.Fatalf("seed failed row: %v", err)
	}
	delivered := &models.NotificationDelivery{
		ChannelID: 42, UserID: 7, EventType: "test", Payload: `{"event":"test"}`,
		Status: DeliveryStatusDelivered, Attempts: 1, NextAttemptAt: time.Now().UTC(),
	}
	if err := db.Create(delivered).Error; err != nil {
		t.Fatalf("seed delivered row: %v", err)
	}

	row, err := svc.Retry(ctx, 42, failed.ID)
	if err != nil {
		t.Fatalf("retry failed row: %v", err)
	}
	if row.Status != DeliveryStatusPending || row.Attempts != 0 {
		t.Errorf("after retry status=%q attempts=%d, want pending/0", row.Status, row.Attempts)
	}
	var reloaded models.NotificationDelivery
	if err := db.First(&reloaded, failed.ID).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.Status != DeliveryStatusPending || reloaded.Attempts != 0 {
		t.Errorf("persisted status=%q attempts=%d, want pending/0", reloaded.Status, reloaded.Attempts)
	}

	if _, err := svc.Retry(ctx, 42, delivered.ID); !errors.Is(err, ErrDeliveryNotRetryable) {
		t.Errorf("retrying a delivered row err=%v, want ErrDeliveryNotRetryable", err)
	}
	if _, err := svc.Retry(ctx, 999, failed.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("retrying a foreign channel err=%v, want ErrRecordNotFound", err)
	}
}

// TestPurgeTerminalKeepsInFlightAndRecent protects the retention sweep: only
// finished (delivered/failed) rows past the cutoff may be deleted, and
// in-flight rows are never mistaken for history no matter how old they look.
func TestPurgeTerminalKeepsInFlightAndRecent(t *testing.T) {
	dsn := os.Getenv("QD_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set QD_TEST_DATABASE_URL to a throwaway Postgres to run the retention test")
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
	svc := NewNotificationDeliveryService(db)
	ctx := context.Background()

	now := time.Now().UTC()
	old := now.AddDate(0, 0, -30)
	seed := []models.NotificationDelivery{
		{ChannelID: 1, UserID: 1, EventType: "test", Payload: "{}", Status: DeliveryStatusDelivered, Attempts: 1, NextAttemptAt: old},
		{ChannelID: 1, UserID: 1, EventType: "test", Payload: "{}", Status: DeliveryStatusFailed, Attempts: 8, NextAttemptAt: old},
		{ChannelID: 1, UserID: 1, EventType: "test", Payload: "{}", Status: DeliveryStatusDelivered, Attempts: 1, NextAttemptAt: now},
		{ChannelID: 1, UserID: 1, EventType: "test", Payload: "{}", Status: DeliveryStatusPending, Attempts: 3, NextAttemptAt: old},
		{ChannelID: 1, UserID: 1, EventType: "test", Payload: "{}", Status: DeliveryStatusDelivering, Attempts: 0, NextAttemptAt: old},
	}
	if err := db.Create(&seed).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}
	// Backdate the rows that should look expired; GORM stamps updated_at on
	// create, so set it explicitly afterwards.
	for i, row := range seed {
		stamp := old
		if i == 2 {
			stamp = now
		}
		if err := db.Model(&models.NotificationDelivery{}).
			Where("id = ?", row.ID).Update("updated_at", stamp).Error; err != nil {
			t.Fatalf("backdate %d: %v", row.ID, err)
		}
	}

	removed, err := svc.PurgeTerminal(ctx, now.AddDate(0, 0, -7))
	if err != nil {
		t.Fatalf("purge: %v", err)
	}
	if removed != 2 {
		t.Errorf("purged %d rows, want 2 (old delivered + old failed)", removed)
	}

	var remaining []models.NotificationDelivery
	if err := db.Order("id").Find(&remaining).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	gotStatuses := make(map[string]int)
	for _, row := range remaining {
		gotStatuses[row.Status]++
	}
	if len(remaining) != 3 {
		t.Fatalf("kept %d rows, want 3 (recent delivered + old pending + old delivering)", len(remaining))
	}
	if gotStatuses[DeliveryStatusPending] != 1 || gotStatuses[DeliveryStatusDelivering] != 1 || gotStatuses[DeliveryStatusDelivered] != 1 {
		t.Errorf("kept statuses = %v, want one each of pending/delivering/delivered", gotStatuses)
	}
}

// TestRetryAllFailedRequeuesOnlyFailedRows covers the bulk replay: every
// `failed` row for the channel is re-armed, and delivered/pending rows are left
// exactly as they were.
func TestRetryAllFailedRequeuesOnlyFailedRows(t *testing.T) {
	dsn := os.Getenv("QD_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set QD_TEST_DATABASE_URL to a throwaway Postgres to run the bulk-retry test")
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
	svc := NewNotificationDeliveryService(db)
	ctx := context.Background()

	now := time.Now().UTC()
	seed := []models.NotificationDelivery{
		{ChannelID: 42, UserID: 1, EventType: "test", Payload: "{}", Status: DeliveryStatusFailed, Attempts: 8, NextAttemptAt: now},
		{ChannelID: 42, UserID: 1, EventType: "test", Payload: "{}", Status: DeliveryStatusFailed, Attempts: 8, NextAttemptAt: now},
		{ChannelID: 42, UserID: 1, EventType: "test", Payload: "{}", Status: DeliveryStatusFailed, Attempts: 8, NextAttemptAt: now},
		{ChannelID: 42, UserID: 1, EventType: "test", Payload: "{}", Status: DeliveryStatusDelivered, Attempts: 1, NextAttemptAt: now},
		{ChannelID: 42, UserID: 1, EventType: "test", Payload: "{}", Status: DeliveryStatusPending, Attempts: 2, NextAttemptAt: now},
		{ChannelID: 7, UserID: 1, EventType: "test", Payload: "{}", Status: DeliveryStatusFailed, Attempts: 8, NextAttemptAt: now},
	}
	if err := db.Create(&seed).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	requeued, err := svc.RetryAllFailed(ctx, 42)
	if err != nil {
		t.Fatalf("retry-all: %v", err)
	}
	if requeued != 3 {
		t.Errorf("requeued %d rows, want 3", requeued)
	}

	var rows []models.NotificationDelivery
	if err := db.Where("channel_id = ?", 42).Find(&rows).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	byStatus := make(map[string]int)
	for _, row := range rows {
		byStatus[row.Status]++
		if row.Status == DeliveryStatusPending && row.Attempts != 0 && row.Attempts != 2 {
			t.Errorf("re-armed row %d attempts=%d, want 0 (the pre-existing pending row keeps 2)", row.ID, row.Attempts)
		}
	}
	if byStatus[DeliveryStatusPending] != 4 || byStatus[DeliveryStatusDelivered] != 1 {
		t.Errorf("after retry-all statuses = %v, want 4 pending (3 re-armed + 1 old) and 1 delivered", byStatus)
	}

	// A different channel's failed row must be untouched.
	var other models.NotificationDelivery
	if err := db.Where("channel_id = ?", 7).First(&other).Error; err != nil {
		t.Fatalf("reload other: %v", err)
	}
	if other.Status != DeliveryStatusFailed || other.Attempts != 8 {
		t.Errorf("channel 7 row status=%q attempts=%d, want failed/8 (untouched)", other.Status, other.Attempts)
	}

	if n, err := svc.RetryAllFailed(ctx, 12345); err != nil || n != 0 {
		t.Errorf("retry-all on empty channel = (%d, %v), want (0, nil)", n, err)
	}
}

// newPurgeTestDB opens the throwaway database and truncates the outbox, the
// shared preamble for the retention tests.
func newPurgeTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("QD_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set QD_TEST_DATABASE_URL to a throwaway Postgres to run the retention tests")
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

// TestPurgeTerminalArchivesBeforeDeleting checks the whole point of the archive
// path: the expired rows reach disk before they leave the database.
func TestPurgeTerminalArchivesBeforeDeleting(t *testing.T) {
	db := newPurgeTestDB(t)
	ctx := context.Background()
	archiveDir := t.TempDir()
	svc := NewNotificationDeliveryService(db).WithArchiveDir(archiveDir)

	now := time.Now().UTC()
	old := now.AddDate(0, 0, -30)
	seed := []models.NotificationDelivery{
		{ChannelID: 5, UserID: 1, EventType: "test", Payload: `{"n":1}`, Status: DeliveryStatusDelivered, Attempts: 1, NextAttemptAt: old},
		{ChannelID: 5, UserID: 1, EventType: "test", Payload: `{"n":2}`, Status: DeliveryStatusFailed, Attempts: 8, NextAttemptAt: old},
		{ChannelID: 5, UserID: 1, EventType: "test", Payload: `{"n":3}`, Status: DeliveryStatusDelivered, Attempts: 1, NextAttemptAt: now},
		{ChannelID: 5, UserID: 1, EventType: "test", Payload: `{"n":4}`, Status: DeliveryStatusPending, Attempts: 2, NextAttemptAt: old},
	}
	if err := db.Create(&seed).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}
	for i, row := range seed {
		stamp := old
		if i == 2 {
			stamp = now
		}
		if err := db.Model(&models.NotificationDelivery{}).Where("id = ?", row.ID).Update("updated_at", stamp).Error; err != nil {
			t.Fatalf("backdate %d: %v", row.ID, err)
		}
	}

	removed, err := svc.PurgeTerminal(ctx, now.AddDate(0, 0, -7))
	if err != nil {
		t.Fatalf("purge: %v", err)
	}
	if removed != 2 {
		t.Errorf("purged %d rows, want 2", removed)
	}

	var remaining []models.NotificationDelivery
	if err := db.Order("id").Find(&remaining).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if len(remaining) != 2 {
		t.Fatalf("kept %d rows, want 2 (recent delivered + old pending)", len(remaining))
	}
	for _, row := range remaining {
		if row.Status == DeliveryStatusDelivered && row.Payload == `{"n":1}` {
			t.Error("an expired delivered row survived")
		}
	}

	entries, err := os.ReadDir(archiveDir)
	if err != nil {
		t.Fatalf("read archive dir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("archive dir has %d files, want 1", len(entries))
	}
	if !strings.HasPrefix(entries[0].Name(), "notification-deliveries-") || !strings.HasSuffix(entries[0].Name(), ".jsonl") {
		t.Errorf("unexpected archive file name %q", entries[0].Name())
	}
	raw, err := os.ReadFile(filepath.Join(archiveDir, entries[0].Name()))
	if err != nil {
		t.Fatalf("read archive: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	if len(lines) != 2 {
		t.Fatalf("archive has %d lines, want 2", len(lines))
	}
	seen := map[string]string{}
	for _, line := range lines {
		var row models.NotificationDelivery
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			t.Fatalf("archive line is not valid JSON: %v (%q)", err, line)
		}
		seen[row.Payload] = row.Status
	}
	if seen[`{"n":1}`] != DeliveryStatusDelivered || seen[`{"n":2}`] != DeliveryStatusFailed {
		t.Errorf("archived rows = %v, want n:1 delivered and n:2 failed", seen)
	}
}

// TestPurgeTerminalKeepsRowsWhenArchiveFails is the safety property: if the
// archive cannot be written, retention must abort instead of deleting the rows.
func TestPurgeTerminalKeepsRowsWhenArchiveFails(t *testing.T) {
	db := newPurgeTestDB(t)
	ctx := context.Background()

	// A regular file where the archive directory should be: MkdirAll fails.
	blocker := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatalf("write blocker: %v", err)
	}
	svc := NewNotificationDeliveryService(db).WithArchiveDir(filepath.Join(blocker, "archive"))

	now := time.Now().UTC()
	old := now.AddDate(0, 0, -30)
	row := &models.NotificationDelivery{
		ChannelID: 5, UserID: 1, EventType: "test", Payload: `{"n":1}`,
		Status: DeliveryStatusFailed, Attempts: 8, NextAttemptAt: old,
	}
	if err := db.Create(row).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := db.Model(&models.NotificationDelivery{}).Where("id = ?", row.ID).Update("updated_at", old).Error; err != nil {
		t.Fatalf("backdate: %v", err)
	}

	removed, err := svc.PurgeTerminal(ctx, now.AddDate(0, 0, -7))
	if err == nil {
		t.Fatal("purge should fail when the archive cannot be written")
	}
	if removed != 0 {
		t.Errorf("purged %d rows despite a failed archive, want 0", removed)
	}
	var count int64
	if err := db.Model(&models.NotificationDelivery{}).Count(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Errorf("kept %d rows, want the expired row to survive a failed archive", count)
	}
}

// TestCountByStatusCoversEveryState checks the admin summary: every status is
// reported, including the ones with no rows, so a dashboard never has to guess
// whether a missing key means zero or "unknown".
func TestCountByStatusCoversEveryState(t *testing.T) {
	db := newPurgeTestDB(t)
	ctx := context.Background()
	svc := NewNotificationDeliveryService(db)
	now := time.Now().UTC()

	empty, err := svc.CountByStatus(ctx)
	if err != nil {
		t.Fatalf("count on empty table: %v", err)
	}
	for _, status := range []string{DeliveryStatusPending, DeliveryStatusDelivering, DeliveryStatusDelivered, DeliveryStatusFailed} {
		if empty[status] != 0 {
			t.Errorf("empty table %s = %d, want 0", status, empty[status])
		}
	}

	seed := []models.NotificationDelivery{
		{ChannelID: 9, UserID: 1, EventType: "test", Payload: "{}", Status: DeliveryStatusDelivered, Attempts: 1, NextAttemptAt: now},
		{ChannelID: 9, UserID: 1, EventType: "test", Payload: "{}", Status: DeliveryStatusDelivered, Attempts: 1, NextAttemptAt: now},
		{ChannelID: 9, UserID: 1, EventType: "test", Payload: "{}", Status: DeliveryStatusFailed, Attempts: 8, NextAttemptAt: now},
		{ChannelID: 9, UserID: 1, EventType: "test", Payload: "{}", Status: DeliveryStatusPending, Attempts: 0, NextAttemptAt: now},
	}
	if err := db.Create(&seed).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	counts, err := svc.CountByStatus(ctx)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	want := map[string]int64{
		DeliveryStatusPending:    1,
		DeliveryStatusDelivering: 0,
		DeliveryStatusDelivered:  2,
		DeliveryStatusFailed:     1,
	}
	for status, expected := range want {
		if counts[status] != expected {
			t.Errorf("%s = %d, want %d", status, counts[status], expected)
		}
	}
}

// TestRetryAllFailedForUserIsUserScoped covers the cross-channel replay: every
// parked delivery of one user is re-armed, and another user's backlog is
// untouched no matter which channels are involved.
func TestRetryAllFailedForUserIsUserScoped(t *testing.T) {
	db := newPurgeTestDB(t)
	ctx := context.Background()
	svc := NewNotificationDeliveryService(db)
	now := time.Now().UTC()

	seed := []models.NotificationDelivery{
		{ChannelID: 10, UserID: 1, EventType: "test", Payload: "{}", Status: DeliveryStatusFailed, Attempts: 8, NextAttemptAt: now},
		{ChannelID: 11, UserID: 1, EventType: "test", Payload: "{}", Status: DeliveryStatusFailed, Attempts: 8, NextAttemptAt: now},
		{ChannelID: 10, UserID: 1, EventType: "test", Payload: "{}", Status: DeliveryStatusDelivered, Attempts: 1, NextAttemptAt: now},
		{ChannelID: 12, UserID: 2, EventType: "test", Payload: "{}", Status: DeliveryStatusFailed, Attempts: 8, NextAttemptAt: now},
	}
	if err := db.Create(&seed).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	requeued, err := svc.RetryAllFailedForUser(ctx, 1)
	if err != nil {
		t.Fatalf("retry-all-for-user: %v", err)
	}
	if requeued != 2 {
		t.Errorf("requeued %d rows, want 2 (both of user 1's channels)", requeued)
	}

	var mine []models.NotificationDelivery
	if err := db.Where("user_id = ?", 1).Find(&mine).Error; err != nil {
		t.Fatalf("reload mine: %v", err)
	}
	pending := 0
	for _, row := range mine {
		if row.Status == DeliveryStatusPending {
			pending++
			if row.Attempts != 0 {
				t.Errorf("re-armed row %d kept attempts=%d, want 0", row.ID, row.Attempts)
			}
		}
	}
	if pending != 2 {
		t.Errorf("user 1 has %d pending rows, want 2 re-armed", pending)
	}

	var theirs models.NotificationDelivery
	if err := db.Where("user_id = ?", 2).First(&theirs).Error; err != nil {
		t.Fatalf("reload theirs: %v", err)
	}
	if theirs.Status != DeliveryStatusFailed || theirs.Attempts != 8 {
		t.Errorf("user 2 row status=%q attempts=%d, want failed/8 (untouched)", theirs.Status, theirs.Attempts)
	}

	// A zero user id would otherwise re-arm the whole table.
	if _, err := svc.RetryAllFailedForUser(ctx, 0); err == nil {
		t.Error("retry-all-for-user(0) should refuse rather than touch every user")
	}
}

// A blanket "replay everything" is often the wrong move - one broken endpoint
// should not drag a healthy device's backlog back through the worker. These
// tests pin the filters, the dry run and the batch cap.
func TestRetryFailedFiltersDryRunAndLimit(t *testing.T) {
	dsn := os.Getenv("QD_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set QD_TEST_DATABASE_URL to a throwaway Postgres to run the replay-filter tests")
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
	svc := NewNotificationDeliveryService(db)
	ctx := context.Background()

	const (
		denied = "device.access.denied"
		online = "device.online.changed"
	)
	now := time.Now().UTC()
	seed := []models.NotificationDelivery{
		// channel 42, user 1
		{ChannelID: 42, UserID: 1, EventType: denied, DeviceID: "dev-A", Payload: "{}", Status: DeliveryStatusFailed, Attempts: 8, NextAttemptAt: now},
		{ChannelID: 42, UserID: 1, EventType: denied, DeviceID: "dev-A", Payload: "{}", Status: DeliveryStatusFailed, Attempts: 8, NextAttemptAt: now},
		{ChannelID: 42, UserID: 1, EventType: denied, DeviceID: "dev-B", Payload: "{}", Status: DeliveryStatusFailed, Attempts: 8, NextAttemptAt: now},
		{ChannelID: 42, UserID: 1, EventType: denied, DeviceID: "dev-B", Payload: "{}", Status: DeliveryStatusFailed, Attempts: 8, NextAttemptAt: now},
		{ChannelID: 42, UserID: 1, EventType: online, DeviceID: "dev-A", Payload: "{}", Status: DeliveryStatusFailed, Attempts: 8, NextAttemptAt: now},
		// A row written before the device_id column existed.
		{ChannelID: 42, UserID: 1, EventType: denied, DeviceID: "", Payload: "{}", Status: DeliveryStatusFailed, Attempts: 8, NextAttemptAt: now},
		// Delivered rows are never in scope, whatever the filters say.
		{ChannelID: 42, UserID: 1, EventType: denied, DeviceID: "dev-A", Payload: "{}", Status: DeliveryStatusDelivered, Attempts: 1, NextAttemptAt: now},
		// Another user's row on another channel, as a scope control.
		{ChannelID: 43, UserID: 2, EventType: denied, DeviceID: "dev-A", Payload: "{}", Status: DeliveryStatusFailed, Attempts: 8, NextAttemptAt: now},
	}
	if err := db.Create(&seed).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}
	failedCount := func(t *testing.T) int64 {
		t.Helper()
		var n int64
		if err := db.Model(&models.NotificationDelivery{}).
			Where("user_id = ? AND status = ?", 1, DeliveryStatusFailed).Count(&n).Error; err != nil {
			t.Fatalf("count failed: %v", err)
		}
		return n
	}

	// 1. A dry run reports the size of the backlog and changes nothing.
	res, err := svc.RetryFailedForUser(ctx, 1, RetryFailedFilters{}, true)
	if err != nil {
		t.Fatalf("dry run: %v", err)
	}
	if res.Matched != 6 || res.Requeued != 0 {
		t.Fatalf("dry run = %+v, want matched 6 / requeued 0", res)
	}
	if n := failedCount(t); n != 6 {
		t.Fatalf("dry run changed the table: %d failed rows left, want 6", n)
	}

	// 2. Filters narrow the match without touching the table.
	res, _ = svc.RetryFailedForUser(ctx, 1, RetryFailedFilters{EventType: denied}, true)
	if res.Matched != 5 {
		t.Fatalf("event_type filter matched %d, want 5 (the online.changed row is out)", res.Matched)
	}
	res, _ = svc.RetryFailedForUser(ctx, 1, RetryFailedFilters{DeviceID: "dev-A"}, true)
	if res.Matched != 3 {
		t.Fatalf("device filter matched %d, want 3 (2 denied + 1 online.changed)", res.Matched)
	}
	res, _ = svc.RetryFailedForUser(ctx, 1, RetryFailedFilters{EventType: denied, DeviceID: "dev-A"}, true)
	if res.Matched != 2 {
		t.Fatalf("combined filter matched %d, want 2", res.Matched)
	}
	res, _ = svc.RetryFailedForUser(ctx, 1, RetryFailedFilters{DeviceID: "dev-Z"}, true)
	if res.Matched != 0 {
		t.Fatalf("unknown device matched %d, want 0", res.Matched)
	}

	// 3. The cap re-arms the oldest rows first and reports both numbers.
	var minFailedID uint
	if err := db.Model(&models.NotificationDelivery{}).
		Where("channel_id = ? AND status = ?", 42, DeliveryStatusFailed).
		Select("MIN(id)").Scan(&minFailedID).Error; err != nil {
		t.Fatalf("min failed id: %v", err)
	}
	res, err = svc.RetryFailedForChannel(ctx, 42, RetryFailedFilters{Limit: 1}, false)
	if err != nil {
		t.Fatalf("capped replay: %v", err)
	}
	if res.Matched != 6 || res.Requeued != 1 {
		t.Fatalf("capped replay = %+v, want matched 6 / requeued 1", res)
	}
	var oldest models.NotificationDelivery
	if err := db.Where("channel_id = ? AND status = ?", 42, DeliveryStatusPending).
		Order("id ASC").First(&oldest).Error; err != nil {
		t.Fatalf("load re-armed row: %v", err)
	}
	if oldest.ID != minFailedID {
		t.Fatalf("re-armed row %d, want the oldest failed row (%d)", oldest.ID, minFailedID)
	}
	if n := failedCount(t); n != 5 {
		t.Fatalf("after a capped replay %d failed rows left, want 5", n)
	}

	// 4. A filtered replay only touches its own slice.
	res, err = svc.RetryFailedForUser(ctx, 1, RetryFailedFilters{EventType: denied, DeviceID: "dev-A"}, false)
	if err != nil {
		t.Fatalf("filtered replay: %v", err)
	}
	if res.Matched != 1 || res.Requeued != 1 {
		// One of the two dev-A denied rows was already re-armed by the cap.
		t.Fatalf("filtered replay = %+v, want matched 1 / requeued 1", res)
	}
	// The other user's identical backlog is untouched.
	var otherFailed int64
	if err := db.Model(&models.NotificationDelivery{}).
		Where("user_id = ? AND status = ?", 2, DeliveryStatusFailed).Count(&otherFailed).Error; err != nil {
		t.Fatalf("count other user: %v", err)
	}
	if otherFailed != 1 {
		t.Fatalf("other user's row changed: %d failed rows, want 1", otherFailed)
	}

	// 5. The user-scoped entry point still refuses a zero id.
	if _, err := svc.RetryFailedForUser(ctx, 0, RetryFailedFilters{}, true); err == nil {
		t.Fatal("RetryFailedForUser(0) must refuse rather than touch every user")
	}
}

// TestClaimDueIsFairAcrossChannels pins the anti-starvation property: a channel
// with a deep backlog must not fill the whole batch on its own, because a
// batch is delivered sequentially and every row it takes from the busy channel
// is a row some other endpoint has to wait behind.
func TestClaimDueIsFairAcrossChannels(t *testing.T) {
	db := newPurgeTestDB(t)
	svc := NewNotificationDeliveryService(db)
	ctx := context.Background()

	// Channel 1 is seeded first so it owns the oldest ids - exactly the setup
	// that made the previous oldest-first claim hand it the entire batch.
	now := time.Now().UTC()
	seed := make([]models.NotificationDelivery, 0, 33)
	for i := 0; i < 30; i++ {
		seed = append(seed, models.NotificationDelivery{
			ChannelID: 1, UserID: 1, EventType: "test", Payload: "{}",
			Status: DeliveryStatusPending, NextAttemptAt: now,
		})
	}
	for i := 0; i < 3; i++ {
		seed = append(seed, models.NotificationDelivery{
			ChannelID: 2, UserID: 1, EventType: "test", Payload: "{}",
			Status: DeliveryStatusPending, NextAttemptAt: now,
		})
	}
	if err := db.Create(&seed).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	claimed, err := svc.claimDue(ctx, notificationClaimBatch)
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	if len(claimed) == 0 {
		t.Fatal("claimed nothing; both channels had due rows")
	}
	if len(claimed) > notificationClaimBatch {
		t.Fatalf("claimed %d rows, want at most %d", len(claimed), notificationClaimBatch)
	}

	perChannel := map[uint]int{}
	for _, row := range claimed {
		perChannel[row.ChannelID]++
	}
	if perChannel[2] != 3 {
		t.Fatalf("quiet channel contributed %d rows, want all 3 - it was starved", perChannel[2])
	}
	if perChannel[1] > notificationPerChannelBatch {
		t.Fatalf("busy channel contributed %d rows, want at most %d", perChannel[1], notificationPerChannelBatch)
	}

	// The busy channel's backlog must still drain on later passes, so the cap
	// is a share of the batch, not an admission limit.
	second, err := svc.claimDue(ctx, notificationClaimBatch)
	if err != nil {
		t.Fatalf("second claim: %v", err)
	}
	secondFromBusy := 0
	for _, row := range second {
		if row.ChannelID == 1 {
			secondFromBusy++
		}
	}
	if secondFromBusy != notificationPerChannelBatch {
		t.Fatalf("second pass took %d rows from the busy channel, want %d", secondFromBusy, notificationPerChannelBatch)
	}
}

// TestRetryFailedSpreadStaggersReplay covers the pacing path: a mass replay
// must not hand the endpoint (and the worker) the whole backlog at once, but it
// must still finish inside the window the caller asked for.
func TestRetryFailedSpreadStaggersReplay(t *testing.T) {
	db := newPurgeTestDB(t)
	svc := NewNotificationDeliveryService(db)
	ctx := context.Background()

	now := time.Now().UTC()
	seed := make([]models.NotificationDelivery, 0, 100)
	for i := 0; i < 100; i++ {
		seed = append(seed, models.NotificationDelivery{
			ChannelID: 42, UserID: 1, EventType: "test", Payload: "{}",
			Status: DeliveryStatusFailed, Attempts: 8, NextAttemptAt: now,
		})
	}
	if err := db.Create(&seed).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	started := time.Now().UTC()
	res, err := svc.RetryFailedForChannel(ctx, 42, RetryFailedFilters{SpreadOverMinutes: 10}, false)
	if err != nil {
		t.Fatalf("spread replay: %v", err)
	}
	if res.Matched != 100 || res.Requeued != 100 {
		t.Fatalf("spread replay = %+v, want matched 100 / requeued 100", res)
	}

	// Nothing may be due at the moment of the call: that is the whole point.
	var immediatelyDue int64
	if err := db.Model(&models.NotificationDelivery{}).
		Where("channel_id = ? AND status = ? AND next_attempt_at <= ?", 42, DeliveryStatusPending, started).
		Count(&immediatelyDue).Error; err != nil {
		t.Fatalf("count immediately due: %v", err)
	}
	if immediatelyDue != 0 {
		t.Fatalf("%d rows were due immediately, want a staggered replay", immediatelyDue)
	}

	// The replay must be spread, not queued behind one giant bucket.
	var buckets int64
	if err := db.Model(&models.NotificationDelivery{}).
		Where("channel_id = ? AND status = ?", 42, DeliveryStatusPending).
		Distinct("next_attempt_at").Count(&buckets).Error; err != nil {
		t.Fatalf("count buckets: %v", err)
	}
	if buckets != notificationReplaySpreadBuckets {
		t.Fatalf("replay used %d distinct due-times, want %d", buckets, notificationReplaySpreadBuckets)
	}

	// ... and it must be done inside the window: the last bucket lands at the
	// end of it, not past it.
	var lastDue time.Time
	if err := db.Model(&models.NotificationDelivery{}).
		Where("channel_id = ? AND status = ?", 42, DeliveryStatusPending).
		Select("MAX(next_attempt_at)").Scan(&lastDue).Error; err != nil {
		t.Fatalf("max due: %v", err)
	}
	windowEnd := started.Add(10 * time.Minute)
	if lastDue.After(windowEnd.Add(time.Second)) {
		t.Fatalf("last bucket due at %s, past the 10m window ending %s", lastDue, windowEnd)
	}
	if lastDue.Before(windowEnd.Add(-2 * time.Minute)) {
		t.Fatalf("last bucket due at %s, well before the 10m window ending %s", lastDue, windowEnd)
	}

	// The attempt budget is reset, so a paced replay is a real second chance.
	var stillExhausted int64
	if err := db.Model(&models.NotificationDelivery{}).
		Where("channel_id = ? AND attempts > 0", 42).Count(&stillExhausted).Error; err != nil {
		t.Fatalf("count attempts: %v", err)
	}
	if stillExhausted != 0 {
		t.Fatalf("%d rows kept their attempt count, want the budget reset", stillExhausted)
	}
}

// TestRetryFailedWindowAndChannelFilters covers the two scoping filters an
// operator reaches for after a partial outage: "only what queued inside this
// window" and "only these endpoints".
//
// The window is over created_at, not updated_at, and that is the property
// worth pinning: updated_at moves on every delivery attempt, so a window over
// it would select a different set each time it was evaluated.
func TestRetryFailedWindowAndChannelFilters(t *testing.T) {
	db := newPurgeTestDB(t)
	svc := NewNotificationDeliveryService(db)
	ctx := context.Background()

	// Whole seconds keep Go's nanosecond timestamps and Postgres' microsecond
	// storage in agreement, so the inclusive bounds below are exact.
	now := time.Now().UTC().Truncate(time.Second)
	const (
		denied = "device.access.denied"
		online = "device.online.changed"
	)
	failed := func(ch uint, at time.Time, eventType, device string, userID uint) models.NotificationDelivery {
		return models.NotificationDelivery{
			ChannelID: ch, UserID: userID, EventType: eventType, DeviceID: device,
			Payload: "{}", Status: DeliveryStatusFailed, Attempts: 8,
			NextAttemptAt: at, CreatedAt: at,
		}
	}
	seed := []models.NotificationDelivery{
		failed(42, now.Add(-3*time.Hour), denied, "dev-A", 1),
		failed(42, now.Add(-2*time.Hour), denied, "dev-A", 1),
		failed(42, now.Add(-1*time.Hour), online, "dev-B", 1),
		failed(44, now.Add(-30*time.Minute), denied, "dev-A", 1),
		failed(44, now.Add(-10*time.Minute), denied, "dev-B", 1),
		failed(45, now.Add(-5*time.Minute), denied, "dev-A", 1),
		// Another user's backlog, as a scope control.
		failed(46, now.Add(-30*time.Minute), denied, "dev-A", 2),
	}
	if err := db.Create(&seed).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	twoHoursAgo := now.Add(-2 * time.Hour)
	hourAgo := now.Add(-1 * time.Hour)
	twentyMinAgo := now.Add(-20 * time.Minute)
	cases := []struct {
		name    string
		filters RetryFailedFilters
		want    int64
	}{
		{"no filter", RetryFailedFilters{}, 6},
		{"created_after only", RetryFailedFilters{CreatedAfter: &twoHoursAgo}, 5},
		{"created_before only", RetryFailedFilters{CreatedBefore: &hourAgo}, 3},
		{"both bounds inclusive", RetryFailedFilters{CreatedAfter: &twoHoursAgo, CreatedBefore: &hourAgo}, 2},
		{"window then narrow", RetryFailedFilters{CreatedAfter: &twentyMinAgo}, 2},
		{"channel ids", RetryFailedFilters{ChannelIDs: []uint{44, 45}}, 3},
		{"channel ids plus event type", RetryFailedFilters{ChannelIDs: []uint{44}, EventType: denied}, 2},
		{"window and channels together", RetryFailedFilters{CreatedAfter: &twentyMinAgo, ChannelIDs: []uint{45}}, 1},
		{"unknown channel", RetryFailedFilters{ChannelIDs: []uint{999}}, 0},
	}
	for _, tc := range cases {
		res, err := svc.RetryFailedForUser(ctx, 1, tc.filters, true)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if res.Matched != tc.want {
			t.Fatalf("%s: matched %d, want %d", tc.name, res.Matched, tc.want)
		}
		if res.Requeued != 0 {
			t.Fatalf("%s: dry run re-armed %d rows", tc.name, res.Requeued)
		}
	}

	// On the per-channel endpoint the channel filter can only narrow further.
	res, err := svc.RetryFailedForChannel(ctx, 42, RetryFailedFilters{ChannelIDs: []uint{44}}, true)
	if err != nil {
		t.Fatalf("per-channel with a foreign channel id: %v", err)
	}
	if res.Matched != 0 {
		t.Fatalf("per-channel replay matched %d rows of another channel, want 0", res.Matched)
	}

	// A real replay with a window only moves its own slice.
	res, err = svc.RetryFailedForUser(ctx, 1, RetryFailedFilters{CreatedAfter: &twentyMinAgo}, false)
	if err != nil {
		t.Fatalf("windowed replay: %v", err)
	}
	if res.Matched != 2 || res.Requeued != 2 {
		t.Fatalf("windowed replay = %+v, want matched 2 / requeued 2", res)
	}
	var pending int64
	if err := db.Model(&models.NotificationDelivery{}).
		Where("user_id = ? AND status = ?", 1, DeliveryStatusPending).Count(&pending).Error; err != nil {
		t.Fatalf("count pending: %v", err)
	}
	if pending != 2 {
		t.Fatalf("%d rows were re-armed, want exactly the 2 inside the window", pending)
	}
	var untouchedOtherUser int64
	if err := db.Model(&models.NotificationDelivery{}).
		Where("user_id = ? AND status = ?", 2, DeliveryStatusFailed).Count(&untouchedOtherUser).Error; err != nil {
		t.Fatalf("count other user: %v", err)
	}
	if untouchedOtherUser != 1 {
		t.Fatalf("other user's backlog changed: %d failed rows, want 1", untouchedOtherUser)
	}
}
