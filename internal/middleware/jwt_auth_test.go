package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/lucrumx/bot/internal/config"

	auth "github.com/lucrumx/bot/internal/auth/services"
)

func TestJwtAuth(t *testing.T) {
	cfg := &config.Config{
		HTTP: config.HTTPConfig{
			Auth: config.AuthConfig{
				JwtSecret:    "some-secret-key",
				JwtExpiresIn: 24,
			},
		},
	}

	authSrv := auth.Create(nil, cfg)

	userID := uint(123)

	token, _ := authSrv.GenerateJWT(userID)

	t.Run("Valid token", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/", nil)
		c.Request.Header.Set("Authorization", "Bearer "+token)

		JwtAuth(cfg)(c)

		assert.False(t, c.IsAborted(), "Middleware should not abort the request")
		val, _ := c.Get("user_id")
		assert.Equal(t, userID, val)
	})

	t.Run("Invalid token", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/", nil)
		c.Request.Header.Set("Authorization", "Bearer invalid-token")

		JwtAuth(cfg)(c)

		assert.True(t, c.IsAborted(), "Middleware should abort the request")
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}
