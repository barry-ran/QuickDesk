package service

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"
)

func writeFile(t *testing.T, dir, name string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func remainingFiles(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	return names
}

// The sweep must delete only old, correctly named archive files: today plus the
// previous N-1 days are kept, and anything the operator put in the directory
// that this server does not recognise is never touched.
func TestSweepArchiveDirRemovesOnlyOldMatchingFiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "audit-logs-2026-10-16.jsonl") // today
	writeFile(t, dir, "audit-logs-2026-10-14.jsonl") // inside the window
	writeFile(t, dir, "audit-logs-2026-10-01.jsonl") // outside
	writeFile(t, dir, "audit-logs-2000-01-01.jsonl") // far outside
	writeFile(t, dir, "audit-logs-notadate.jsonl")   // unrecognised stamp
	writeFile(t, dir, "other-2026-10-01.jsonl")      // wrong prefix
	writeFile(t, dir, "README.md")                   // not an archive

	now := time.Date(2026, 10, 16, 7, 0, 0, 0, time.UTC)
	removed, err := sweepArchiveDir(dir, "audit-logs", 3, now)
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if removed != 2 {
		t.Fatalf("removed %d files, want the two out-of-window archives", removed)
	}
	want := []string{
		"README.md",
		"audit-logs-2026-10-14.jsonl",
		"audit-logs-2026-10-16.jsonl",
		"audit-logs-notadate.jsonl",
		"other-2026-10-01.jsonl",
	}
	if got := remainingFiles(t, dir); !equalStrings(got, want) {
		t.Fatalf("remaining = %v, want %v", got, want)
	}
}

// A retention of 1 keeps only today; the boundary is inclusive so the file for
// the cutoff day itself survives.
func TestSweepArchiveDirKeepsTheCutoffDay(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "audit-logs-2026-10-16.jsonl")
	writeFile(t, dir, "audit-logs-2026-10-15.jsonl")

	now := time.Date(2026, 10, 16, 23, 30, 0, 0, time.UTC)
	removed, err := sweepArchiveDir(dir, "audit-logs", 1, now)
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if removed != 1 {
		t.Fatalf("removed %d files, want exactly yesterday", removed)
	}
	if got, want := remainingFiles(t, dir), []string{"audit-logs-2026-10-16.jsonl"}; !equalStrings(got, want) {
		t.Fatalf("remaining = %v, want %v", got, want)
	}
}

// Disabled, or pointed at a directory that does not exist, must be a no-op
// rather than an error: the default is "keep the archives forever".
func TestSweepArchiveDirDisabledOrMissing(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "audit-logs-2000-01-01.jsonl")
	now := time.Date(2026, 10, 16, 7, 0, 0, 0, time.UTC)

	if removed, err := sweepArchiveDir(dir, "audit-logs", 0, now); err != nil || removed != 0 {
		t.Fatalf("disabled sweep = %d,%v, want 0,nil", removed, err)
	}
	if removed, err := sweepArchiveDir(filepath.Join(dir, "nope"), "audit-logs", 3, now); err != nil || removed != 0 {
		t.Fatalf("missing dir sweep = %d,%v, want 0,nil", removed, err)
	}
	if got := remainingFiles(t, dir); len(got) != 1 {
		t.Fatalf("remaining = %v, want the archive untouched", got)
	}
}

// Cold storage changes rotation from "delete the only copy" to "move it out of
// the hot directory"; the result must report both the total and the move.
func TestSweepArchiveDirMovesToColdStorage(t *testing.T) {
	hot := t.TempDir()
	cold := t.TempDir()
	writeFile(t, hot, "audit-logs-2026-10-16.jsonl")
	writeFile(t, hot, "audit-logs-2026-10-01.jsonl")
	now := time.Date(2026, 10, 16, 7, 0, 0, 0, time.UTC)

	result, err := sweepArchiveDirDetailed(hot, "audit-logs", 3, now, cold)
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if result.Removed != 1 || result.Moved != 1 || result.Deleted != 0 {
		t.Fatalf("result = %+v, want removed=1 moved=1 deleted=0", result)
	}
	if got, want := remainingFiles(t, hot), []string{"audit-logs-2026-10-16.jsonl"}; !equalStrings(got, want) {
		t.Fatalf("hot remaining = %v, want %v", got, want)
	}
	if got, want := remainingFiles(t, cold), []string{"audit-logs-2026-10-01.jsonl"}; !equalStrings(got, want) {
		t.Fatalf("cold files = %v, want %v", got, want)
	}
}

// A name collision in cold storage must not overwrite the existing copy; the
// moved file gets a suffix and the original cold file keeps its bytes.
func TestSweepArchiveDirColdCollisionGetsSuffix(t *testing.T) {
	hot := t.TempDir()
	cold := t.TempDir()
	const name = "audit-logs-2000-01-01.jsonl"
	writeFile(t, hot, name)
	if err := os.WriteFile(filepath.Join(cold, name), []byte("existing\n"), 0o644); err != nil {
		t.Fatalf("seed cold collision: %v", err)
	}

	result, err := sweepArchiveDirDetailed(hot, "audit-logs", 1,
		time.Date(2026, 10, 16, 7, 0, 0, 0, time.UTC), cold)
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if result.Moved != 1 || result.Removed != 1 {
		t.Fatalf("result = %+v, want one move", result)
	}
	got := remainingFiles(t, cold)
	want := []string{name, name + ".1"}
	if !equalStrings(got, want) {
		t.Fatalf("cold files = %v, want %v", got, want)
	}
	body, err := os.ReadFile(filepath.Join(cold, name))
	if err != nil {
		t.Fatalf("read existing cold file: %v", err)
	}
	if string(body) != "existing\n" {
		t.Fatalf("existing cold file was overwritten: %q", body)
	}
}

// If the cold directory cannot be created, the hot file must remain in place
// and the sweep must return an error instead of pretending the move happened.
func TestSweepArchiveDirColdFailureLeavesSource(t *testing.T) {
	hot := t.TempDir()
	blocker := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatalf("write blocker: %v", err)
	}
	writeFile(t, hot, "audit-logs-2000-01-01.jsonl")

	_, err := sweepArchiveDirDetailed(hot, "audit-logs", 1,
		time.Date(2026, 10, 16, 7, 0, 0, 0, time.UTC), filepath.Join(blocker, "cold"))
	if err == nil {
		t.Fatal("sweep reported success with an unusable cold directory")
	}
	if got, want := remainingFiles(t, hot), []string{"audit-logs-2000-01-01.jsonl"}; !equalStrings(got, want) {
		t.Fatalf("hot remaining = %v, want %v", got, want)
	}
}

// Pointing cold storage at the hot archive directory would turn a move into a
// no-op and then an irreversible delete; reject that configuration.
func TestSweepArchiveDirColdMustDiffer(t *testing.T) {
	hot := t.TempDir()
	writeFile(t, hot, "audit-logs-2000-01-01.jsonl")
	_, err := sweepArchiveDirDetailed(hot, "audit-logs", 1,
		time.Date(2026, 10, 16, 7, 0, 0, 0, time.UTC), hot)
	if err == nil {
		t.Fatal("sweep accepted the same directory for cold storage")
	}
	if got := remainingFiles(t, hot); len(got) != 1 {
		t.Fatalf("hot remaining = %v, want the source untouched", got)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
