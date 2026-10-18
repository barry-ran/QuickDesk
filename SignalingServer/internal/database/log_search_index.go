package database

import (
	"context"
	"log"

	"gorm.io/gorm"
)

// The statements below are what make the shipped-log search indexable. They are
// deliberately separate and idempotent, because the two halves can fail
// independently on a managed Postgres:
//
//   - the trigram index accelerates the `q=` substring search that has always
//     existed, and needs the pg_trgm extension;
//   - the generated tsvector column plus its GIN index back the newer `match=`
//     full-text search, and needs nothing but a recent server.
//
// Neither is created by GORM's AutoMigrate (it manages columns it knows about,
// not expression indexes or generated columns), so they are ensured explicitly
// on startup. A deployment where either fails keeps working: the queries are
// still correct, they just fall back to a sequential scan, and the failure is
// logged once at boot rather than silently.
const (
	createTrigramExtension = `CREATE EXTENSION IF NOT EXISTS pg_trgm`
	createTrigramIndex     = `CREATE INDEX IF NOT EXISTS idx_instance_log_entries_line_trgm
		ON instance_log_entries USING gin (line gin_trgm_ops)`
	createSearchVectorColumn = `ALTER TABLE instance_log_entries
		ADD COLUMN IF NOT EXISTS line_tsv tsvector
		GENERATED ALWAYS AS (to_tsvector('simple', line)) STORED`
	createSearchVectorIndex = `CREATE INDEX IF NOT EXISTS idx_instance_log_entries_line_tsv
		ON instance_log_entries USING gin (line_tsv)`
)

// EnsureLogSearchIndexes creates the search indexes if they are not there yet.
// It reports what it managed to do, and never returns an error: a missing index
// is a performance problem, and refusing to start over one would turn it into
// an outage.
func EnsureLogSearchIndexes(ctx context.Context, db *gorm.DB) {
	if db == nil {
		return
	}
	// The extension has to exist before the operator class `gin_trgm_ops` can be
	// named, so the two are one unit.
	if err := db.WithContext(ctx).Exec(createTrigramExtension).Error; err != nil {
		log.Printf("Warning: pg_trgm is unavailable (%v); substring log search stays unindexed", err)
	} else if err := db.WithContext(ctx).Exec(createTrigramIndex).Error; err != nil {
		log.Printf("Warning: could not create the trigram index on instance_log_entries.line: %v", err)
	} else {
		// `q=` keeps its substring semantics; the index only makes it fast, so
		// the message says "substring", not "search".
		log.Println("Log search index: substring (q=) queries are index-backed by pg_trgm")
	}

	if err := db.WithContext(ctx).Exec(createSearchVectorColumn).Error; err != nil {
		log.Printf("Warning: could not add instance_log_entries.line_tsv (%v); match= queries would fail", err)
		return
	}
	if err := db.WithContext(ctx).Exec(createSearchVectorIndex).Error; err != nil {
		log.Printf("Warning: could not index instance_log_entries.line_tsv: %v", err)
		return
	}
	log.Println("Log search index: full-text (match=) queries are index-backed")
}
