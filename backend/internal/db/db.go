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
	if err := database.AutoMigrate(
		&CheckIn{},
		&Setting{},
		&SyncedPerson{},
		&SyncedGroupMembership{},
		&SyncedRelationship{},
	); err != nil {
		return nil, err
	}
	return database, nil
}

// Today returns the current local date formatted as "YYYY-MM-DD".
func Today() string {
	return time.Now().Format("2006-01-02")
}
