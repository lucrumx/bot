// Package http provides HTTP handlers for user-related operations.
package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/lucrumx/bot/internal/users/services"
)

// Handler handles user-related HTTP requests.
type Handler struct {
	service *services.UsersService
}

// Create is a constructor for UsersHandler.
func Create(service *services.UsersService) *Handler {
	return &Handler{service: service}
}

// CreateUserRequest is DTO for creating a new user.
type CreateUserRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

// CreateUser handles POST /users requests.
func (h *Handler) CreateUser(c *gin.Context) {
	var userData CreateUserRequest

	if err := c.ShouldBindJSON(&userData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.service.CreateUser(userData.Email, userData.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, ToUserResponse(user))
}

// GetMe handles GET /users/me requests.
func (h *Handler) GetMe(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	user, err := h.service.GetByID(userID.(uint))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{})
		return
	}

	c.JSON(http.StatusOK, ToUserResponse(user))
}
