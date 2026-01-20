package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"github.com/lucrumx/bot/internal/auth/services"
	"github.com/lucrumx/bot/internal/models"
	usersService "github.com/lucrumx/bot/internal/users/services"
	"github.com/lucrumx/bot/internal/utils"
	"github.com/lucrumx/bot/internal/utils/testutils"
)

func setup(t *testing.T) (*gin.Engine, *gorm.DB) {
	db := testutils.SetupTestDB(t)
	testutils.ClearTables(db, "users")

	t.Setenv("JWT_SECRET", "some-secret-key")

	gin.SetMode(gin.TestMode)

	r := gin.Default()

	usersRepo := usersService.CreateUserRepo(db)
	usersSrv := usersService.Create(usersRepo)
	service := services.Create(usersSrv)
	handler := Create(service)

	r.POST("/auth", handler.Auth)

	return r, db
}

func createUser(email string, password string, db *gorm.DB) (*models.User, error) {
	passwordHash, _ := utils.GenPasswordHash(password)
	user := models.User{Email: email, PasswordHash: passwordHash}
	result := db.Create(&user)

	return &user, result.Error
}

func TestAuth_Integration(t *testing.T) {
	password := "123123123xxx"
	email := "some@example.com"

	var tests = []struct {
		name          string
		email         string
		password      string
		httpStatus    int
		shouldBeToken bool
	}{
		{name: "valid credentials", email: email, password: password, httpStatus: http.StatusCreated, shouldBeToken: true},
		{name: "wrong email", email: "wrong@email.com", password: password, httpStatus: http.StatusUnauthorized},
		{name: "invalid email", email: "no-domain@email", password: password, httpStatus: http.StatusBadRequest},
		{name: "wrong password", email: email, password: "000cv123123123xxx", httpStatus: http.StatusUnauthorized},
		{name: "invalid password (less then min len)", email: email, password: "000cv1", httpStatus: http.StatusBadRequest},
	}

	router, db := setup(t)

	_, _ = createUser(email, password, db)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := bytes.NewBuffer([]byte(fmt.Sprintf(`{"email": "%s", "password": "%s"}`, tt.email, tt.password)))

			w := testutils.DoHTTPRequest(router,
				"POST",
				"/auth",
				body.Bytes(), map[string]string{"Content-type": "application/json"},
			)

			assert.Equal(t, tt.httpStatus, w.Code)
			if tt.shouldBeToken {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err, "Failed to parse JSON response")
				assert.Contains(t, w.Body.String(), "token")
			}
		})
	}
}
