package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestValidateJWT tests the generation and validation of a JWT, ensuring valid claims and no errors during the process.
func TestValidateJWT(t *testing.T) {
	t.Setenv("JWT_SECRET", "some-secret-key")
	userID := uint(123)
	validToken, _ := GenerateJWT(userID)

	tests := []struct {
		name        string
		tokenString string
		wantErr     bool
		wantUserID  uint
	}{
		{
			name:        "Valid token",
			tokenString: validToken,
			wantErr:     false,
			wantUserID:  userID,
		},
		{
			name:        "Invalidate token",
			tokenString: "invalid-token-string",
			wantErr:     true,
		},
		{
			name:        "Empty token",
			tokenString: "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := ValidateJWT(tt.tokenString)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantUserID, claims.Sub)
			}
		})
	}
}
