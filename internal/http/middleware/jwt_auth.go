// Package middleware provides HTTP middleware functions.
package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lucrumx/bot/internal/auth"
)

// JwtAuth is a middleware that checks for a valid JWT in the Authorization header.
func JwtAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader("Authorization")
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing Authorization token"})
			c.Abort()
			return
		}

		if tokenString[:7] == "Bearer " {
			tokenString = tokenString[7:]
		}

		claims, err := auth.ValidateJWT(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token or expired"})
			c.Abort()
			return
		}

		c.Set("user_id", claims.Sub)

		c.Next()
	}
}
