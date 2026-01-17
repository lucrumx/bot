// Package models contains the data structures (entities) used for database operations.
package models

import "gorm.io/gorm"

// User represents a registered user in the system.
type User struct {
	gorm.Model
	Email        string `gorm:"uniqueIndex;not null"`
	PasswordHash string `gorm:"not null"`
	Active       bool   `gorm:"default:true"`
}
