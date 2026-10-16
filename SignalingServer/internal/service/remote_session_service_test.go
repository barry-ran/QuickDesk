package service

import (
	"context"
	"fmt"
	"os"
	"testing"

	"quickdesk/signaling/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func newRemoteSessionTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("QD_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set QD_TEST_DATABASE_URL to a throwaway Postgres to run the remote-session tests")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.RemoteSession{}, &models.Device{}, &models.User{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := db.Exec("DELETE FROM remote_sessions").Error; err != nil {
		t.Fatalf("clear remote sessions: %v", err)
	}
	return db
}

// seedRemoteSessionDevice creates the owner and device row the session resolves
// its owner from; the api's owner-scoped list is what the row is for.
func seedRemoteSessionDevice(t *testing.T, db *gorm.DB, deviceID string, userID uint) {
	t.Helper()
	username := fmt.Sprintf("qa-remote-session-%d", userID)
	// Devices first: `devices.user_id` is a foreign key, so a leftover device
	// from an earlier run would block the user delete. Both clears are by this
	// test's own identifiers, so a neighbouring test's rows are never touched.
	if err := db.Exec("DELETE FROM devices WHERE device_id = ?", deviceID).Error; err != nil {
		t.Fatalf("clear device: %v", err)
	}
	if err := db.Exec("DELETE FROM devices WHERE user_id = ?", userID).Error; err != nil {
		t.Fatalf("clear devices owned by the seeded user: %v", err)
	}
	if err := db.Exec("DELETE FROM users WHERE username = ?", username).Error; err != nil {
		t.Fatalf("clear user: %v", err)
	}
	if err := db.Exec(
		"INSERT INTO users (id, username, password) VALUES (?, ?, 'test-hash') ON CONFLICT (id) DO NOTHING",
		userID, username,
	).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	owner := userID
	device := models.Device{
		DeviceID:   deviceID,
		DeviceUUID: "uuid-" + deviceID,
		DeviceName: "workshop-" + deviceID,
		UserID:     &owner,
	}
	if err := db.Create(&device).Error; err != nil {
		t.Fatalf("seed device: %v", err)
	}
}

// A session has a real lifecycle: it opens once, it is queryable while it is
// open, it closes once, and it stays closed. The idempotence on both ends is the
// part worth pinning - bookkeeping must never fail a signaling handshake, and it
// must not let a duplicate teardown look like a second session ending.
func TestRemoteSessionLifecycle(t *testing.T) {
	db := newRemoteSessionTestDB(t)
	svc := NewRemoteSessionService(db)
	ctx := context.Background()
	seedRemoteSessionDevice(t, db, "700000001", 990201)

	opened, err := svc.Open(ctx, OpenRemoteSessionInput{
		SessionID: "sig_test_open",
		DeviceID:  "700000001",
		ClientID:  "controller-1",
		ClientIP:  "203.0.113.5",
	})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if opened.UserID != 990201 {
		t.Fatalf("session owner = %d, want the device's owner 990201", opened.UserID)
	}
	if !opened.Active || opened.EndedAt != nil || opened.EndReason != "" {
		t.Fatalf("a fresh session is not open: %+v", opened)
	}

	// Opening the same session id again is a retry, not a second session.
	again, err := svc.Open(ctx, OpenRemoteSessionInput{
		SessionID: "sig_test_open",
		DeviceID:  "700000001",
		ClientID:  "controller-1",
	})
	if err != nil {
		t.Fatalf("re-open: %v", err)
	}
	if again.ID != opened.ID {
		t.Fatalf("re-open created a second row: %d vs %d", again.ID, opened.ID)
	}

	active := true
	open, err := svc.List(ctx, RemoteSessionFilter{UserID: 990201, Active: &active})
	if err != nil {
		t.Fatalf("list active: %v", err)
	}
	if len(open) != 1 || open[0].SessionID != "sig_test_open" {
		t.Fatalf("active list = %+v, want the one open session", open)
	}
	if open[0].DurationSeconds < 0 {
		t.Fatalf("an open session reported a negative duration: %+v", open[0])
	}

	ended := false
	finished, err := svc.List(ctx, RemoteSessionFilter{UserID: 990201, Active: &ended})
	if err != nil {
		t.Fatalf("list ended: %v", err)
	}
	if len(finished) != 0 {
		t.Fatalf("a session that has not ended appeared in the ended list: %+v", finished)
	}

	closed, changed, err := svc.Close(ctx, "sig_test_open", models.RemoteSessionEndDisconnected)
	if err != nil {
		t.Fatalf("close: %v", err)
	}
	if !changed {
		t.Fatal("the first close reported that it changed nothing")
	}
	if closed.Active || closed.EndedAt == nil || closed.EndReason != models.RemoteSessionEndDisconnected {
		t.Fatalf("closed session = %+v, want a stamped end", closed)
	}

	// A second close (socket teardown racing a sweep, say) must not report
	// another end: exactly one "it ended" event may be published per session.
	_, changedAgain, err := svc.Close(ctx, "sig_test_open", models.RemoteSessionEndServerRestart)
	if err != nil {
		t.Fatalf("second close: %v", err)
	}
	if changedAgain {
		t.Fatal("the second close also claimed to have ended the session")
	}
	reloaded, err := svc.Get(ctx, "sig_test_open")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if reloaded.EndReason != models.RemoteSessionEndDisconnected {
		t.Fatalf("the second close overwrote the first end reason with %q", reloaded.EndReason)
	}

	finished, err = svc.List(ctx, RemoteSessionFilter{UserID: 990201, Active: &ended})
	if err != nil {
		t.Fatalf("list ended again: %v", err)
	}
	if len(finished) != 1 || finished[0].Active {
		t.Fatalf("ended list = %+v, want the one finished session", finished)
	}
}

// A device that is not bound to an account still gets a row - the signaling side
// legitimately knows about devices the control plane has not bound - but it has
// no owner to show it to, and a restart has to close whatever the previous
// process left open.
func TestRemoteSessionUnboundDeviceAndRestartSweep(t *testing.T) {
	db := newRemoteSessionTestDB(t)
	svc := NewRemoteSessionService(db)
	ctx := context.Background()
	seedRemoteSessionDevice(t, db, "700000002", 990202)

	unbound, err := svc.Open(ctx, OpenRemoteSessionInput{SessionID: "sig_unbound", DeviceID: "700000009"})
	if err != nil {
		t.Fatalf("open on an unknown device: %v", err)
	}
	if unbound.UserID != 0 {
		t.Fatalf("an unknown device produced owner %d, want 0", unbound.UserID)
	}

	if _, err := svc.Open(ctx, OpenRemoteSessionInput{SessionID: "sig_restart", DeviceID: "700000002"}); err != nil {
		t.Fatalf("open second session: %v", err)
	}
	// The restart sweep is unconditional: every open row belongs to a process
	// that no longer exists, so the count is exactly what was left open.
	closed, err := svc.CloseOpenSessions(ctx, models.RemoteSessionEndServerRestart)
	if err != nil {
		t.Fatalf("close open sessions: %v", err)
	}
	if closed != 2 {
		t.Fatalf("restart sweep closed %d sessions, want the 2 that were open", closed)
	}

	restarted, err := svc.Get(ctx, "sig_restart")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if restarted.Active || restarted.EndReason != models.RemoteSessionEndServerRestart {
		t.Fatalf("restart sweep left %+v, want a server_restart end", restarted)
	}
	if restarted.EndedAt == nil || restarted.EndedAt.Before(restarted.StartedAt) {
		t.Fatalf("restart sweep produced an impossible interval: %+v", restarted)
	}

	// Nothing to close the second time.
	if again, err := svc.CloseOpenSessions(ctx, models.RemoteSessionEndServerRestart); err != nil || again != 0 {
		t.Fatalf("second restart sweep closed %d sessions (err=%v), want 0", again, err)
	}
}

// The list is the owner's view: another account's sessions must never appear in
// it, whatever device_id they pass.
func TestRemoteSessionListIsOwnerScoped(t *testing.T) {
	db := newRemoteSessionTestDB(t)
	svc := NewRemoteSessionService(db)
	ctx := context.Background()
	seedRemoteSessionDevice(t, db, "700000003", 990203)
	seedRemoteSessionDevice(t, db, "700000004", 990204)

	if _, err := svc.Open(ctx, OpenRemoteSessionInput{SessionID: "sig_mine", DeviceID: "700000003"}); err != nil {
		t.Fatalf("open mine: %v", err)
	}
	if _, err := svc.Open(ctx, OpenRemoteSessionInput{SessionID: "sig_theirs", DeviceID: "700000004"}); err != nil {
		t.Fatalf("open theirs: %v", err)
	}

	mine, err := svc.List(ctx, RemoteSessionFilter{UserID: 990203})
	if err != nil {
		t.Fatalf("list mine: %v", err)
	}
	if len(mine) != 1 || mine[0].SessionID != "sig_mine" {
		t.Fatalf("owner 990203 sees %+v, want only their own session", mine)
	}

	// Asking for somebody else's device by id yields nothing rather than a
	// filtered view of their sessions.
	crossed, err := svc.List(ctx, RemoteSessionFilter{UserID: 990203, DeviceID: "700000004"})
	if err != nil {
		t.Fatalf("list crossed: %v", err)
	}
	if len(crossed) != 0 {
		t.Fatalf("owner 990203 can see another account's device sessions: %+v", crossed)
	}

	// The device filter does work inside the owner's own set.
	own, err := svc.List(ctx, RemoteSessionFilter{UserID: 990203, DeviceID: "700000003"})
	if err != nil {
		t.Fatalf("list own device: %v", err)
	}
	if len(own) != 1 {
		t.Fatalf("own-device filter returned %+v", own)
	}

	if _, _, err := svc.Close(ctx, "sig_mine", models.RemoteSessionEndDisconnected); err != nil {
		t.Fatalf("close: %v", err)
	}
	after, err := svc.List(ctx, RemoteSessionFilter{UserID: 990203})
	if err != nil {
		t.Fatalf("list after close: %v", err)
	}
	if len(after) != 1 || after[0].Active || after[0].DurationSeconds < 0 {
		t.Fatalf("after close = %+v, want one finished row with a real duration", after)
	}
}
