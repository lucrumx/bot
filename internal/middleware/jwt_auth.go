// Package middleware provides HTTP middleware functions.
package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/lucrumx/bot/internal/config"

	athService "github.com/lucrumx/bot/internal/auth/services"
)

// JwtAuth is a middleware that checks for a valid JWT in the Authorization header.
func JwtAuth(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		authSrv := athService.Create(nil, cfg)

		tokenString := c.GetHeader("Authorization")
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing Authorization token"})
			c.Abort()
			return
		}

		if tokenString[:7] == "Bearer " {
			tokenString = tokenString[7:]
		}

		if len(tokenString) == 0 {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Empty Authorization token"})
			c.Abort()
			return
		}

		claims, err := authSrv.ValidateJWT(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token or expired"})
			c.Abort()
			return
		}

		c.Set("user_id", claims.Sub)

		c.Next()
	}
}
