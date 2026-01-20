package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	auth "github.com/lucrumx/bot/internal/auth/services"
)

func TestJwtAuth(t *testing.T) {
	t.Setenv("JWT_SECRET", "some-secret-key")

	userID := uint(123)

	token, _ := auth.GenerateJWT(userID)

	t.Run("Valid token", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/", nil)
		c.Request.Header.Set("Authorization", "Bearer "+token)

		JwtAuth()(c)

		assert.False(t, c.IsAborted(), "Middleware should not abort the request")
		val, _ := c.Get("user_id")
		assert.Equal(t, userID, val)
	})

	t.Run("Invalid token", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/", nil)
		c.Request.Header.Set("Authorization", "Bearer invalid-token")

		JwtAuth()(c)

		assert.True(t, c.IsAborted(), "Middleware should abort the request")
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}
