// Package testutils provides utilities for testing.
package testutils

import (
	"gorm.io/gorm"

	"github.com/lucrumx/bot/internal/config"

	"github.com/lucrumx/bot/internal/storage"
)

// SetupTestDB initializes a test database connection.
func SetupTestDB() *gorm.DB {
	cfg := config.Config{
		Database: config.DatabaseConfig{
			SslMode:  "disable",
			Host:     "localhost",
			Port:     "5433",
			User:     "postgres",
			Password: "password",
			DbName:   "lucrum_bot_test",
		},
	}

	db := storage.InitDB(&cfg)

	return db
}

// ClearTables clears all tables in the given database.
func ClearTables(db *gorm.DB, tables ...string) {
	for _, table := range tables {
		db.Exec("TRUNCATE TABLE " + table + " RESTART IDENTITY CASCADE")
	}
}
