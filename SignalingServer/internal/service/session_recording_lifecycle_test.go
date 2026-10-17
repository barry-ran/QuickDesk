package service

import (
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
