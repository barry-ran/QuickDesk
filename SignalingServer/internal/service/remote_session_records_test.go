package service

import (
	"context"
	"fmt"
	"testing"

	"quickdesk/signaling/internal/models"
)

// The device-scoped list joins a page of sessions with their records in two
// batched queries, so those queries are what every row on that page depends on.
// The filters matter as much as the join: a recording belongs to the account
// that produced it, and an access-log row belongs to a device, so neither batch
// may hand back a neighbour's row just because the session id matched.
func TestRemoteSessionRelatedRecordsBatchFilters(t *testing.T) {
	db := newRemoteSessionTestDB(t)
	if err := db.AutoMigrate(&models.SessionRecording{}, &models.DeviceAccessLog{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	ctx := context.Background()
	const (
		deviceID   = "700000006"
		otherDev   = "700000007"
		owner      = 990301
		otherOwner = 990302
	)
	// Clear this test's own rows first: `session_recordings.user_id` is a foreign
	// key, so a leftover row from an earlier run would make the seeding below
	// fail on the user delete.
	for _, table := range []string{"session_recordings", "device_access_logs"} {
		if err := db.Exec("DELETE FROM "+table+" WHERE device_id IN ?", []string{deviceID, otherDev}).Error; err != nil {
			t.Fatalf("clear %s: %v", table, err)
		}
	}
	seedRemoteSessionDevice(t, db, deviceID, owner)
	seedRemoteSessionDevice(t, db, otherDev, owner)
	// The recording rows point at a user, so the second account needs a row of
	// its own before a row claiming it can exist.
	if err := db.Exec(
		"INSERT INTO users (id, username, password) VALUES (?, ?, 'test-hash') ON CONFLICT (id) DO NOTHING",
		otherOwner, fmt.Sprintf("qa-remote-session-%d", otherOwner),
	).Error; err != nil {
		t.Fatalf("seed second user: %v", err)
	}

	recordings := []models.SessionRecording{
		{UserID: owner, DeviceID: deviceID, SessionID: "sig_a", Status: "completed"},
		{UserID: owner, DeviceID: deviceID, SessionID: "sig_b", Status: "recording"},
		{UserID: otherOwner, DeviceID: otherDev, SessionID: "sig_a", Status: "completed"},
	}
	if err := db.Create(&recordings).Error; err != nil {
		t.Fatalf("seed recordings: %v", err)
	}
	logs := []models.DeviceAccessLog{
		{DeviceID: deviceID, OwnerID: owner, SessionID: "sig_a", Result: DeviceAccessSessionStarted},
		{DeviceID: otherDev, OwnerID: owner, SessionID: "sig_a", Result: DeviceAccessSessionStarted},
		{DeviceID: deviceID, OwnerID: owner, SessionID: "sig_c", Result: DeviceAccessSessionStarted},
	}
	if err := db.Create(&logs).Error; err != nil {
		t.Fatalf("seed access log: %v", err)
	}

	recSvc := NewSessionRecordingService(db)
	mine, err := recSvc.ListForSessions(ctx, owner, []string{"sig_a", "sig_b"}, 100)
	if err != nil {
		t.Fatalf("batch recordings: %v", err)
	}
	if len(mine) != 2 {
		t.Fatalf("batch returned %d recordings for the caller's own two sessions, want 2", len(mine))
	}
	for _, row := range mine {
		if row.UserID != owner {
			t.Fatalf("a recording from another account leaked into the batch: %+v", row)
		}
	}

	accessSvc := NewDeviceAccessService(db)
	onDevice, err := accessSvc.ListForSessions(ctx, deviceID, []string{"sig_a", "sig_b", "sig_c"}, 100)
	if err != nil {
		t.Fatalf("batch access log: %v", err)
	}
	if len(onDevice) != 2 {
		t.Fatalf("batch returned %d access-log rows for this device, want the 2 that name it", len(onDevice))
	}
	for _, row := range onDevice {
		if row.DeviceID != deviceID {
			t.Fatalf("an access-log row from another device leaked into the batch: %+v", row)
		}
		if row.SessionID == "sig_b" {
			t.Fatalf("the batch returned a row for a session that never reached this device: %+v", row)
		}
	}

	// An empty page is not a query, and must not be an error.
	if rows, err := recSvc.ListForSessions(ctx, owner, nil, 100); err != nil || len(rows) != 0 {
		t.Fatalf("empty recording batch = %d rows (err=%v), want none", len(rows), err)
	}
	if rows, err := accessSvc.ListForSessions(ctx, deviceID, nil, 100); err != nil || len(rows) != 0 {
		t.Fatalf("empty access batch = %d rows (err=%v), want none", len(rows), err)
	}
}
