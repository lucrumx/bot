// Package storage provides functions to initialize and manage the database connection, run migrations.
package storage

import (
	"log"

	"fmt"

	"github.com/lucrumx/bot/internal/models"
	"github.com/lucrumx/bot/internal/utils"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// InitDB initializes the database connection and runs migrations.
func InitDB() *gorm.DB {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		utils.GetEnv("DB_HOST", ""),
		utils.GetEnv("DB_USER", ""),
		utils.GetEnv("DB_PASSWORD", ""),
		utils.GetEnv("DB_NAME", ""),
		utils.GetEnv("DB_PORT", ""),
		utils.GetEnv("DB_SSLMODE", ""),
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&models.User{}); err != nil {
		log.Fatalf("Failed to migrate database schema: %v", err)
	}

	return db
}
