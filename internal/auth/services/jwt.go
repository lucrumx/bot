package services

import (
	"errors"
	"fmt"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
)

// Claims represent the JWT claims.
type Claims struct {
	Sub uint `json:"sub"`
	jwt.RegisteredClaims
}

// GenerateJWT generates a JWT for the given user ID.
func (a *AuthService) GenerateJWT(userID uint) (string, error) {
	exp := a.cfg.HTTP.Auth.JwtExpiresIn

	payload := Claims{
		Sub: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(exp) * time.Hour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)

	return token.SignedString([]byte(a.cfg.HTTP.Auth.JwtSecret))
}

// ValidateJWT validates the given JWT and returns the claims if valid.
func (a *AuthService) ValidateJWT(tokenString string) (*Claims, error) {
	secret := a.cfg.HTTP.Auth.JwtSecret

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

	return nil, errors.New("invalid token claims")
}
