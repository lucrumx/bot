package services

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"

	"github.com/lucrumx/bot/internal/utils"
)

// Claims represent the JWT claims.
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
