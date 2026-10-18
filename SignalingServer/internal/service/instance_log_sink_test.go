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

// newInstanceLogTestDB gives the shipping tests a real Postgres, because what
// has to hold (the unique key and ON CONFLICT DO NOTHING) is database
// behaviour, not application behaviour.
func newInstanceLogTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("QD_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set QD_TEST_DATABASE_URL to a throwaway Postgres to run the log-shipping tests")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	// The scratch Postgres has a connection limit and this package opens a pool
	// per test; leaving them idle would eventually make an unrelated test fail
	// with "too many clients".
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	})
	if err := db.AutoMigrate(&models.InstanceLogEntry{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := db.Exec("TRUNCATE instance_log_entries").Error; err != nil {
		t.Fatalf("truncate: %v", err)
	}
	return db
}

func loadInstanceLogRows(t *testing.T, db *gorm.DB) []models.InstanceLogEntry {
	t.Helper()
	var rows []models.InstanceLogEntry
	if err := db.Order("seq").Find(&rows).Error; err != nil {
		t.Fatalf("load shipped rows: %v", err)
	}
	return rows
}

// A shipped line keeps the timestamp the logger put on it, so a cross-instance
// search orders by when the line was written rather than when it was flushed.
func TestInstanceLogSinkShipsLinesWithTheirOwnTimestamp(t *testing.T) {
	db := newInstanceLogTestDB(t)
	ctx := context.Background()
	sink := NewInstanceLogSink(db, "inst-a").WithSource("signaling.log")

	sink.Write([]byte("2026/10/16 08:00:00.123456 component=http event=request status=404\n"))
	sink.Write([]byte("a line from something else with no prefix\n"))
	if stored := sink.Flush(ctx); stored != 2 {
		t.Fatalf("flush stored %d lines, want 2", stored)
	}
	rows := loadInstanceLogRows(t, db)
	if len(rows) != 2 {
		t.Fatalf("shipped rows = %d, want 2", len(rows))
	}
	if rows[0].InstanceID != "inst-a" || rows[0].Source != "signaling.log" || rows[0].Seq != 1 {
		t.Fatalf("first row = %+v, want inst-a/signaling.log/seq 1", rows[0])
	}
	want := time.Date(2026, 10, 16, 8, 0, 0, 123456000, time.UTC)
	if !rows[0].LoggedAt.Equal(want) {
		t.Fatalf("parsed timestamp = %v, want %v", rows[0].LoggedAt, want)
	}
	if rows[0].Line != "2026/10/16 08:00:00.123456 component=http event=request status=404" {
		t.Fatalf("line = %q, want the whole line kept intact", rows[0].Line)
	}
	// A line with no standard prefix still ships; it just gets "now".
	if rows[1].LoggedAt.IsZero() || !rows[1].LoggedAt.After(rows[0].LoggedAt) {
		t.Fatalf("unprefixed line got %v, want a receipt-time timestamp", rows[1].LoggedAt)
	}
}

// The same batch delivered twice must land once: a retry after a timeout that
// had actually committed is the normal failure mode of shipping.
func TestInstanceLogSinkIgnoresAResentBatch(t *testing.T) {
	db := newInstanceLogTestDB(t)
	ctx := context.Background()
	sink := NewInstanceLogSink(db, "inst-a")

	entry := models.InstanceLogEntry{
		InstanceID: "inst-a",
		Source:     DefaultLogSourceName,
		Seq:        1,
		LoggedAt:   time.Now().UTC(),
		Line:       "component=http event=request status=404",
	}
	if stored := sink.store(ctx, []models.InstanceLogEntry{entry}); stored != 1 {
		t.Fatalf("first delivery stored %d rows, want 1", stored)
	}
	if stored := sink.store(ctx, []models.InstanceLogEntry{entry}); stored != 0 {
		t.Fatalf("re-sent batch stored %d rows, want 0", stored)
	}
	if rows := loadInstanceLogRows(t, db); len(rows) != 1 {
		t.Fatalf("table holds %d rows after the retry, want 1", len(rows))
	}
}

// The buffer is the whole back-pressure story: when it fills, lines are dropped
// and counted. Blocking would mean an incident - the time logs matter most -
// stops answering requests.
func TestInstanceLogSinkDropsInsteadOfBlocking(t *testing.T) {
	db := newInstanceLogTestDB(t)
	sink := NewInstanceLogSink(db, "inst-a").WithBuffer(1)

	for i := 0; i < 5; i++ {
		if _, err := sink.Write([]byte("component=http event=request status=500\n")); err != nil {
			t.Fatalf("write %d returned %v; a sink must never fail a log call", i, err)
		}
	}
	if dropped := sink.Dropped(); dropped != 4 {
		t.Fatalf("dropped = %d, want 4 (one line fits the buffer)", dropped)
	}
	if stored := sink.Stored(); stored != 0 {
		t.Fatalf("stored = %d, want 0 - nothing was flushed", stored)
	}
}

// A sink with no instance id (or no database) must be inert rather than
// half-working: every shipped row needs an instance to be attributed to.
func TestInstanceLogSinkWithoutAnInstanceIsInert(t *testing.T) {
	sink := NewInstanceLogSink(nil, "")
	if _, err := sink.Write([]byte("line\n")); err != nil {
		t.Fatalf("inert sink returned %v, want nil", err)
	}
	if stored := sink.Flush(context.Background()); stored != 0 {
		t.Fatalf("inert sink stored %d rows, want 0", stored)
	}
	if dropped := sink.Dropped(); dropped != 0 {
		t.Fatalf("inert sink dropped %d lines, want 0", dropped)
	}
}

// Retention is age-only and bounded: shipped lines are a copy, so there is
// nothing to archive and no reason to delete in one unbounded statement.
func TestInstanceLogSinkRetentionPurgesOnlyExpiredRows(t *testing.T) {
	db := newInstanceLogTestDB(t)
	ctx := context.Background()
	sink := NewInstanceLogSink(db, "inst-a")

	// Truncated to microseconds: logged_at round-trips through Postgres, which
	// keeps microseconds, so a nanosecond-precision now() would never equal what
	// comes back out.
	fresh := time.Now().UTC().Truncate(time.Microsecond)
	old := fresh.AddDate(0, 0, -10)
	for _, at := range []time.Time{old.Add(-time.Minute), old, fresh} {
		if err := db.Create(&models.InstanceLogEntry{
			InstanceID: "inst-a", Source: "signaling.log", Seq: at.UnixNano(),
			LoggedAt: at, Line: "component=http event=request",
		}).Error; err != nil {
			t.Fatalf("seed row at %v: %v", at, err)
		}
	}
	removed, err := sink.PurgeBefore(ctx, fresh.AddDate(0, 0, -7))
	if err != nil {
		t.Fatalf("purge: %v", err)
	}
	if removed != 2 {
		t.Fatalf("purged %d rows, want the two outside the window", removed)
	}
	rows := loadInstanceLogRows(t, db)
	if len(rows) != 1 || !rows[0].LoggedAt.Equal(fresh) {
		t.Fatalf("remaining = %+v, want only the fresh row", rows)
	}
}
