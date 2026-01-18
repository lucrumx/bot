package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestValidateJWT tests the generation and validation of a JWT, ensuring valid claims and no errors during the process.
func TestValidateJWT(t *testing.T) {
	t.Setenv("JWT_SECRET", "some-secret-key")

	userID := uint(123)
	token, err := GenerateJWT(userID)

	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := ValidateJWT(token)

	assert.NoError(t, err)
	assert.Equal(t, userID, claims.Sub)
}
