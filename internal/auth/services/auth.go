// Package services provides methods to authenticate users.
package services

import (
	"errors"
	"log"

	"golang.org/x/crypto/bcrypt"

	"github.com/lucrumx/bot/internal/models"
)

// UserFinder interface defines methods for finding users by email.
type UserFinder interface {
	GetByEmail(email string) (*models.User, error)
}

// AuthService provides methods to authenticate users.
type AuthService struct {
	users UserFinder
}

// Create is a constructor for AuthService.
func Create(users UserFinder) *AuthService {
	return &AuthService{users: users}
}

// Login authenticates a user and returns a JWT token if successful.
func (s *AuthService) Login(email string, password string) (string, error) {
	invalidCredentialsError := errors.New("invalid credentials")

	user, err := s.users.GetByEmail(email)
	if err != nil {
		return "", invalidCredentialsError
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", invalidCredentialsError
	}

	token, err := GenerateJWT(user.ID)
	if err != nil {
		log.Printf("Failed to generate jwt token: %v", err)
		return "", errors.New("failed to generate token")
	}

	return token, nil

}
