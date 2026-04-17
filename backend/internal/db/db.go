package db

import (
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Open opens (or creates) the SQLite database at path and auto-migrates all models.
func Open(path string) (*gorm.DB, error) {
	database, err := gorm.Open(sqlite.Open(path), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, err
	}
	// Explicit column additions as a safety net when AutoMigrate silently
	// skips new columns on existing SQLite tables. These are no-ops if the
	// column already exists (SQLite returns "duplicate column name" which is
	// ignored because we don't check the result).
	database.Exec(`ALTER TABLE check_ins ADD COLUMN tag_received BOOLEAN NOT NULL DEFAULT false`)
	database.Exec(`ALTER TABLE check_ins ADD COLUMN registered_at DATETIME`)
	database.Exec(`ALTER TABLE check_ins ADD COLUMN last_notified_at DATETIME`)
	database.Exec(`ALTER TABLE check_ins ADD COLUMN checked_out_at DATETIME`)
	database.Exec(`ALTER TABLE check_ins ADD COLUMN is_guest BOOLEAN NOT NULL DEFAULT false`)
	database.Exec(`ALTER TABLE synced_persons ADD COLUMN is_guest BOOLEAN NOT NULL DEFAULT false`)

	if err := database.AutoMigrate(
		&CheckIn{},
		&PushSubscription{},
		&Setting{},
		&SyncedPerson{},
		&SyncedGroupMembership{},
		&SyncedRelationship{},
		&SyncedStaff{},
		&EventStats{},
	); err != nil {
		return nil, err
	}
	// One-time migration: records that were created before TagReceived was
	// introduced had Status="registered" to indicate a name tag was given.
	// Convert them to Status="pending" + TagReceived=true so the new model
	// correctly reflects their state.
	database.Exec(
		`UPDATE check_ins SET status = 'pending', tag_received = true
		 WHERE status = 'registered' AND tag_received = false`,
	)
	// Partial unique index so that ct_id=0 (used as a temporary placeholder
	// during the two-step guest creation) is allowed more than once.
	database.Exec(
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_synced_persons_ct_id
		 ON synced_persons(ct_id) WHERE ct_id != 0`,
	)
	return database, nil
}

// Today returns the current local date formatted as "YYYY-MM-DD".
func Today() string {
	return time.Now().Format("2006-01-02")
}
