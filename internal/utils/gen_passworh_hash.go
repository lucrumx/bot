package utils

import "golang.org/x/crypto/bcrypt"

// GenPasswordHash generates a hash from the given password.
func GenPasswordHash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}
