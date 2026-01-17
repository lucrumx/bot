// Package handlers contains HTTP handlers
package handlers

import (
	"log"
	"net/http"

	"errors"

	"github.com/gin-gonic/gin"
	"github.com/lucrumx/bot/internal/auth"
	"github.com/lucrumx/bot/internal/http/responses"
	"github.com/lucrumx/bot/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// UserHandler handles user-related HTTP requests.
type UserHandler struct {
	DB *gorm.DB
}

// NewUserHandler is a constructor for UserHandler.
func NewUserHandler(db *gorm.DB) *UserHandler {
	return &UserHandler{DB: db}
}

// CreateUserRequest is DTO for creating a new user.
type CreateUserRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

// LoginRequest is DTO for user login.
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// CreateUser handles the creation of a new user.
func (h *UserHandler) CreateUser(c *gin.Context) {
	var userData CreateUserRequest

	if err := c.ShouldBindJSON(&userData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(userData.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	user := models.User{
		Email:        userData.Email,
		PasswordHash: string(hashedBytes),
	}

	result := h.DB.Create(&user)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	c.JSON(http.StatusCreated, responses.ToUserResponse(&user))
}

// Login handles user login.
func (h *UserHandler) Login(c *gin.Context) {
	var loginData LoginRequest

	if err := c.ShouldBindJSON(&loginData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	result := h.DB.Where("email = ?", loginData.Email).First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		log.Printf("Database error: %v", result.Error)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(loginData.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	token, err := auth.GenerateJWT(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		log.Printf("Failed to generate token (login): %v", err)
		return
	}

	c.JSON(http.StatusCreated, responses.ToLoginResponse(token))
}
