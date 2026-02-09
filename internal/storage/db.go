// Package storage provides functions to initialize and manage the database connection, run migrations.
package storage

import (
	"github.com/rs/zerolog/log"

	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/lucrumx/bot/internal/config"

	"github.com/lucrumx/bot/internal/models"
)

// InitDB initializes the database connection and runs migrations.
func InitDB(cfg *config.Config) *gorm.DB {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		cfg.Database.Host,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.DbName,
		cfg.Database.Port,
		cfg.Database.SslMode,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal().Err(err).Msg(
			"Failed to connect to database. Check your database connection settings in the config file.",
		)
	}

	if err := db.AutoMigrate(&models.User{}); err != nil {
		log.Fatal().Err(err).Msg("Failed to migrate database schema")
	}

	return db
}
