package service

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func appendFile(t *testing.T, path, text string) {
	t.Helper()
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer file.Close()
	if _, err := file.WriteString(text); err != nil {
		t.Fatalf("append to %s: %v", path, err)
	}
}

// A tailer that replayed the whole file on attach would dump whatever history
// the database left on disk into the shared store on every restart, so it
// starts at the end and only picks up what is written afterwards.
func TestFileLogTailerStartsAtEndAndShipsNewLines(t *testing.T) {
	db := newInstanceLogTestDB(t)
	ctx := context.Background()
	sink := NewInstanceLogSink(db, "inst-tail").WithBuffer(64)

	path := filepath.Join(t.TempDir(), "postgres.log")
	history := "2026/10/16 08:00:00.000000 history nobody asked for\n"
	if err := os.WriteFile(path, []byte(history), 0o644); err != nil {
		t.Fatalf("seed log: %v", err)
	}

	tailer := NewFileLogTailer(sink, path, PostgresLogSourceName)
	if shipped, err := tailer.Poll(ctx); err != nil || shipped != 0 {
		t.Fatalf("first poll shipped %d lines (err %v), want 0", shipped, err)
	}

	line := "2026/10/16 08:00:01.000000 FATAL: the line an operator wants\n"
	appendFile(t, path, line)
	shipped, err := tailer.Poll(ctx)
	if err != nil {
		t.Fatalf("poll: %v", err)
	}
	if shipped != 1 {
		t.Fatalf("poll shipped %d lines, want 1", shipped)
	}
	// The offset is the cursor: polling again without a write must not duplicate.
	if shipped, err := tailer.Poll(ctx); err != nil || shipped != 0 {
		t.Fatalf("second poll shipped %d lines (err %v), want 0", shipped, err)
	}

	if stored := sink.Flush(ctx); stored != 1 {
		t.Fatalf("flush stored %d lines, want 1", stored)
	}
	rows := loadInstanceLogRows(t, db)
	if len(rows) != 1 {
		t.Fatalf("shipped rows = %d, want 1", len(rows))
	}
	if rows[0].Source != PostgresLogSourceName {
		t.Fatalf("source = %q, want %q", rows[0].Source, PostgresLogSourceName)
	}
	if !strings.Contains(rows[0].Line, "the line an operator wants") {
		t.Fatalf("line = %q, want the appended line", rows[0].Line)
	}
	// The sequence has to be derived from where the line started in the file,
	// not from a counter: that is what makes re-reading the same bytes a no-op.
	if want := fileLogTailerSeqScale + int64(len(history)); rows[0].Seq != want {
		t.Fatalf("seq = %d, want %d (generation 1 + the line's offset)", rows[0].Seq, want)
	}
}

// A line still being written must not be shipped in pieces: the offset stays at
// the start of the fragment until the newline arrives.
func TestFileLogTailerShipsWholeLinesOnly(t *testing.T) {
	db := newInstanceLogTestDB(t)
	ctx := context.Background()
	sink := NewInstanceLogSink(db, "inst-partial").WithBuffer(64)

	path := filepath.Join(t.TempDir(), "redis.log")
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatalf("create log: %v", err)
	}
	tailer := NewFileLogTailer(sink, path, RedisLogSourceName)
	if _, err := tailer.Poll(ctx); err != nil {
		t.Fatalf("attach poll: %v", err)
	}

	appendFile(t, path, "2026/10/16 09:00:00.000000 1:M half")
	if shipped, err := tailer.Poll(ctx); err != nil || shipped != 0 {
		t.Fatalf("poll of a partial line shipped %d (err %v), want 0", shipped, err)
	}

	appendFile(t, path, "way\n")
	if shipped, err := tailer.Poll(ctx); err != nil || shipped != 1 {
		t.Fatalf("poll after the newline shipped %d (err %v), want 1", shipped, err)
	}
	if stored := sink.Flush(ctx); stored != 1 {
		t.Fatalf("flush stored %d lines, want 1", stored)
	}
	rows := loadInstanceLogRows(t, db)
	if len(rows) != 1 || !strings.HasSuffix(rows[0].Line, "halfway") {
		t.Fatalf("rows = %+v, want one whole line ending in \"halfway\"", rows)
	}
}

// Rotation and truncation both restart the offsets, so a tailer that keys lines
// by offset alone would drop everything written after them as a duplicate.
func TestFileLogTailerSurvivesRotationAndTruncation(t *testing.T) {
	db := newInstanceLogTestDB(t)
	ctx := context.Background()
	sink := NewInstanceLogSink(db, "inst-rotate").WithBuffer(64)

	dir := t.TempDir()
	path := filepath.Join(dir, "postgres.log")
	if err := os.WriteFile(path, []byte("2026/10/16 10:00:00.000000 before\n"), 0o644); err != nil {
		t.Fatalf("seed log: %v", err)
	}
	tailer := NewFileLogTailer(sink, path, PostgresLogSourceName)
	if _, err := tailer.Poll(ctx); err != nil {
		t.Fatalf("attach poll: %v", err)
	}

	appendFile(t, path, "2026/10/16 10:00:01.000000 generation one\n")
	if shipped, err := tailer.Poll(ctx); err != nil || shipped != 1 {
		t.Fatalf("poll shipped %d (err %v), want 1", shipped, err)
	}

	// logrotate-style rotation: the old file is renamed away and a new one takes
	// the path over. The new file is already longer than the old offset, so only
	// the inode change can reveal it.
	if err := os.Rename(path, path+".1"); err != nil {
		t.Fatalf("rotate: %v", err)
	}
	if err := os.WriteFile(path, []byte("2026/10/16 10:00:02.000000 generation two, longer than the offset\n"), 0o644); err != nil {
		t.Fatalf("create rotated log: %v", err)
	}
	if shipped, err := tailer.Poll(ctx); err != nil || shipped != 1 {
		t.Fatalf("poll after rotation shipped %d (err %v), want 1", shipped, err)
	}

	// copytruncate-style rotation: the same file is emptied and written again.
	if err := os.Truncate(path, 0); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	appendFile(t, path, "2026/10/16 10:00:03.000000 generation three\n")
	if shipped, err := tailer.Poll(ctx); err != nil || shipped != 1 {
		t.Fatalf("poll after truncation shipped %d (err %v), want 1", shipped, err)
	}

	if got := tailer.Rotations(); got != 2 {
		t.Fatalf("rotations = %d, want 2", got)
	}
	if stored := sink.Flush(ctx); stored != 3 {
		t.Fatalf("flush stored %d lines, want 3", stored)
	}

	rows := loadInstanceLogRows(t, db)
	if len(rows) != 3 {
		t.Fatalf("shipped rows = %d, want 3 (no line may be lost to a reused offset)", len(rows))
	}
	seen := map[int64]bool{}
	for _, row := range rows {
		if seen[row.Seq] {
			t.Fatalf("sequence %d appears twice", row.Seq)
		}
		seen[row.Seq] = true
	}
}

// A configured path that does not exist yet is normal (the database may not
// have started), so it must be quiet rather than an error loop.
func TestFileLogTailerMissingFileIsNotAnError(t *testing.T) {
	db := newInstanceLogTestDB(t)
	ctx := context.Background()
	sink := NewInstanceLogSink(db, "inst-missing").WithBuffer(8)

	tailer := NewFileLogTailer(sink, filepath.Join(t.TempDir(), "absent.log"), RedisLogSourceName)
	shipped, err := tailer.Poll(ctx)
	if err != nil {
		t.Fatalf("poll of a missing file returned %v, want nil", err)
	}
	if shipped != 0 {
		t.Fatalf("poll of a missing file shipped %d lines, want 0", shipped)
	}
}
