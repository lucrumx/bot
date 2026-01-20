package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	usersService "github.com/lucrumx/bot/internal/users/services"
	"github.com/lucrumx/bot/internal/utils/testutils"
)

func setupRouter(t *testing.T) *gin.Engine {
	db := testutils.SetupTestDB(t)
	testutils.ClearTables(db, "users")

	gin.SetMode(gin.TestMode)

	r := gin.Default()

	usersRepo := usersService.CreateUserRepo(db)
	usersSrv := usersService.Create(usersRepo)
	handler := Create(usersSrv)

	r.POST("/users", handler.CreateUser)

	return r
}

func TestCreateUser_Integration(t *testing.T) {
	router := setupRouter(t)

	userEmail := "test@test.com"

	w := testutils.DoHTTPRequest(router, "POST", "/users",
		bytes.NewBuffer([]byte(fmt.Sprintf(`{"email": "%s", "password": "some-password"}`, userEmail))).Bytes(),
		map[string]string{"Content-type": "application/json"},
	)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)

	assert.NoError(t, err, "Failed to parse JSON response")

	assert.Contains(t, response, "id")
	assert.Contains(t, response, "created_at")
	assert.Contains(t, response, "updated_at")

	assert.Equal(t, response["email"], userEmail)
}
