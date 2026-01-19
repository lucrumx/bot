package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"github.com/lucrumx/bot/internal/testutils"
)

func setupRouter(db *gorm.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)

	r := gin.Default()

	handler := NewUserHandler(db)

	r.POST("/users", handler.CreateUser)

	return r
}

func TestCreateUser_Integration(t *testing.T) {
	db := testutils.SetupTestDB(t)
	testutils.ClearTables(db, "users")

	router := setupRouter(db)

	w := httptest.NewRecorder()

	userEmail := "test@test.com"

	body := bytes.NewBuffer([]byte(fmt.Sprintf(`{"email": "%s", "password": "some-password"}`, userEmail)))
	req, _ := http.NewRequest("POST", "/users", body)
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)

	assert.NoError(t, err, "Failed to parse JSON response")

	assert.Contains(t, response, "id")
	assert.Contains(t, response, "created_at")
	assert.Contains(t, response, "updated_at")

	assert.Equal(t, response["email"], userEmail)
}
