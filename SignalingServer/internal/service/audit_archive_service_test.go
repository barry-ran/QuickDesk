package service

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"quickdesk/signaling/internal/models"
)

// writeAuditArchive writes one day's archive file the way retention does: one
// JSON object per line, oldest first. The `raw` lines let a test inject a
// malformed entry.
func writeAuditArchive(t *testing.T, dir, date string, rows []models.AuditLog, raw ...string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir archive: %v", err)
	}
	path := filepath.Join(dir, auditArchiveFileName(date))
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create archive: %v", err)
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	for _, row := range rows {
		if err := enc.Encode(row); err != nil {
			t.Fatalf("encode row: %v", err)
		}
	}
	for _, line := range raw {
		if _, err := f.WriteString(line + "\n"); err != nil {
			t.Fatalf("write raw line: %v", err)
		}
	}
}

func sampleArchiveRows() []models.AuditLog {
	return []models.AuditLog{
		{
			ID:           1,
			ActorUserID:  7,
			Action:       "device.access.denied",
			ResourceType: "device",
			ResourceID:   "126735214",
			Details:      "denied by policy",
			IP:           "198.51.100.7",
			CreatedAt:    time.Date(2026, 10, 16, 1, 0, 0, 0, time.UTC),
		},
		{
			ID:           2,
			ActorUserID:  9,
			Action:       "recording.delete",
			ResourceType: "session_recording",
			ResourceID:   "5",
			Details:      "recording 5 deleted",
			IP:           "203.0.113.9",
			CreatedAt:    time.Date(2026, 10, 16, 2, 0, 0, 0, time.UTC),
		},
		{
			ID:            3,
			AdminID:       1,
			AdminUsername: "admin",
			Action:        "audit.sweep",
			ResourceType:  "audit_log",
			Details:       "purged=2",
			IP:            "127.0.0.1",
			CreatedAt:     time.Date(2026, 10, 16, 3, 0, 0, 0, time.UTC),
		},
	}
}

func TestAuditArchiveRequiresConfigurationAndFile(t *testing.T) {
	ctx := context.Background()

	svc := NewAuditService(nil)
	if _, err := svc.QueryArchive(ctx, AuditArchiveQuery{}); !errors.Is(err, ErrAuditArchiveNotConfigured) {
		t.Fatalf("unconfigured archive err = %v, want ErrAuditArchiveNotConfigured", err)
	}

	svc = NewAuditService(nil).WithArchiveDir(t.TempDir())
	if _, err := svc.QueryArchive(ctx, AuditArchiveQuery{Date: "2026-10-16"}); !errors.Is(err, ErrAuditArchiveNotFound) {
		t.Fatalf("missing archive err = %v, want ErrAuditArchiveNotFound", err)
	}
}

func TestAuditArchiveRejectsBadDate(t *testing.T) {
	svc := NewAuditService(nil).WithArchiveDir(t.TempDir())
	if _, err := svc.QueryArchive(context.Background(), AuditArchiveQuery{Date: "16-10-2026"}); !errors.Is(err, ErrAuditArchiveBadDate) {
		t.Fatalf("bad date err = %v, want ErrAuditArchiveBadDate", err)
	}
}

func TestAuditArchiveScansNewestFirstAndFilters(t *testing.T) {
	dir := t.TempDir()
	writeAuditArchive(t, dir, "2026-10-16", sampleArchiveRows())
	svc := NewAuditService(nil).WithArchiveDir(dir)
	ctx := context.Background()

	all, err := svc.QueryArchive(ctx, AuditArchiveQuery{Date: "2026-10-16"})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(all.Items) != 3 {
		t.Fatalf("got %d entries, want 3", len(all.Items))
	}
	if all.Items[0].Action != "audit.sweep" || all.Items[2].Action != "device.access.denied" {
		t.Fatalf("order = [%s %s %s], want newest first",
			all.Items[0].Action, all.Items[1].Action, all.Items[2].Action)
	}
	if all.Truncated || all.HasMore {
		t.Fatalf("small complete scan reported truncated=%t has_more=%t", all.Truncated, all.HasMore)
	}

	byAction, err := svc.QueryArchive(ctx, AuditArchiveQuery{Date: "2026-10-16", Action: "recording.delete"})
	if err != nil {
		t.Fatalf("action filter: %v", err)
	}
	if len(byAction.Items) != 1 || byAction.Items[0].ID != 2 {
		t.Fatalf("action filter returned %+v, want only row 2", byAction.Items)
	}

	byText, err := svc.QueryArchive(ctx, AuditArchiveQuery{Date: "2026-10-16", Query: "POLICY"})
	if err != nil {
		t.Fatalf("text filter: %v", err)
	}
	if len(byText.Items) != 1 || byText.Items[0].ID != 1 {
		t.Fatalf("text filter returned %+v, want only row 1 (case-insensitive)", byText.Items)
	}

	byActor, err := svc.QueryArchive(ctx, AuditArchiveQuery{Date: "2026-10-16", ActorUserID: 9})
	if err != nil {
		t.Fatalf("actor filter: %v", err)
	}
	if len(byActor.Items) != 1 || byActor.Items[0].ID != 2 {
		t.Fatalf("actor filter returned %+v, want only row 2", byActor.Items)
	}

	byAdmin, err := svc.QueryArchive(ctx, AuditArchiveQuery{Date: "2026-10-16", AdminUsername: "ADM"})
	if err != nil {
		t.Fatalf("admin filter: %v", err)
	}
	if len(byAdmin.Items) != 1 || byAdmin.Items[0].ID != 3 {
		t.Fatalf("admin filter returned %+v, want only row 3", byAdmin.Items)
	}
}

func TestAuditArchiveLimitReportsHasMore(t *testing.T) {
	dir := t.TempDir()
	writeAuditArchive(t, dir, "2026-10-16", sampleArchiveRows())
	svc := NewAuditService(nil).WithArchiveDir(dir)

	res, err := svc.QueryArchive(context.Background(), AuditArchiveQuery{Date: "2026-10-16", Limit: 1})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(res.Items) != 1 || res.Items[0].ID != 3 {
		t.Fatalf("limit 1 returned %+v, want only the newest row", res.Items)
	}
	if !res.HasMore {
		t.Fatal("limit 1 over 3 rows reported has_more=false, want true")
	}
	if res.Truncated {
		t.Fatal("limit hit reported truncated=true; it should be has_more, not a budget cut")
	}
}

func TestAuditArchiveSkipsMalformedLine(t *testing.T) {
	dir := t.TempDir()
	// A single damaged line sits in the middle of otherwise good rows.
	writeAuditArchive(t, dir, "2026-10-16", sampleArchiveRows()[:2], "{not json", "")
	svc := NewAuditService(nil).WithArchiveDir(dir)

	res, err := svc.QueryArchive(context.Background(), AuditArchiveQuery{Date: "2026-10-16"})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(res.Items) != 2 {
		t.Fatalf("got %d entries, want the 2 valid rows and the bad one skipped", len(res.Items))
	}
}
