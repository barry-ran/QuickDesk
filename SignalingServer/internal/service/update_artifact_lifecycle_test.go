package service

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

// Re-hosting means installers live on this server's disk, so an abandoned
// resumable upload is real space nobody else reclaims: the upload only cleans up
// when it finishes, and the release row only cleans up when it is deleted. The
// pass has two rules and has to keep them apart - age decides a staging file, a
// missing row decides an orphan - and both have to leave a live artifact alone.
func TestUpdateArtifactLifecycleSweepsStaleUploadsAndOrphans(t *testing.T) {
	db := newUpdateReleaseTestDB(t)
	svc := NewUpdateReleaseService(db)
	ctx := context.Background()
	dir := t.TempDir()

	old := time.Now().Add(-48 * time.Hour)

	// A release whose artifact the row still claims: it must survive even though
	// it is older than the orphan rule's minimum age.
	live, err := svc.Publish(ctx, sampleRelease("4.3.0.0"))
	if err != nil {
		t.Fatalf("publish live release: %v", err)
	}
	liveDir := filepath.Join(dir, strconv.FormatUint(uint64(live.ID), 10))
	if err := os.MkdirAll(liveDir, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	livePath := filepath.Join(liveDir, "QuickDesk-4.3.0.0.exe")
	liveBytes := bytes.Repeat([]byte("L"), 4096)
	if err := os.WriteFile(livePath, liveBytes, 0o640); err != nil {
		t.Fatalf("write live artifact: %v", err)
	}
	if _, err := svc.SetArtifact(ctx, live.ID, livePath, live.SHA256, "http://example.test/artifact", int64(len(liveBytes))); err != nil {
		t.Fatalf("set artifact: %v", err)
	}
	mustAge(t, livePath, old)

	// A staging file for that same release, abandoned long ago.
	stalePart := filepath.Join(liveDir, "QuickDesk-4.3.0.0-old.exe.part")
	if err := os.WriteFile(stalePart, bytes.Repeat([]byte("S"), 2048), 0o640); err != nil {
		t.Fatalf("write stale staging file: %v", err)
	}
	mustAge(t, stalePart, old)

	// A staging file touched a moment ago: an upload may still be running under
	// it, so the pass has to leave it alone.
	freshPart := filepath.Join(liveDir, "QuickDesk-4.3.0.0-new.exe.part")
	if err := os.WriteFile(freshPart, bytes.Repeat([]byte("F"), 1024), 0o640); err != nil {
		t.Fatalf("write fresh staging file: %v", err)
	}

	// An artifact whose release row is gone. Deleting through the service leaves
	// the bytes behind on purpose here, which is exactly the crash-shaped case:
	// a rename that happened, a database write that did not.
	orphan, err := svc.Publish(ctx, sampleRelease("4.3.1.0"))
	if err != nil {
		t.Fatalf("publish orphaned release: %v", err)
	}
	orphanDir := filepath.Join(dir, strconv.FormatUint(uint64(orphan.ID), 10))
	if err := os.MkdirAll(orphanDir, 0o750); err != nil {
		t.Fatalf("mkdir orphan dir: %v", err)
	}
	orphanPath := filepath.Join(orphanDir, "QuickDesk-4.3.1.0.exe")
	if err := os.WriteFile(orphanPath, bytes.Repeat([]byte("O"), 8192), 0o640); err != nil {
		t.Fatalf("write orphan: %v", err)
	}
	mustAge(t, orphanPath, old)
	if _, err := svc.Delete(ctx, orphan.ID); err != nil {
		t.Fatalf("delete orphan row: %v", err)
	}

	lifecycle := NewUpdateArtifactLifecycle(svc, dir, 6*time.Hour, time.Hour)
	result, err := lifecycle.SweepOnce(ctx)
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if result.UploadsReaped != 1 || result.BytesReaped != 2048 {
		t.Fatalf("reaped %d uploads / %d bytes, want the 1 x 2048 stale staging file",
			result.UploadsReaped, result.BytesReaped)
	}
	if result.OrphansRemoved != 1 || result.OrphanBytes != 8192 {
		t.Fatalf("removed %d orphans / %d bytes, want the 1 x 8192 unclaimed artifact",
			result.OrphansRemoved, result.OrphanBytes)
	}

	if _, err := os.Stat(stalePart); !os.IsNotExist(err) {
		t.Fatalf("the stale staging file survived: %v", err)
	}
	if _, err := os.Stat(orphanPath); !os.IsNotExist(err) {
		t.Fatalf("the orphaned artifact survived: %v", err)
	}
	if _, err := os.Stat(freshPart); err != nil {
		t.Fatalf("a staging file an upload may still be writing was reaped: %v", err)
	}
	if _, err := os.Stat(livePath); err != nil {
		t.Fatalf("the claimed artifact was reaped: %v", err)
	}
	if _, err := os.Stat(orphanDir); !os.IsNotExist(err) {
		t.Fatalf("an emptied release directory survived: %v", err)
	}

	// A second pass has nothing left to do: the rules are idempotent.
	again, err := lifecycle.SweepOnce(ctx)
	if err != nil {
		t.Fatalf("second sweep: %v", err)
	}
	if again.UploadsReaped != 0 || again.OrphansRemoved != 0 {
		t.Fatalf("a second pass changed something: %+v", again)
	}
}

// The age window is the whole safety property: a staging file younger than the
// window is an upload in progress, and deleting it would destroy bytes the
// client is about to resume from.
func TestUpdateArtifactLifecycleKeepsEverythingWithinTheWindow(t *testing.T) {
	db := newUpdateReleaseTestDB(t)
	svc := NewUpdateReleaseService(db)
	ctx := context.Background()
	dir := t.TempDir()

	release, err := svc.Publish(ctx, sampleRelease("4.3.2.0"))
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	releaseDir := filepath.Join(dir, strconv.FormatUint(uint64(release.ID), 10))
	if err := os.MkdirAll(releaseDir, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	part := filepath.Join(releaseDir, "QuickDesk-4.3.2.0.exe.part")
	if err := os.WriteFile(part, bytes.Repeat([]byte("P"), 512), 0o640); err != nil {
		t.Fatalf("write staging file: %v", err)
	}
	mustAge(t, part, time.Now().Add(-time.Hour))

	lifecycle := NewUpdateArtifactLifecycle(svc, dir, 6*time.Hour, time.Hour)
	result, err := lifecycle.SweepOnce(ctx)
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if result.UploadsReaped != 0 {
		t.Fatalf("reaped %d staging files inside the window, want none", result.UploadsReaped)
	}
	if _, err := os.Stat(part); err != nil {
		t.Fatalf("a staging file inside the window was removed: %v", err)
	}
}

func mustAge(t *testing.T, path string, when time.Time) {
	t.Helper()
	if err := os.Chtimes(path, when, when); err != nil {
		t.Fatalf("age %s: %v", path, err)
	}
}
