package services

import (
	"gorm.io/gorm"

	"github.com/lucrumx/bot/internal/models"
)

// UserRepo provides methods to interact with the users table in the database.
type UserRepo interface {
	Create(*models.User) error
	GetByID(uint) (*models.User, error)
	GetByEmail(string) (*models.User, error)
}

// GormUserRepo provides methods to interact with the users table in the database.
type GormUserRepo struct {
	db *gorm.DB
}

// CreateUserRepo is a constructor for UserRepo.
func CreateUserRepo(db *gorm.DB) *GormUserRepo {
	return &GormUserRepo{db: db}
}

// Create creates a new user in the database.
func (r *GormUserRepo) Create(user *models.User) error {
	return r.db.Create(user).Error
}

// GetByID retrieves a user by their ID from the database.
func (r *GormUserRepo) GetByID(id uint) (*models.User, error) {
	var user models.User
	if err := r.db.Where(`id = ? AND active = true`, id).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByEmail retrieves a user by their email address from the database.
func (r *GormUserRepo) GetByEmail(email string) (*models.User, error) {
	var user models.User
	if err := r.db.Where(`email = ? AND active = true`, email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}
