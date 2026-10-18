package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"quickdesk/signaling/internal/models"
)

func TestArtifactExpired(t *testing.T) {
	now := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	cutoff := now.Add(-24 * time.Hour)
	old := now.Add(-48 * time.Hour)
	recent := now.Add(-1 * time.Hour)

	withArtifact := func(stamp time.Time) models.SessionRecording {
		ended := stamp
		return models.SessionRecording{
			ArtifactPath: "/data/recordings/1/rec-1.mp4",
			ArtifactURL:  "/v1/me/recordings/1/artifact",
			EndedAt:      &ended,
			UpdatedAt:    stamp,
		}
	}

	cases := []struct {
		name string
		row  models.SessionRecording
		want bool
	}{
		{"old artifact expires", withArtifact(old), true},
		{"recent artifact is kept", withArtifact(recent), false},
		{"row without media is never a purge candidate", models.SessionRecording{
			EndedAt: &old, UpdatedAt: old,
		}, false},
		{"url without path is still purged", models.SessionRecording{
			ArtifactURL: "/v1/me/recordings/2/artifact",
			EndedAt:     &old,
			UpdatedAt:   old,
		}, true},
		{"a row holding only a preview is still a purge candidate", models.SessionRecording{
			ThumbnailPath: "/data/recordings/3/thumb-3.jpg",
			EndedAt:       &old,
			UpdatedAt:     old,
		}, true},
	}
	for _, tc := range cases {
		if got := ArtifactExpired(tc.row, cutoff); got != tc.want {
			t.Fatalf("%s: ArtifactExpired() = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestArtifactExpiredFallsBackToUpdatedAt(t *testing.T) {
	now := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	cutoff := now.Add(-24 * time.Hour)
	// A recording that never finished has no ended_at; updated_at decides.
	row := models.SessionRecording{
		ArtifactPath: "/data/recordings/1/rec-9.mp4",
		UpdatedAt:    now.Add(-72 * time.Hour),
	}
	if !ArtifactExpired(row, cutoff) {
		t.Fatal("expected a recording with only updated_at in the past to expire")
	}
}

// The orphan reaper must recognise exactly the names this service writes, and
// must refuse to touch an in-flight upload temporary.
func TestRecordFileID(t *testing.T) {
	cases := []struct {
		name string
		want uint
		ok   bool
	}{
		{"rec-42.mp4", 42, true},
		{"rec-42.webm", 42, true},
		{"thumb-42.jpg", 42, true},
		{"thumb-42.png", 42, true},
		{"rec-42.mp4.part", 0, false},
		{"thumb-42.jpg.part", 0, false},
		{"rec-abc.mp4", 0, false},
		{"rec-.mp4", 0, false},
		{"other-42.mp4", 0, false},
		{"42.mp4", 0, false},
		{"", 0, false},
	}
	for _, tc := range cases {
		got, ok := RecordFileID(tc.name)
		if ok != tc.ok || (ok && got != tc.want) {
			t.Errorf("RecordFileID(%q) = (%d, %v), want (%d, %v)", tc.name, got, ok, tc.want, tc.ok)
		}
	}
}

// The staging names are the same names with a `.part` suffix, and recognising
// them is what lets the reaper clean up an abandoned resumable upload without
// touching a file that is still being written.
func TestUploadTempRecordingID(t *testing.T) {
	cases := []struct {
		name string
		want uint
		ok   bool
	}{
		{"rec-42.mp4.part", 42, true},
		{"rec-42.webm.part", 42, true},
		{"thumb-42.jpg.part", 42, true},
		{"rec-42.mp4", 0, false},
		{"rec-42.mp4.partial", 0, false},
		{"rec-.mp4.part", 0, false},
		{"rec-abc.mp4.part", 0, false},
		{"other-42.mp4.part", 0, false},
		{"", 0, false},
	}
	for _, tc := range cases {
		got, ok := UploadTempRecordingID(tc.name)
		if ok != tc.ok || (ok && got != tc.want) {
			t.Errorf("UploadTempRecordingID(%q) = (%d, %v), want (%d, %v)",
				tc.name, got, ok, tc.want, tc.ok)
		}
	}
}

// A staging file is cleaned up on age alone: the row it belongs to may be
// perfectly alive, which is exactly the case an orphan-only rule would miss.
func TestReapOrphanFilesRemovesOnlyStaleUploads(t *testing.T) {
	dir := t.TempDir()
	userDir := filepath.Join(dir, "7")
	if err := os.MkdirAll(userDir, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	write := func(name string, age time.Duration) {
		t.Helper()
		path := filepath.Join(userDir, name)
		if err := os.WriteFile(path, []byte("chunk"), 0o640); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
		stamp := time.Now().Add(-age)
		if err := os.Chtimes(path, stamp, stamp); err != nil {
			t.Fatalf("chtimes %s: %v", name, err)
		}
	}
	write("rec-1.mp4.part", uploadTempMinAge+time.Minute) // abandoned
	write("rec-2.mp4.part", time.Minute)                  // still being written
	write("notes.txt", 48*time.Hour)                      // not a name we own

	// The staging branch never asks the database whether a row exists, which is
	// the property under test; the stored-media branch is exercised elsewhere.
	lifecycle := NewRecordingLifecycle(NewSessionRecordingService(nil), dir, time.Hour, time.Hour, time.Hour)
	orphans, uploads, err := lifecycle.ReapOrphanFiles(context.Background())
	if err != nil {
		t.Fatalf("reap: %v", err)
	}
	if uploads != 1 || orphans != 0 {
		t.Fatalf("reaped uploads=%d orphans=%d, want 1 and 0", uploads, orphans)
	}
	if _, err := os.Stat(filepath.Join(userDir, "rec-1.mp4.part")); !os.IsNotExist(err) {
		t.Fatalf("the abandoned staging file survived: %v", err)
	}
	for _, keep := range []string{"rec-2.mp4.part", "notes.txt"} {
		if _, err := os.Stat(filepath.Join(userDir, keep)); err != nil {
			t.Fatalf("%s should have been left alone: %v", keep, err)
		}
	}
}

// Deleting a row has to take its staging file with it. Otherwise a client that
// abandons a half-uploaded recording and then deletes the row leaves bytes that
// nothing points at - the exact leak the resumable path makes easier to create.
func TestRemoveStoredMediaDropsStagingFiles(t *testing.T) {
	dir := t.TempDir()
	artifact := filepath.Join(dir, "rec-5.mp4")
	thumbnail := filepath.Join(dir, "thumb-5.jpg")
	paths := []string{artifact, artifact + ".part", thumbnail, thumbnail + ".part"}
	for _, path := range paths {
		if err := os.WriteFile(path, []byte("x"), 0o640); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	svc := NewSessionRecordingService(nil)
	svc.removeStoredMedia(models.SessionRecording{
		ArtifactPath:  artifact,
		ThumbnailPath: thumbnail,
	})

	for _, path := range paths {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("%s should have been deleted, stat err = %v", path, err)
		}
	}
}
