package service

import (
	"context"
	"testing"
	"time"

	"quickdesk/signaling/internal/models"
)

// TestAggregatedLogQueryMergesInstances is the point of the whole feature: a
// search run on one instance must return lines another instance wrote.
func TestAggregatedLogQueryMergesInstances(t *testing.T) {
	db := newInstanceLogTestDB(t)
	ctx := context.Background()
	now := time.Now().UTC()

	seed := []models.InstanceLogEntry{
		{InstanceID: "inst-a", Source: "signaling.log", Seq: 1, LoggedAt: now.Add(-3 * time.Minute),
			Line: "component=http event=request status=404 path=/v1/from-a"},
		{InstanceID: "inst-a", Source: "signaling.log", Seq: 2, LoggedAt: now.Add(-2 * time.Minute),
			Line: "component=api_auth event=rejected reason=missing_key"},
		{InstanceID: "inst-b", Source: "signaling.log", Seq: 1, LoggedAt: now.Add(-time.Minute),
			Line: "component=http event=request status=404 path=/v1/from-b"},
	}
	for _, row := range seed {
		if err := db.Create(&row).Error; err != nil {
			t.Fatalf("seed shipped row: %v", err)
		}
	}

	svc := NewAggregatedLogQueryService(db)

	// A marker that only instance B ever wrote must be found when the search
	// runs against the shared store.
	onlyB, err := svc.Query(ctx, AggregatedLogQuery{Query: "path=/v1/from-b"})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(onlyB.Entries) != 1 || onlyB.Entries[0].Instance != "inst-b" {
		t.Fatalf("entries = %+v, want the single inst-b line", onlyB.Entries)
	}
	if onlyB.Entries[0].Source != "signaling.log" || onlyB.Entries[0].Timestamp == "" {
		t.Fatalf("entry = %+v, want source and timestamp labelled", onlyB.Entries[0])
	}
	if len(onlyB.Instances) != 2 || onlyB.Instances[0] != "inst-a" || onlyB.Instances[1] != "inst-b" {
		t.Fatalf("instances = %v, want both shippers listed", onlyB.Instances)
	}

	// Newest first, across instances.
	all, err := svc.Query(ctx, AggregatedLogQuery{Query: "component=http"})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(all.Entries) != 2 {
		t.Fatalf("entries = %+v, want both http lines", all.Entries)
	}
	if all.Entries[0].Instance != "inst-b" {
		t.Fatalf("first entry = %+v, want the newest (inst-b) first", all.Entries[0])
	}

	// component is a whole-token match, exactly like the file search: a search
	// for `http` must not surface `api_auth` lines.
	if got, err := svc.Query(ctx, AggregatedLogQuery{Component: "http"}); err != nil || len(got.Entries) != 2 {
		t.Fatalf("component=http entries = %+v (%v), want the 2 http lines", got.Entries, err)
	}
	if got, err := svc.Query(ctx, AggregatedLogQuery{Component: "api"}); err != nil || len(got.Entries) != 0 {
		t.Fatalf("component=api entries = %+v (%v), want none - api is not a whole token", got.Entries, err)
	}
	if got, err := svc.Query(ctx, AggregatedLogQuery{Component: "api_auth"}); err != nil || len(got.Entries) != 1 {
		t.Fatalf("component=api_auth entries = %+v (%v), want the api_auth line", got.Entries, err)
	}

	// Narrowing to one instance, and to one source.
	if got, err := svc.Query(ctx, AggregatedLogQuery{Instance: "inst-a"}); err != nil || len(got.Entries) != 2 {
		t.Fatalf("instance=inst-a entries = %+v (%v), want 2", got.Entries, err)
	}
	if got, err := svc.Query(ctx, AggregatedLogQuery{Source: "postgres.log"}); err != nil || len(got.Entries) != 0 {
		t.Fatalf("source=postgres.log entries = %+v (%v), want none", got.Entries, err)
	}

	// The limit reports truncation rather than pretending it saw everything.
	limited, err := svc.Query(ctx, AggregatedLogQuery{Query: "component=http", Limit: 1})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(limited.Entries) != 1 || !limited.Truncated {
		t.Fatalf("limited = %+v, want 1 entry and truncated=true", limited)
	}
}

// The file scanner does a literal substring match, so the aggregated path must
// too: a search for "100%" cannot be allowed to match every line.
func TestAggregatedLogQueryTreatsWildcardsLiterally(t *testing.T) {
	db := newInstanceLogTestDB(t)
	ctx := context.Background()
	now := time.Now().UTC()
	for _, row := range []models.InstanceLogEntry{
		{InstanceID: "inst-a", Source: "signaling.log", Seq: 1, LoggedAt: now, Line: "retention kept 100% of rows"},
		{InstanceID: "inst-a", Source: "signaling.log", Seq: 2, LoggedAt: now, Line: "retention kept 100 rows"},
	} {
		if err := db.Create(&row).Error; err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	svc := NewAggregatedLogQueryService(db)
	got, err := svc.Query(ctx, AggregatedLogQuery{Query: "100%"})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(got.Entries) != 1 || got.Entries[0].Line != "retention kept 100% of rows" {
		t.Fatalf("entries = %+v, want only the literal percent line", got.Entries)
	}
}

// Time bounds are inclusive on both ends, like the file search.
func TestAggregatedLogQueryBoundsByTime(t *testing.T) {
	db := newInstanceLogTestDB(t)
	ctx := context.Background()
	base := time.Date(2026, 10, 16, 8, 0, 0, 0, time.UTC)
	for i, at := range []time.Time{base.Add(-time.Minute), base, base.Add(time.Minute)} {
		row := models.InstanceLogEntry{
			InstanceID: "inst-a", Source: "signaling.log", Seq: int64(i + 1),
			LoggedAt: at, Line: "component=http event=request",
		}
		if err := db.Create(&row).Error; err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	svc := NewAggregatedLogQueryService(db)
	got, err := svc.Query(ctx, AggregatedLogQuery{Since: &base, Until: &base})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(got.Entries) != 1 {
		t.Fatalf("entries = %+v, want exactly the boundary line", got.Entries)
	}
}
