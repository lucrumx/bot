// Package auth provides functionality for generating JSON Web Tokens (JWT) for user authentication.
package auth

import (
	"log"
	"strconv"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/lucrumx/bot/internal/utils"
)

// Claims represents the JWT claims.
type Claims struct {
	Sub uint `json:"sub"`
	jwt.RegisteredClaims
}

// GenerateJWT generates a JWT for the given user ID.
func GenerateJWT(userID uint) (string, error) {
	exp, err := strconv.Atoi(utils.GetEnv("JWT_EXPIRES_IN", "24"))
	if err != nil {
		log.Printf("Cannot get expiration of jwt token: %v", err)
		return "", err
	}

	payload := Claims{
		Sub: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(exp) * time.Hour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)

	return token.SignedString([]byte(utils.GetEnv("JWT_SECRET", "")))
}
