package service

import (
	"errors"
	"testing"
	"time"
)

func TestNotificationBackoffGrowsAndCaps(t *testing.T) {
	// First retry is 2s and each subsequent retry doubles, until the cap.
	want := []time.Duration{
		2 * time.Second,
		4 * time.Second,
		8 * time.Second,
		16 * time.Second,
	}
	for i, w := range want {
		if got := notificationBackoff(i + 1); got != w {
			t.Errorf("notificationBackoff(%d) = %v, want %v", i+1, got, w)
		}
	}

	// Every later attempt is clamped to the cap and never smaller than the
	// previous one (monotonic, no wrap-around from the shift).
	prev := notificationBackoff(4)
	for attempts := 5; attempts <= 20; attempts++ {
		got := notificationBackoff(attempts)
		if got > notificationBackoffCap {
			t.Fatalf("notificationBackoff(%d) = %v exceeds cap %v", attempts, got, notificationBackoffCap)
		}
		if got < prev {
			t.Fatalf("notificationBackoff(%d) = %v dropped below previous %v", attempts, got, prev)
		}
		prev = got
	}
	if prev != notificationBackoffCap {
		t.Errorf("high attempt counts should reach the cap, got %v", prev)
	}

	// Non-positive attempts are treated as the first retry.
	if got := notificationBackoff(0); got != 2*time.Second {
		t.Errorf("notificationBackoff(0) = %v, want 2s", got)
	}
}

func TestMessageFor(t *testing.T) {
	if got := messageFor(0, errors.New("dial tcp: connection refused")); got != "dial tcp: connection refused" {
		t.Errorf("transport error should surface the error text, got %q", got)
	}
	if got := messageFor(0, nil); got != "transport error" {
		t.Errorf("status 0 without error = %q, want transport error", got)
	}
	if got := messageFor(503, nil); got != "HTTP 503" {
		t.Errorf("messageFor(503) = %q, want HTTP 503", got)
	}
}

// The outbox archive can live in the same directory as the audit archive, so
// its rotation must only ever consider its own files. A shared `archives/`
// folder is a realistic deployment, and deleting the other family's files
// would be silent evidence loss.
func TestNotificationSweepArchiveOnlyTouchesItsOwnFiles(t *testing.T) {
	dir := t.TempDir()
	now := time.Now().UTC()
	today := now.Format("2006-01-02")
	old := now.AddDate(0, 0, -40).Format("2006-01-02")
	writeFile(t, dir, "notification-deliveries-"+today+".jsonl")
	writeFile(t, dir, "notification-deliveries-"+old+".jsonl")
	writeFile(t, dir, "audit-logs-"+old+".jsonl")

	svc := NewNotificationDeliveryService(nil).WithArchiveDir(dir)
	removed, err := svc.SweepArchive(7)
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if removed != 1 {
		t.Fatalf("removed %d files, want only the old outbox archive", removed)
	}
	want := []string{
		"audit-logs-" + old + ".jsonl",
		"notification-deliveries-" + today + ".jsonl",
	}
	if got := remainingFiles(t, dir); !equalStrings(got, want) {
		t.Fatalf("remaining = %v, want %v", got, want)
	}
}
