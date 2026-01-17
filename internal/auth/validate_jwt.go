package auth

import (
	"errors"
	"fmt"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/lucrumx/bot/internal/utils"
)

// ValidateJWT validates the given JWT and returns the claims if valid.
func ValidateJWT(tokenString string) (*Claims, error) {
	secret := utils.GetEnv("JWT_SECRET", "")

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("Invalid token claims")
}
