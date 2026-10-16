package service

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"quickdesk/signaling/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Version order is the whole point of an update channel: if it cannot say which
// of two versions is newer, it cannot answer "is there an update".
func TestCompareUpdateVersions(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"4.2.1.0", "4.2.1.0", 0},
		{"4.2", "4.2.0.0", 0},
		{"4.2.10.0", "4.2.9.0", 1},
		{"4.2.9.0", "4.2.10.0", -1},
		{"v4.2.1.0", "4.2.0.9", 1},
		{"4.2.1.1", "4.2.1.0", 1},
		{"5.0.0.0", "4.99.99.99", 1},
		{"0.0.1", "0.0.0", 1},
	}
	for _, tc := range cases {
		got, err := CompareUpdateVersions(tc.a, tc.b)
		if err != nil {
			t.Fatalf("compare %q %q: %v", tc.a, tc.b, err)
		}
		if got != tc.want {
			t.Fatalf("compare %q %q = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
	for _, invalid := range []string{"", "4..1", "4.x", "4.1-rc", "-1.0"} {
		if _, err := ParseUpdateVersion(invalid); err == nil {
			t.Fatalf("version %q parsed, want an error", invalid)
		}
	}
}

func newUpdateReleaseTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("QD_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set QD_TEST_DATABASE_URL to a throwaway Postgres to run the update-channel tests")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	})
	if err := db.AutoMigrate(&models.UpdateRelease{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := db.Exec("TRUNCATE update_releases").Error; err != nil {
		t.Fatalf("truncate: %v", err)
	}
	return db
}

func sampleRelease(version string) PublishUpdateReleaseInput {
	return PublishUpdateReleaseInput{
		Channel:     UpdateChannelStable,
		Platform:    "windows",
		Arch:        "x64",
		Version:     version,
		DownloadURL: "https://updates.example.com/quickdesk-" + version + ".exe",
		SHA256:      "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		SizeBytes:   123456789,
		Notes:       "test build",
	}
}

func TestUpdateReleasePublishValidatesInput(t *testing.T) {
	svc := NewUpdateReleaseService(newUpdateReleaseTestDB(t))
	cases := []struct {
		name   string
		mutate func(*PublishUpdateReleaseInput)
	}{
		{"unknown channel", func(in *PublishUpdateReleaseInput) { in.Channel = "canary" }},
		{"unknown platform", func(in *PublishUpdateReleaseInput) { in.Platform = "solaris" }},
		{"unknown arch", func(in *PublishUpdateReleaseInput) { in.Arch = "sparc" }},
		{"non-numeric version", func(in *PublishUpdateReleaseInput) { in.Version = "4.2-beta" }},
		{"relative download url", func(in *PublishUpdateReleaseInput) { in.DownloadURL = "/downloads/setup.exe" }},
		{"missing checksum", func(in *PublishUpdateReleaseInput) { in.SHA256 = "" }},
		{"short checksum", func(in *PublishUpdateReleaseInput) { in.SHA256 = "abc123" }},
		{"negative size", func(in *PublishUpdateReleaseInput) { in.SizeBytes = -1 }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := sampleRelease("4.2.1.0")
			tc.mutate(&input)
			if _, err := svc.Publish(context.Background(), input); err == nil {
				t.Fatal("publish accepted invalid input")
			} else {
				var invalid *UpdateReleaseValidationError
				if !errors.As(err, &invalid) {
					t.Fatalf("error %v is not a validation error", err)
				}
			}
		})
	}
}

func TestUpdateReleasePublishListDelete(t *testing.T) {
	svc := NewUpdateReleaseService(newUpdateReleaseTestDB(t))
	ctx := context.Background()

	if _, err := svc.Publish(ctx, sampleRelease("4.2.1.0")); err != nil {
		t.Fatalf("publish: %v", err)
	}
	// The same version for a different platform is a different release.
	other := sampleRelease("4.2.1.0")
	other.Platform = "macos"
	other.Arch = "arm64"
	if _, err := svc.Publish(ctx, other); err != nil {
		t.Fatalf("publish other platform: %v", err)
	}
	// Publishing the same tuple twice is a conflict, not a replacement.
	if _, err := svc.Publish(ctx, sampleRelease("4.2.1.0")); !errors.Is(err, ErrUpdateReleaseExists) {
		t.Fatalf("duplicate publish error = %v, want ErrUpdateReleaseExists", err)
	}

	items, err := svc.List(ctx, UpdateReleaseFilter{Platform: "windows"})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 1 || items[0].Version != "4.2.1.0" {
		t.Fatalf("windows listing = %+v, want the one release", items)
	}

	if err := svc.Delete(ctx, items[0].ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if err := svc.Delete(ctx, items[0].ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("second delete error = %v, want record not found", err)
	}
}

func TestUpdateReleaseCheck(t *testing.T) {
	svc := NewUpdateReleaseService(newUpdateReleaseTestDB(t))
	ctx := context.Background()

	empty, err := svc.Check(ctx, "windows", "x64", "", "4.0.0.0")
	if err != nil {
		t.Fatalf("check with no releases: %v", err)
	}
	if empty.UpdateAvailable || empty.Latest != nil {
		t.Fatalf("empty channel reported %+v, want no update and no latest", empty)
	}
	if empty.Channel != UpdateChannelStable {
		t.Fatalf("empty check channel = %q, want the default %q", empty.Channel, UpdateChannelStable)
	}

	first := sampleRelease("4.2.0.0")
	if _, err := svc.Publish(ctx, first); err != nil {
		t.Fatalf("publish 4.2.0.0: %v", err)
	}
	older, err := svc.Check(ctx, "windows", "x64", "stable", "4.1.9.9")
	if err != nil {
		t.Fatalf("check older: %v", err)
	}
	if !older.UpdateAvailable || older.Latest == nil || older.Latest.Version != "4.2.0.0" {
		t.Fatalf("older check = %+v, want an available 4.2.0.0", older)
	}

	same, err := svc.Check(ctx, "windows", "x64", "stable", "4.2.0.0")
	if err != nil {
		t.Fatalf("check same: %v", err)
	}
	if same.UpdateAvailable {
		t.Fatalf("re-checking the installed version reported an update: %+v", same)
	}

	newer, err := svc.Check(ctx, "windows", "x64", "stable", "4.3.0.0")
	if err != nil {
		t.Fatalf("check newer: %v", err)
	}
	if newer.UpdateAvailable {
		t.Fatalf("a client ahead of the channel reported an update: %+v", newer)
	}

	// A second publish that is actually newer wins regardless of insert order.
	second := sampleRelease("4.2.1.0")
	second.Mandatory = true
	at := time.Now().UTC().Add(time.Minute)
	second.PublishedAt = at
	if _, err := svc.Publish(ctx, second); err != nil {
		t.Fatalf("publish 4.2.1.0: %v", err)
	}
	latest, err := svc.Check(ctx, "windows", "x64", "stable", "4.2.0.0")
	if err != nil {
		t.Fatalf("check latest: %v", err)
	}
	if !latest.UpdateAvailable || !latest.Mandatory || latest.Latest == nil || latest.Latest.Version != "4.2.1.0" {
		t.Fatalf("latest check = %+v, want the mandatory 4.2.1.0", latest)
	}

	// A caller that does not know its version still gets the newest release.
	unknown, err := svc.Check(ctx, "windows", "x64", "stable", "")
	if err != nil {
		t.Fatalf("check without a version: %v", err)
	}
	if !unknown.UpdateAvailable || unknown.Latest == nil {
		t.Fatalf("version-less check = %+v, want the newest release", unknown)
	}

	if _, err := svc.Check(ctx, "solaris", "x64", "stable", "4.0.0.0"); err == nil {
		t.Fatal("check accepted an unknown platform")
	}
	if _, err := svc.Check(ctx, "windows", "x64", "stable", "not-a-version"); err == nil {
		t.Fatal("check accepted an invalid current_version")
	}
}
