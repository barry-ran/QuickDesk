package handler

import (
	"context"
	"fmt"
	"os"
	"testing"

	"quickdesk/signaling/internal/models"
	"quickdesk/signaling/internal/service"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// newDeviceRevokeTestDB opens the throwaway Postgres the service tests use and
// leaves it empty of this test's own device. The batch read is a real query, so
// a fake database would only be testing the fake.
func newDeviceRevokeTestDB(t *testing.T, slot int) (*gorm.DB, string) {
	t.Helper()
	dsn := os.Getenv("QD_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set QD_TEST_DATABASE_URL to a throwaway Postgres to run the device revoke tests")
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
	if err := db.AutoMigrate(&models.RemoteSession{}, &models.Device{}, &models.User{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	// One device id per test, passed in, so two tests in this file cannot
	// collide and the ids do not silently move when a test is renamed.
	deviceID := fmt.Sprintf("7000000%02d", slot)
	if err := db.Exec("DELETE FROM remote_sessions WHERE device_id = ?", deviceID).Error; err != nil {
		t.Fatalf("clear sessions: %v", err)
	}
	if err := db.Exec("DELETE FROM devices WHERE device_id = ?", deviceID).Error; err != nil {
		t.Fatalf("clear device: %v", err)
	}
	owner := uint(990240 + slot)
	username := fmt.Sprintf("qa-devrevoke-%d", owner)
	if err := db.Exec("DELETE FROM users WHERE username = ?", username).Error; err != nil {
		t.Fatalf("clear user: %v", err)
	}
	if err := db.Exec(
		"INSERT INTO users (id, username, password) VALUES (?, ?, 'test-hash') ON CONFLICT (id) DO NOTHING",
		owner, username,
	).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := db.Create(&models.Device{
		DeviceID:   deviceID,
		DeviceUUID: "uuid-" + deviceID,
		DeviceName: "batch-" + deviceID,
		UserID:     &owner,
	}).Error; err != nil {
		t.Fatalf("seed device: %v", err)
	}
	return db, deviceID
}

// A device-wide revoke is a batch, so the thing worth pinning is not that it
// closes a socket but that it reports every row honestly: one session that
// cannot be reached must not stop the others, and the operator has to be able to
// see which one it was.
func TestRevokeDeviceSessionsReportsPerSessionOutcomes(t *testing.T) {
	db, deviceID := newDeviceRevokeTestDB(t, 41)
	svc := service.NewRemoteSessionService(db)
	ctx := context.Background()
	for _, in := range []service.OpenRemoteSessionInput{
		{SessionID: "batch_a", DeviceID: deviceID, InstanceID: "inst-here"},
		{SessionID: "batch_b", DeviceID: deviceID, InstanceID: "inst-here"},
		// Held by another instance, which no command reaches in this test.
		{SessionID: "batch_far", DeviceID: deviceID, InstanceID: "inst-far"},
	} {
		if _, err := svc.Open(ctx, in); err != nil {
			t.Fatalf("open %s: %v", in.SessionID, err)
		}
	}
	rev := &fakeRevoker{
		instanceID: "inst-here",
		local:      map[string]bool{"batch_a": true, "batch_b": true},
		// Stand in for the owning socket's teardown, which is what actually
		// stamps the row: without it the local path could not be confirmed.
		onClose: func(id string) {
			if _, _, err := svc.Close(ctx, id, models.RemoteSessionEndRevoked); err != nil {
				t.Errorf("closing %s: %v", id, err)
			}
		},
	}
	revoked, failed, truncated, results, err := revokeDeviceSessions(ctx, svc, rev, deviceID, 10)
	if err != nil {
		t.Fatalf("revoke device sessions: %v", err)
	}
	if revoked != 2 || failed != 1 {
		t.Fatalf("revoked=%d failed=%d, want 2 and 1", revoked, failed)
	}
	if truncated {
		t.Fatal("a 3-session batch reported truncation under a cap of 10")
	}
	if len(results) != 3 {
		t.Fatalf("results = %d entries, want one per live session", len(results))
	}
	byID := map[string]map[string]interface{}{}
	for _, entry := range results {
		byID[entry["session_id"].(string)] = entry
	}
	for _, id := range []string{"batch_a", "batch_b"} {
		if byID[id]["revoked"] != true {
			t.Fatalf("%s is not reported as revoked: %+v", id, byID[id])
		}
	}
	if byID["batch_far"]["revoked"] != false {
		t.Fatalf("the unreachable session is reported as revoked: %+v", byID["batch_far"])
	}
	if byID["batch_far"]["reason"] == nil || byID["batch_far"]["reason"] == "" {
		t.Fatalf("the unreachable session carries no reason: %+v", byID["batch_far"])
	}
	if !rev.asked || rev.askedInst != "inst-far" {
		t.Fatalf("the unreachable session was not handed to its instance: %+v", rev)
	}
}

// A cap that is hit is reported, not silently swallowed: an operator looking at
// "revoked 2" on a machine with three controllers has to know the machine is not
// necessarily empty.
func TestRevokeDeviceSessionsReportsTruncation(t *testing.T) {
	db, deviceID := newDeviceRevokeTestDB(t, 42)
	svc := service.NewRemoteSessionService(db)
	ctx := context.Background()
	for _, id := range []string{"cap_a", "cap_b", "cap_c"} {
		if _, err := svc.Open(ctx, service.OpenRemoteSessionInput{
			SessionID: id, DeviceID: deviceID, InstanceID: "inst-here",
		}); err != nil {
			t.Fatalf("open %s: %v", id, err)
		}
	}
	rev := &fakeRevoker{
		instanceID: "inst-here",
		local:      map[string]bool{"cap_a": true, "cap_b": true, "cap_c": true},
		onClose: func(id string) {
			if _, _, err := svc.Close(ctx, id, models.RemoteSessionEndRevoked); err != nil {
				t.Errorf("closing %s: %v", id, err)
			}
		},
	}
	revoked, failed, truncated, results, err := revokeDeviceSessions(ctx, svc, rev, deviceID, 2)
	if err != nil {
		t.Fatalf("revoke device sessions: %v", err)
	}
	if revoked != 2 || failed != 0 {
		t.Fatalf("revoked=%d failed=%d, want 2 and 0", revoked, failed)
	}
	if !truncated {
		t.Fatal("hitting the cap was not reported")
	}
	if len(results) != 2 {
		t.Fatalf("results = %d entries, want the cap", len(results))
	}
}

// A device with nothing live is not an error and not a failure - it is zero.
func TestRevokeDeviceSessionsOnAQuietDevice(t *testing.T) {
	db, deviceID := newDeviceRevokeTestDB(t, 43)
	svc := service.NewRemoteSessionService(db)
	revoked, failed, truncated, results, err := revokeDeviceSessions(
		context.Background(), svc, &fakeRevoker{instanceID: "inst-here"}, deviceID, 10)
	if err != nil {
		t.Fatalf("revoke device sessions: %v", err)
	}
	if revoked != 0 || failed != 0 || truncated || len(results) != 0 {
		t.Fatalf("quiet device reported revoked=%d failed=%d truncated=%t rows=%d",
			revoked, failed, truncated, len(results))
	}
}
