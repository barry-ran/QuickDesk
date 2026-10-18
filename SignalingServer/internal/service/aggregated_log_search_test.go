package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"quickdesk/signaling/internal/database"
	"quickdesk/signaling/internal/models"

	"gorm.io/gorm"
)

// The substring search (`q=`) and the full-text search (`match=`) have to stay
// different tools: `q` finds a fragment anywhere in a line, `match` is for
// words and phrases. A search for a phrase must not match a line that contains
// the words apart, which is exactly what the substring mode does.
func TestAggregatedLogQueryMatchUsesFullText(t *testing.T) {
	db := newInstanceLogTestDB(t)
	ctx := context.Background()
	database.EnsureLogSearchIndexes(ctx, db)
	now := time.Now().UTC()

	seed := []models.InstanceLogEntry{
		{InstanceID: "inst-a", Source: "signaling.log", Seq: 1, LoggedAt: now.Add(-3 * time.Minute),
			Line: "component=dial event=failed error=\"connection refused by peer\""},
		{InstanceID: "inst-a", Source: "signaling.log", Seq: 2, LoggedAt: now.Add(-2 * time.Minute),
			Line: "component=dial event=failed error=\"connection reset by peer\""},
		{InstanceID: "inst-b", Source: "signaling.log", Seq: 1, LoggedAt: now.Add(-time.Minute),
			Line: "component=dial event=closed peer connection dropped"},
	}
	for _, row := range seed {
		if err := db.Create(&row).Error; err != nil {
			t.Fatalf("seed shipped row: %v", err)
		}
	}
	svc := NewAggregatedLogQueryService(db)

	cases := []struct {
		name  string
		match string
		want  int
	}{
		{name: "quoted phrase", match: `"connection refused"`, want: 1},
		{name: "two words are ANDed", match: "connection refused", want: 1},
		{name: "negation", match: "connection -refused", want: 2},
		{name: "OR", match: "refused OR reset", want: 2},
		{name: "no match", match: "nonexistent-marker", want: 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := svc.Query(ctx, AggregatedLogQuery{Match: tc.match})
			if err != nil {
				t.Fatalf("query: %v", err)
			}
			if len(got.Entries) != tc.want {
				t.Fatalf("match %q returned %d lines, want %d (%+v)", tc.match, len(got.Entries), tc.want, got.Entries)
			}
			if got.Match != tc.match {
				t.Fatalf("result echoed match %q, want %q", got.Match, tc.match)
			}
		})
	}

	// The substring mode keeps its semantics: the same words in a different
	// order still match, because that is what substring search means.
	fragment, err := svc.Query(ctx, AggregatedLogQuery{Query: "connection refused"})
	if err != nil {
		t.Fatalf("substring query: %v", err)
	}
	if len(fragment.Entries) != 1 {
		t.Fatalf("substring query returned %d lines, want 1", len(fragment.Entries))
	}
	if fragment.Match != "" {
		t.Fatalf("a substring-only query reported match=%q", fragment.Match)
	}
	// And a fragment that is not a whole word is still findable, which full text
	// alone could not do.
	if got, err := svc.Query(ctx, AggregatedLogQuery{Query: "onnection refu"}); err != nil || len(got.Entries) != 1 {
		t.Fatalf("partial-fragment query = %d lines (err %v), want 1", len(got.Entries), err)
	}
}

// The indexes are what make the search usable on a table that grows with the
// fleet's log volume, so they are asserted to exist and to be usable by the
// planner rather than merely declared in a file.
func TestAggregatedLogSearchIndexesAreUsable(t *testing.T) {
	db := newInstanceLogTestDB(t)
	ctx := context.Background()
	// Twice on purpose: startup calls this every boot, so it has to be
	// idempotent or a restart would fail on the second CREATE.
	database.EnsureLogSearchIndexes(ctx, db)
	database.EnsureLogSearchIndexes(ctx, db)

	for _, index := range []string{
		"idx_instance_log_entries_line_trgm",
		"idx_instance_log_entries_line_tsv",
	} {
		var count int64
		if err := db.Raw(
			`SELECT count(*) FROM pg_indexes WHERE tablename = 'instance_log_entries' AND indexname = ?`,
			index).Scan(&count).Error; err != nil {
			t.Fatalf("look up index %s: %v", index, err)
		}
		if count != 1 {
			t.Fatalf("index %s is missing", index)
		}
	}

	if err := db.Create(&models.InstanceLogEntry{
		InstanceID: "inst-idx", Source: "signaling.log", Seq: 1, LoggedAt: time.Now().UTC(),
		Line: "component=dial event=failed error=\"connection refused by peer\"",
	}).Error; err != nil {
		t.Fatalf("seed row: %v", err)
	}

	// enable_seqscan=off only makes the planner *prefer* an index; if no index
	// could satisfy the predicate it would still choose a sequential scan, so
	// seeing the index name in the plan is the evidence that it is usable.
	plans := map[string]string{
		"substring": explainInTx(t, db, `SELECT id FROM instance_log_entries WHERE line ILIKE '%refused by peer%'`),
		"full-text": explainInTx(t, db, `SELECT id FROM instance_log_entries WHERE line_tsv @@ websearch_to_tsquery('simple', 'refused')`),
	}
	if !strings.Contains(plans["substring"], "idx_instance_log_entries_line_trgm") {
		t.Fatalf("substring plan does not use the trigram index:\n%s", plans["substring"])
	}
	if !strings.Contains(plans["full-text"], "idx_instance_log_entries_line_tsv") {
		t.Fatalf("full-text plan does not use the GIN index:\n%s", plans["full-text"])
	}
}

// explainInTx runs EXPLAIN with sequential scans disabled, inside the
// transaction that needs SET LOCAL.
func explainInTx(t *testing.T, db *gorm.DB, sql string) string {
	t.Helper()
	var plan strings.Builder
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SET LOCAL enable_seqscan = off").Error; err != nil {
			return err
		}
		rows, err := tx.Raw("EXPLAIN " + sql).Rows()
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var line string
			if err := rows.Scan(&line); err != nil {
				return err
			}
			plan.WriteString(line)
			plan.WriteString("\n")
		}
		return rows.Err()
	})
	if err != nil {
		t.Fatalf("explain: %v", err)
	}
	return plan.String()
}
