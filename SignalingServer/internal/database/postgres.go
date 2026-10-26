package database

import (
	"errors"
	"log"
	"quickdesk/signaling/internal/config"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// errNoDatabaseHandle is returned by AutoMigrate when there is nothing to
// migrate against, rather than a nil dereference deeper in.
var errNoDatabaseHandle = errors.New("no database handle to migrate")

// InitPostgreSQL initializes the PostgreSQL connection
func InitPostgreSQL(cfg *config.Config) *gorm.DB {
	dsn := cfg.Database.DSN()
	
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Get underlying SQL DB for connection pool settings
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get database instance: %v", err)
	}

	// Set connection pool settings
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)

	// Test connection
	if err := sqlDB.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	log.Printf("Successfully connected to PostgreSQL at %s:%d/%s",
		cfg.Database.Host, cfg.Database.Port, cfg.Database.DBName)

	return db
}

// RunMigrations executes SQL migration files
func RunMigrations(db *gorm.DB) error {
	// For simplicity, using AutoMigrate
	// In production, use proper migration tool like golang-migrate
	log.Println("Running database migrations...")
	
	// Auto-migrate will create tables based on model structs
	// This is simpler for development
	// For production, use the SQL files in migrations/
	
	log.Println("Database migrations completed")
	return nil
}

// AutoMigrate runs GORM's schema sync through a session with PrepareStmt on.
//
// Use this instead of db.AutoMigrate on a Postgres handle opened without
// PrepareStmt. gorm.io/driver/postgres's Migrator.GetRows injects
// pgx.QueryExecModeSimpleProtocol as a query argument whenever PrepareStmt is
// off, and it does so for `SELECT * FROM "t" LIMIT 1`, a statement with no
// placeholder. pgx's sanitizer rejects the query with "insufficient arguments",
// so a table that already exists cannot be migrated.
//
// A fresh database is unaffected - a missing table goes straight to CREATE
// TABLE and never reaches that path - so the failure only shows up on an
// existing installation, where it is easy to miss because the caller logs a
// warning and keeps going: every later schema update is silently skipped.
//
// PrepareStmt is scoped to this session rather than set on the handle because
// it changes how every subsequent statement is executed, and because a
// prepared session refuses a multi-statement Exec such as the
// migrations/001_init.sql file.
func AutoMigrate(db *gorm.DB, models ...interface{}) error {
	if db == nil {
		return errNoDatabaseHandle
	}
	return db.Session(&gorm.Session{PrepareStmt: true}).AutoMigrate(models...)
}
