package database

import (
	"errors"
	"testing"
)

// AutoMigrate has to route the migration through a session that has
// PrepareStmt on, and it has to refuse a nil handle instead of panicking.
//
// The PrepareStmt part is the fix: gorm.io/driver/postgres's Migrator.GetRows
// injects pgx.QueryExecModeSimpleProtocol as a query argument whenever
// PrepareStmt is off, and it does so for `SELECT * FROM "t" LIMIT 1`, a
// statement with no placeholder. pgx's sanitizer rejects that with
// "insufficient arguments", so a table that already exists cannot be migrated.
// A fresh database never reaches that path, which is why the failure hides on
// an existing installation.
func TestAutoMigrateRejectsANilHandle(t *testing.T) {
	if err := AutoMigrate(nil); !errors.Is(err, errNoDatabaseHandle) {
		t.Fatalf("AutoMigrate(nil) = %v, want errNoDatabaseHandle", err)
	}
}
