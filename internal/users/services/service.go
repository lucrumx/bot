// Package services provide methods to manage user-related operations in the database.
package services

import (
	"github.com/lucrumx/bot/internal/models"
	"github.com/lucrumx/bot/internal/utils"
)

// UsersService provides methods to manage user-related operations in the database.
type UsersService struct {
	repo UserRepo
}

// Create is a constructor for UsersService.
func Create(repo UserRepo) *UsersService {
	return &UsersService{repo: repo}
}

// CreateUser creates a new user in the database.
func (s *UsersService) CreateUser(email string, password string) (*models.User, error) {
	passwordHash, err := utils.GenPasswordHash(password)
	if err != nil {
		return nil, err
	}

	user := models.User{
		Email:        email,
		PasswordHash: passwordHash,
		Active:       true,
	}

	if err := s.repo.Create(&user); err != nil {
		return nil, err
	}

	return &user, nil
}

// GetByEmail retrieves a user from the database by their email address. Returns the user or an error if not found.
func (s *UsersService) GetByEmail(email string) (*models.User, error) {
	user, err := s.repo.GetByEmail(email)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// GetByID retrieves a user from the database by their email address. Returns the user or an error if not found.
func (s *UsersService) GetByID(id uint) (*models.User, error) {
	user, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	return user, nil
}
