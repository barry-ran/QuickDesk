package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// writeLogFile drops a synthetic log file into dir. Lines are written verbatim
// so a test can control exactly what the reverse scanner sees.
func writeLogFile(t *testing.T, dir, name string, lines []string) {
	t.Helper()
	body := strings.Join(lines, "\n")
	if body != "" {
		body += "\n"
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

// stamp renders the prefix the standard logger writes.
func stamp(at time.Time) string {
	return at.UTC().Format("2006/01/02 15:04:05.000000")
}

func sampleLogDir(t *testing.T) (string, time.Time) {
	t.Helper()
	dir := t.TempDir()
	base := time.Date(2026, 10, 16, 10, 0, 0, 0, time.UTC)
	writeLogFile(t, dir, "signaling.log", []string{
		stamp(base) + " component=api_auth event=request path=/v1/auth/sessions",
		stamp(base.Add(time.Second)) + " component=api event=other plain line",
		stamp(base.Add(5*time.Minute)) + " component=notification event=enqueue channel=7",
		stamp(base.Add(10*time.Minute)) + " a line without any component field",
	})
	// .1 is the most recent rotation, i.e. older than signaling.log.
	writeLogFile(t, dir, "signaling.log.1", []string{
		stamp(base.Add(-time.Hour)) + " component=api_auth event=old",
	})
	return dir, base
}

func lineTexts(entries []LogEntry) []string {
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, e.Line)
	}
	return out
}

// Newest first is the whole point of a reverse scan, and the order has to
// survive crossing into a rotated file.
func TestLogQueryReturnsNewestFirstAcrossRotations(t *testing.T) {
	dir, _ := sampleLogDir(t)
	svc := NewLogQueryService(dir)

	res, err := svc.Query(context.Background(), LogQuery{})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(res.Entries) != 5 {
		t.Fatalf("got %d lines, want all 5 across both files: %v", len(res.Entries), lineTexts(res.Entries))
	}
	wantSubstrings := []string{
		"a line without any component field",
		"component=notification",
		"component=api event=other",
		"component=api_auth event=request",
		"component=api_auth event=old",
	}
	for i, want := range wantSubstrings {
		if !strings.Contains(res.Entries[i].Line, want) {
			t.Fatalf("entry %d = %q, want it to contain %q", i, res.Entries[i].Line, want)
		}
	}
	if len(res.Files) != 2 || res.Files[0].Name != "signaling.log" || res.Files[1].Name != "signaling.log.1" {
		t.Fatalf("files = %+v, want the current log first then .1", res.Files)
	}
	if res.Entries[0].Timestamp == "" {
		t.Fatal("entries should carry a parsed timestamp")
	}
	// The limit is a page size, not a truncation of the search.
	page, err := svc.Query(context.Background(), LogQuery{Limit: 2})
	if err != nil {
		t.Fatalf("paged query: %v", err)
	}
	if len(page.Entries) != 2 || page.Truncated {
		t.Fatalf("limit 2 returned %d entries (truncated=%t), want 2 and not truncated", len(page.Entries), page.Truncated)
	}
}

// `q` is a substring match; `component` is a whole token. The difference
// matters: filtering component=api must not drag in component=api_auth.
func TestLogQueryFiltersSubstringAndComponentToken(t *testing.T) {
	dir, _ := sampleLogDir(t)
	svc := NewLogQueryService(dir)
	ctx := context.Background()

	res, err := svc.Query(ctx, LogQuery{Query: "NOTIFICATION"})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(res.Entries) != 1 || !strings.Contains(res.Entries[0].Line, "component=notification") {
		t.Fatalf("case-insensitive substring match got %v", lineTexts(res.Entries))
	}

	res, err = svc.Query(ctx, LogQuery{Component: "api"})
	if err != nil {
		t.Fatalf("component query: %v", err)
	}
	if len(res.Entries) != 1 {
		t.Fatalf("component=api matched %d lines, want 1 (api_auth is a different component): %v",
			len(res.Entries), lineTexts(res.Entries))
	}
	if !strings.Contains(res.Entries[0].Line, "component=api event=other") {
		t.Fatalf("component=api matched the wrong line: %q", res.Entries[0].Line)
	}

	res, err = svc.Query(ctx, LogQuery{Component: "api_auth"})
	if err != nil {
		t.Fatalf("component query: %v", err)
	}
	if len(res.Entries) != 2 {
		t.Fatalf("component=api_auth matched %d lines, want 2", len(res.Entries))
	}
}

// The time bounds are inclusive and the window is read from the line's own
// timestamp, so a boundary line is never lost to an off-by-one.
func TestLogQueryTimeWindowIsInclusive(t *testing.T) {
	dir, base := sampleLogDir(t)
	svc := NewLogQueryService(dir)
	ctx := context.Background()

	at := base.Add(5 * time.Minute)
	res, err := svc.Query(ctx, LogQuery{Since: &at, Until: &at})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(res.Entries) != 1 || !strings.Contains(res.Entries[0].Line, "component=notification") {
		t.Fatalf("exact-boundary window returned %v, want the 10:05 line", lineTexts(res.Entries))
	}

	since := base.Add(5 * time.Minute)
	res, err = svc.Query(ctx, LogQuery{Since: &since})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(res.Entries) != 2 {
		t.Fatalf("since=10:05 returned %d lines, want 2", len(res.Entries))
	}

	until := base.Add(-time.Hour)
	res, err = svc.Query(ctx, LogQuery{Until: &until})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(res.Entries) != 1 || !strings.Contains(res.Entries[0].Line, "event=old") {
		t.Fatalf("until=09:00 returned %v, want only the rotated line", lineTexts(res.Entries))
	}
}

// The scanner reads fixed-size blocks from the end, so a line that straddles a
// block boundary has to be reassembled rather than handed out in pieces. This
// writes a file several blocks long and checks the count and the oldest line.
func TestLogQueryReassemblesLinesAcrossBlocks(t *testing.T) {
	dir := t.TempDir()
	base := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	const total = 1500
	lines := make([]string, 0, total)
	for i := 1; i <= total; i++ {
		lines = append(lines, fmt.Sprintf("%s component=load event=beat seq=%04d payload=%s",
			stamp(base.Add(time.Duration(i)*time.Second)), i, strings.Repeat("x", 30)))
	}
	writeLogFile(t, dir, "signaling.log", lines)
	info, err := os.Stat(filepath.Join(dir, "signaling.log"))
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Size() <= logBlockSize {
		t.Fatalf("fixture is only %d bytes, it has to exceed one %d-byte block", info.Size(), logBlockSize)
	}

	svc := NewLogQueryService(dir)
	res, err := svc.Query(context.Background(), LogQuery{Limit: total})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(res.Entries) != total {
		t.Fatalf("got %d lines, want %d", len(res.Entries), total)
	}
	if res.Truncated {
		t.Fatal("a full read of the file must not report truncation")
	}
	if !strings.Contains(res.Entries[0].Line, "seq=1500") {
		t.Fatalf("newest line is %q, want seq=1500", res.Entries[0].Line)
	}
	if !strings.Contains(res.Entries[total-1].Line, "seq=0001") {
		t.Fatalf("oldest line is %q, want seq=0001", res.Entries[total-1].Line)
	}
	// Every line has to come back whole; a block-boundary bug shows up as a
	// fragment with no timestamp.
	for i, entry := range res.Entries {
		if entry.Timestamp == "" {
			t.Fatalf("entry %d is not a whole line: %q", i, entry.Line)
		}
	}
}

// Running out of byte budget must be reported, because an empty result from a
// truncated scan is not evidence that the event never happened.
func TestLogQueryReportsTruncatedScan(t *testing.T) {
	dir := t.TempDir()
	base := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	lines := make([]string, 0, 1200)
	for i := 1; i <= 1200; i++ {
		lines = append(lines, fmt.Sprintf("%s component=load event=beat seq=%04d", stamp(base), i))
	}
	writeLogFile(t, dir, "signaling.log", lines)

	svc := NewLogQueryService(dir)
	svc.scanBudget = 32 * 1024 // smaller than one block, so the walk stops early
	res, err := svc.Query(context.Background(), LogQuery{Limit: 5000})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if !res.Truncated {
		t.Fatalf("scan stopped at %d bytes but truncated=false (scanned %d lines)",
			res.ScannedBytes, res.ScannedLines)
	}
	if res.ScannedBytes > svc.scanBudget {
		t.Fatalf("scanned %d bytes, over the %d-byte budget", res.ScannedBytes, svc.scanBudget)
	}

	// A missing log is not an error: a server that has not written yet simply
	// has nothing to show.
	empty := NewLogQueryService(t.TempDir())
	res, err = empty.Query(context.Background(), LogQuery{})
	if err != nil {
		t.Fatalf("missing log: %v", err)
	}
	if len(res.Entries) != 0 || len(res.Files) != 0 {
		t.Fatalf("missing log returned %+v, want empty", res)
	}
}

// The search endpoint used to scan only the application log directory, so a
// configured PostgreSQL or Redis log could be listed and downloaded but not
// queried. A registered extra source has to be selectable by name, and the
// default must stay the application log.
func TestLogQuerySearchesRegisteredExtraSource(t *testing.T) {
	dir, base := sampleLogDir(t)
	pgPath := filepath.Join(t.TempDir(), "postgres.log")
	writeLogFile(t, filepath.Dir(pgPath), "postgres.log", []string{
		stamp(base) + " FATAL: pg-marker-1 could not connect",
		stamp(base.Add(time.Second)) + " LOG: checkpoint complete pg-marker-2",
	})

	svc := NewLogQueryService(dir).AddFile("postgres.log", pgPath)

	// Default (no file=) still reads the application log.
	app, err := svc.Query(context.Background(), LogQuery{Query: "component=notification"})
	if err != nil {
		t.Fatalf("app query: %v", err)
	}
	if len(app.Entries) != 1 {
		t.Fatalf("default query returned %d entries, want the one app-log match", len(app.Entries))
	}
	if app.Files[0].Name != "signaling.log" {
		t.Fatalf("default query scanned %q, want signaling.log", app.Files[0].Name)
	}

	// The extra source is selectable and reports its own logical name.
	pg, err := svc.Query(context.Background(), LogQuery{Query: "pg-marker", File: "postgres.log"})
	if err != nil {
		t.Fatalf("pg query: %v", err)
	}
	if len(pg.Entries) != 2 {
		t.Fatalf("postgres.log query returned %d entries, want 2", len(pg.Entries))
	}
	if len(pg.Files) != 1 || pg.Files[0].Name != "postgres.log" {
		t.Fatalf("postgres.log query reported files %+v, want the logical name", pg.Files)
	}
	// An unknown name must not silently fall back to another file.
	unknown, err := svc.Query(context.Background(), LogQuery{File: "nope.log"})
	if err != nil {
		t.Fatalf("unknown source: %v", err)
	}
	if len(unknown.Entries) != 0 || len(unknown.Files) != 0 {
		t.Fatalf("unknown source returned %+v, want empty", unknown)
	}
}

// Rotation handling has to work for extra sources too: a PostgreSQL log
// rotated to postgres.log.1 is exactly the file an operator wants when asking
// "what happened before the restart".
func TestLogQueryFollowsExtraSourceRotation(t *testing.T) {
	dir, base := sampleLogDir(t)
	pgDir := t.TempDir()
	writeLogFile(t, pgDir, "postgres.log", []string{
		stamp(base) + " LOG: current pg-current",
	})
	writeLogFile(t, pgDir, "postgres.log.1", []string{
		stamp(base.Add(-time.Hour)) + " FATAL: rotated pg-rotated",
	})

	svc := NewLogQueryService(dir).AddFile("postgres.log", filepath.Join(pgDir, "postgres.log"))
	res, err := svc.Query(context.Background(), LogQuery{Query: "pg-", File: "postgres.log"})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(res.Entries) != 2 {
		t.Fatalf("got %d entries, want both the current and rotated line: %+v", len(res.Entries), lineTexts(res.Entries))
	}
	// Newest first across the rotation.
	if !strings.Contains(res.Entries[0].Line, "pg-current") {
		t.Fatalf("first entry = %q, want the newest (current file) line", res.Entries[0].Line)
	}
	if len(res.Files) != 2 || res.Files[1].Name != "postgres.log.1" {
		t.Fatalf("files = %+v, want postgres.log then postgres.log.1", res.Files)
	}

	// Sources() reports registration order so the API can document what is
	// searchable.
	names := make([]string, 0, len(svc.Sources()))
	for _, s := range svc.Sources() {
		names = append(names, s.Name)
	}
	if strings.Join(names, ",") != "signaling.log,postgres.log" {
		t.Fatalf("Sources() = %v, want registration order with the app log first", names)
	}
}

// `file=all` searches every configured source in one pass and labels each hit
// with the source it came from, so an operator does not have to guess which log
// to look at first during an incident.
func TestLogQueryAllSourcesScansEveryRegisteredSource(t *testing.T) {
	dir := t.TempDir()
	base := time.Date(2026, 10, 16, 10, 0, 0, 0, time.UTC)
	writeLogFile(t, dir, "signaling.log", []string{
		stamp(base) + " component=api shared-marker from the app log",
	})
	pgDir := t.TempDir()
	pgPath := filepath.Join(pgDir, "postgres.log")
	writeLogFile(t, pgDir, "postgres.log", []string{
		stamp(base.Add(time.Second)) + " LOG: shared-marker from the database log",
	})
	redisDir := t.TempDir()
	redisPath := filepath.Join(redisDir, "redis.log")
	writeLogFile(t, redisDir, "redis.log", []string{
		stamp(base.Add(2*time.Second)) + " notice shared-marker from redis",
	})

	svc := NewLogQueryService(dir).
		AddFile("postgres.log", pgPath).
		AddFile("redis.log", redisPath)

	all, err := svc.Query(context.Background(), LogQuery{Query: "shared-marker", File: AllLogSources})
	if err != nil {
		t.Fatalf("all-source query: %v", err)
	}
	if len(all.Entries) != 3 {
		t.Fatalf("all-source query returned %d entries, want one per source: %+v", len(all.Entries), all.Entries)
	}
	seen := map[string]int{}
	for _, e := range all.Entries {
		seen[e.File]++
	}
	for _, name := range []string{"signaling.log", "postgres.log", "redis.log"} {
		if seen[name] != 1 {
			t.Fatalf("source %q appeared %d times, want once (entries=%+v)", name, seen[name], all.Entries)
		}
	}
	if len(all.Files) != 3 {
		t.Fatalf("all-source query reported %d files, want the three logged sources", len(all.Files))
	}

	// A single-source query still labels its entries, so the field is safe to
	// rely on whichever form the caller uses.
	one, err := svc.Query(context.Background(), LogQuery{Query: "shared-marker", File: "redis.log"})
	if err != nil {
		t.Fatalf("redis query: %v", err)
	}
	if len(one.Entries) != 1 || one.Entries[0].File != "redis.log" {
		t.Fatalf("redis query entries = %+v, want one labelled redis.log", one.Entries)
	}

	// The budget is shared across sources: a tiny budget must bound the whole
	// scan, not be handed out per file.
	tiny := NewLogQueryService(dir).
		AddFile("postgres.log", pgPath).
		AddFile("redis.log", redisPath)
	tiny.scanBudget = 8
	limited, err := tiny.Query(context.Background(), LogQuery{File: AllLogSources})
	if err != nil {
		t.Fatalf("tiny budget query: %v", err)
	}
	if !limited.Truncated {
		t.Fatal("an 8-byte budget across all sources did not report truncated")
	}
	if limited.ScannedBytes > 8 {
		t.Fatalf("scanned %d bytes with an 8-byte budget, want the budget to bound the whole scan", limited.ScannedBytes)
	}
}

// The file scanner answers the same `match=` expression the aggregated store
// hands to Postgres. The counts here are the same ones the aggregated test
// pins, so a change that makes one backend diverge is caught by the other.
func TestLogQueryFullTextMatch(t *testing.T) {
	dir := t.TempDir()
	writeLogFile(t, dir, "signaling.log", []string{
		"component=dial event=failed error=\"connection refused by peer\"",
		"component=dial event=failed error=\"connection reset by peer\"",
		"component=dial event=closed peer connection dropped",
	})
	svc := NewLogQueryService(dir)

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
			res, err := svc.Query(context.Background(), LogQuery{Match: tc.match})
			if err != nil {
				t.Fatalf("query: %v", err)
			}
			if len(res.Entries) != tc.want {
				t.Fatalf("match %q returned %d lines, want %d (%v)",
					tc.match, len(res.Entries), tc.want, lineTexts(res.Entries))
			}
			if res.Match != tc.match {
				t.Fatalf("result echoed match %q, want %q", res.Match, tc.match)
			}
		})
	}

	// The substring mode keeps its own semantics: it finds the phrase inside
	// the line, which is what substring search means, and it does not claim to
	// have applied full text.
	fragment, err := svc.Query(context.Background(), LogQuery{Query: "connection refused"})
	if err != nil {
		t.Fatalf("substring query: %v", err)
	}
	if len(fragment.Entries) != 1 {
		t.Fatalf("substring query returned %d lines, want the one that contains the phrase",
			len(fragment.Entries))
	}
	if fragment.Match != "" {
		t.Fatalf("a substring-only query reported match=%q", fragment.Match)
	}
}
