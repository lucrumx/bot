// Package http provides HTTP handlers for auth-related operations.
package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Authenticator represents an interface for handling user authentication processes, such as login operations.
type Authenticator interface {
	Login(email string, password string) (string, error)
}

// AuthHandler handles authentication-related HTTP requests and delegates authentication operations to AuthService.
type AuthHandler struct {
	services Authenticator
}

// Create is a constructor for AuthHandler.
func Create(services Authenticator) *AuthHandler {
	return &AuthHandler{services: services}
}

// LoginRequest is DTO for user login.
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

// Auth handles POST /auth/login requests.
func (h *AuthHandler) Auth(c *gin.Context) {
	var data LoginRequest

	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, err := h.services.Login(data.Email, data.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
	}

	c.JSON(http.StatusCreated, ToLoginResponse(token))
}
