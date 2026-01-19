// Package testutils provides utilities for testing.
package testutils

import (
	"testing"

	"github.com/lucrumx/bot/internal/storage"
	"gorm.io/gorm"
)

// SetupTestDB initializes a test database connection.
func SetupTestDB(t *testing.T) *gorm.DB {
	t.Setenv("DB_HOST", "localhost")
	t.Setenv("DB_USER", "postgres")
	t.Setenv("DB_PASSWORD", "password")
	t.Setenv("DB_NAME", "lucrum_bot_test")
	t.Setenv("DB_PORT", "5433")
	t.Setenv("DB_SSLMODE", "disable")

	db := storage.InitDB()

	return db
}

// ClearTables clears all tables in the given database.
func ClearTables(db *gorm.DB, tables ...string) {
	for _, table := range tables {
		db.Exec("TRUNCATE TABLE " + table + " RESTART IDENTITY CASCADE")
	}
}
