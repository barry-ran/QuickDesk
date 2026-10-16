package service

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Daily archive files are the second half of retention: the table (or outbox)
// is bounded, and the bytes that left it live in `<dir>/<prefix>-YYYY-MM-DD.jsonl`.
// That directory only ever grew, so a long-lived deployment accumulated one
// file per day forever. This is the sweep for it.

// archiveFileLayout is the date stamp both archive writers put in their file
// names (`audit-logs-YYYY-MM-DD.jsonl`, `notification-deliveries-YYYY-MM-DD.jsonl`).
const archiveFileLayout = "2006-01-02"

// ArchiveSweepResult separates what happened to expired archive files. The
// historical API called all of them "removed"; now a deployment can move them
// into cold storage instead of deleting them, and the response must say which
// happened.
type ArchiveSweepResult struct {
	Removed int
	Moved   int
	Deleted int
}

// SweepArchiveDir deletes daily archive files older than retentionDays and
// reports how many it removed. Only files named `<prefix>-YYYY-MM-DD.jsonl` are
// candidates: anything else in the directory (a README, a hand-rotated copy, a
// temporary file) is left alone, because the archive directory belongs to the
// operator as much as to this server.
//
// retentionDays <= 0 disables it. That default matters: archiving exists so a
// purged row stays recoverable, so "keep the archives forever" is the safe
// default and deleting them has to be asked for explicitly.
//
// The boundary is inclusive of today: `retentionDays = 3` keeps today plus the
// two previous days and deletes anything older. A missing directory is not an
// error - a deployment that never archived simply has nothing to sweep.
func sweepArchiveDir(dir, prefix string, retentionDays int, now time.Time) (int, error) {
	result, err := sweepArchiveDirDetailed(dir, prefix, retentionDays, now, "")
	return result.Removed, err
}

// sweepArchiveDirDetailed is the same sweep with an optional cold directory.
// When coldDir is set, expired files are moved there (with a suffix if the
// destination name already exists) instead of being deleted; a failed move
// leaves the source file in place and stops the sweep with an error.
func sweepArchiveDirDetailed(dir, prefix string, retentionDays int, now time.Time, coldDir string) (ArchiveSweepResult, error) {
	var result ArchiveSweepResult
	if retentionDays <= 0 || strings.TrimSpace(dir) == "" {
		return result, nil
	}
	coldDir = strings.TrimSpace(coldDir)
	if coldDir != "" && sameArchivePath(dir, coldDir) {
		return result, fmt.Errorf("archive cold directory %q must differ from archive directory %q", coldDir, dir)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return result, nil
		}
		return result, err
	}

	// Midnight of the oldest day to keep. Comparing against a day boundary
	// rather than "now" avoids deleting a file whose date happens to equal the
	// current time-of-day subtraction.
	utcNow := now.UTC()
	keepFrom := time.Date(utcNow.Year(), utcNow.Month(), utcNow.Day(), 0, 0, 0, 0, time.UTC).
		AddDate(0, 0, -(retentionDays - 1))

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		date, ok := archiveFileDate(entry.Name(), prefix)
		if !ok || !date.Before(keepFrom) {
			continue
		}
		source := filepath.Join(dir, entry.Name())
		if coldDir == "" {
			if err := os.Remove(source); err != nil {
				return result, err
			}
			result.Deleted++
		} else {
			if err := moveArchiveToCold(source, coldDir, entry.Name()); err != nil {
				return result, err
			}
			result.Moved++
		}
		result.Removed++
	}
	return result, nil
}

// archiveFileDate extracts the day from one archive file name, reporting false
// for anything that does not match the expected `<prefix>-YYYY-MM-DD.jsonl`
// shape so unrecognised files are never deleted.
func archiveFileDate(name, prefix string) (time.Time, bool) {
	if !strings.HasPrefix(name, prefix+"-") || !strings.HasSuffix(name, ".jsonl") {
		return time.Time{}, false
	}
	stamp := strings.TrimSuffix(strings.TrimPrefix(name, prefix+"-"), ".jsonl")
	date, err := time.Parse(archiveFileLayout, stamp)
	if err != nil {
		return time.Time{}, false
	}
	return date, true
}

// sameArchivePath compares paths after making them absolute and, when both
// exist, resolving symlinks. A same-directory cold target would make the move
// a no-op followed by a delete, which is exactly the unrecoverable behavior
// cold storage is meant to avoid.
func sameArchivePath(a, b string) bool {
	absA, errA := filepath.Abs(a)
	absB, errB := filepath.Abs(b)
	if errA != nil || errB != nil {
		return filepath.Clean(a) == filepath.Clean(b)
	}
	if resolved, err := filepath.EvalSymlinks(absA); err == nil {
		absA = resolved
	}
	if resolved, err := filepath.EvalSymlinks(absB); err == nil {
		absB = resolved
	}
	return filepath.Clean(absA) == filepath.Clean(absB)
}

// moveArchiveToCold moves one archive file into the cold directory. The
// destination gets a numeric suffix rather than overwriting an existing cold
// copy, so a retry after a partial failure can never destroy the only copy.
func moveArchiveToCold(source, coldDir, name string) error {
	if err := os.MkdirAll(coldDir, 0o755); err != nil {
		return err
	}
	target, err := uniqueArchiveTarget(coldDir, name)
	if err != nil {
		return err
	}
	if err := os.Rename(source, target); err == nil {
		return nil
	}
	// Rename fails across filesystems (EXDEV); fall back to a synced copy
	// followed by removing the source. Any copy or sync failure removes the
	// incomplete target and leaves the source untouched.
	return copyArchiveFile(source, target)
}

func uniqueArchiveTarget(dir, name string) (string, error) {
	for i := 0; ; i++ {
		candidate := filepath.Join(dir, name)
		if i > 0 {
			candidate = filepath.Join(dir, fmt.Sprintf("%s.%d", name, i))
		}
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate, nil
		} else if err != nil {
			return "", err
		}
	}
}

func copyArchiveFile(source, target string) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(target)
		return err
	}
	if err := out.Sync(); err != nil {
		out.Close()
		os.Remove(target)
		return err
	}
	if err := out.Close(); err != nil {
		os.Remove(target)
		return err
	}
	return os.Remove(source)
}
