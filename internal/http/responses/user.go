// Package responses provides structures and functions for creating HTTP responses.
package responses

import (
	"time"

	"github.com/lucrumx/bot/internal/models"
)

// UserResponse represents the data returned in response to user-related API calls.
type UserResponse struct {
	ID        uint      `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ToUserResponse converts a User model to a UserResponse.
func ToUserResponse(user *models.User) UserResponse {
	return UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}
