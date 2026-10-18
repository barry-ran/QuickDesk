package service

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"quickdesk/signaling/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// auditFilterTestDB needs the users table as well as audit_logs: the username
// filter resolves an end-user name through it.
func auditFilterTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("QD_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set QD_TEST_DATABASE_URL to a throwaway Postgres to run the audit-filter tests")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	// The scratch Postgres has a connection limit and this package opens a pool
	// per test, so the pool is closed with the test rather than left idle.
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	})
	if err := db.AutoMigrate(&models.AuditLog{}, &models.User{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := db.Exec("DELETE FROM audit_logs").Error; err != nil {
		t.Fatalf("clear audit logs: %v", err)
	}
	return db
}

// "What did wei do" has to work whether the name belongs to an administrator or
// to an end user, without the caller knowing which table the account lives in.
func TestAuditListFiltersByActingUsername(t *testing.T) {
	db := auditFilterTestDB(t)
	svc := NewAuditService(db)
	ctx := context.Background()

	// The scratch database is shared with the other Postgres-backed tests, so
	// this one names its own accounts and removes them again instead of
	// emptying the users table (devices reference it).
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	filter := "weiming-" + suffix
	user := models.User{Username: filter, Password: "x"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	other := models.User{Username: "someone-else-" + suffix, Password: "x"}
	if err := db.Create(&other).Error; err != nil {
		t.Fatalf("seed other user: %v", err)
	}
	// Resource ids are unique per run as well: the scratch database is shared,
	// and a leftover row from an interrupted run must not be able to change
	// what this test counts.
	ownID := "own-" + suffix
	otherID := "other-" + suffix
	adminID := "admin-" + suffix
	rootID := "root-" + suffix
	t.Cleanup(func() {
		db.Exec("DELETE FROM audit_logs WHERE resource_id IN ?",
			[]string{ownID, otherID, adminID, rootID})
		if err := db.Delete(&models.User{}, []uint{user.ID, other.ID}).Error; err != nil {
			t.Logf("cleanup: could not delete the seeded users: %v", err)
		}
	})

	rows := []models.AuditLog{
		{ActorUserID: user.ID, Action: "recording.delete", ResourceType: "session_recording", ResourceID: ownID},
		{ActorUserID: other.ID, Action: "recording.delete", ResourceType: "session_recording", ResourceID: otherID},
		{AdminID: 9, AdminUsername: filter + "-admin", Action: "user.disable", ResourceType: "user", ResourceID: adminID},
		{AdminID: 8, AdminUsername: "root", Action: "user.disable", ResourceType: "user", ResourceID: rootID},
	}
	for i := range rows {
		if err := db.Create(&rows[i]).Error; err != nil {
			t.Fatalf("seed audit row: %v", err)
		}
	}

	got, total, err := svc.List(ctx, AuditListParams{Username: filter, Limit: 50})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 2 || len(got) != 2 {
		t.Fatalf("total=%d rows=%d, want the user's row and the admin name that starts with it", total, len(got))
	}
	for _, row := range got {
		if row.ResourceID == otherID || row.ResourceID == rootID {
			t.Fatalf("row %s does not belong to the filter", row.ResourceID)
		}
	}

	// A name that matches nothing must return nothing rather than everything:
	// the trap a filter like this falls into when an empty resolution is passed
	// through as "no filter".
	_, total, err = svc.List(ctx, AuditListParams{Username: "nobody-by-that-name-" + suffix, Limit: 50})
	if err != nil {
		t.Fatalf("list with a miss: %v", err)
	}
	if total != 0 {
		t.Fatalf("a miss matched %d rows", total)
	}

	// The narrow filters still work together with it.
	got, total, err = svc.List(ctx, AuditListParams{Username: filter, Action: "user.disable", Limit: 50})
	if err != nil {
		t.Fatalf("list with action: %v", err)
	}
	if total != 1 || len(got) != 1 || got[0].ResourceID != adminID {
		t.Fatalf("username + action returned %d rows (%+v), want just the admin row", total, got)
	}
}
