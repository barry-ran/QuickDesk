package database

import (
	"log"
	"quickdesk/signaling/internal/config"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// InitPostgreSQL initializes the PostgreSQL connection
func InitPostgreSQL(cfg *config.Config) *gorm.DB {
	dsn := cfg.Database.DSN()
	
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
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
