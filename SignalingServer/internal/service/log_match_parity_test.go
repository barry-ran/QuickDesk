package service

import (
	"context"
	"testing"
	"time"

	"quickdesk/signaling/internal/database"
	"quickdesk/signaling/internal/models"
)

// The local matcher is only worth having if it agrees with Postgres on the same
// lines and the same expressions. This seeds the shipped-log table, lets the
// aggregated service evaluate `websearch_to_tsquery('simple', …)` for every
// expression, and compares its per-line verdict with the Go matcher's.
func TestLogMatchParityWithPostgresFullText(t *testing.T) {
	db := newInstanceLogTestDB(t)
	ctx := context.Background()
	database.EnsureLogSearchIndexes(ctx, db)

	lines := []string{
		`component=dial event=failed error="connection refused by peer"`,
		`component=dial event=failed error="connection reset by peer"`,
		`component=dial event=closed peer connection dropped`,
		`component=auth event=ok request_id=abc-123 v1.2 12.5ms`,
		`component=auth event=ok error="api_auth rejected"`,
		`component=log rotated_at=2026-10-16T12:00:00Z`,
	}
	now := time.Now().UTC()
	for i, line := range lines {
		row := models.InstanceLogEntry{
			InstanceID: "inst-parity",
			Source:     "signaling.log",
			Seq:        int64(i + 1),
			LoggedAt:   now.Add(time.Duration(i) * time.Second),
			Line:       line,
		}
		if err := db.Create(&row).Error; err != nil {
			t.Fatalf("seed row %d: %v", i, err)
		}
	}
	aggregated := NewAggregatedLogQueryService(db)

	expressions := []string{
		`"connection refused"`,
		"connection refused",
		"connection -refused",
		"refused OR reset",
		"connection or",
		"api_auth",
		"-api_auth",
		"abc-123",
		"v1.2",
		"12.5",
		"t12",
		"00z",
		"2026",
		"nonexistent-marker",
		"-",
	}
	for _, expr := range expressions {
		t.Run(expr, func(t *testing.T) {
			got, err := aggregated.Query(ctx, AggregatedLogQuery{Match: expr, Limit: 100})
			if err != nil {
				t.Fatalf("aggregated query: %v", err)
			}
			postgres := make(map[string]bool, len(got.Entries))
			for _, entry := range got.Entries {
				postgres[entry.Line] = true
			}
			query := parseLogMatch(expr)
			for _, line := range lines {
				if match := query.matches(line); match != postgres[line] {
					t.Errorf("expression %q: local matcher says %v for %q, Postgres says %v",
						expr, match, line, postgres[line])
				}
			}
		})
	}
}
