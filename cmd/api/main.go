// Package main implements the entry point for the API server.
package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"github.com/lucrumx/bot/internal/middleware"

	"github.com/lucrumx/bot/internal/storage"

	authHandler "github.com/lucrumx/bot/internal/auth/http"
	authService "github.com/lucrumx/bot/internal/auth/services"
	usersHandler "github.com/lucrumx/bot/internal/users/http"
	userService "github.com/lucrumx/bot/internal/users/services"

	"github.com/lucrumx/bot/internal/utils"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, hope environment variables are set")
	}

	db := storage.InitDB()

	// users
	usersRepo := userService.CreateUserRepo(db)
	usersSrv := userService.Create(usersRepo)
	usersH := usersHandler.Create(usersSrv)

	// auth
	authSrv := authService.Create(usersSrv)
	authH := authHandler.Create(authSrv)

	r := gin.Default()

	r.POST("/users", usersH.CreateUser)
	r.POST("/auth", authH.Auth)

	private := r.Group("/")
	private.Use(middleware.JwtAuth())
	{
		private.GET("/users/me", usersH.GetMe)
	}

	port := utils.GetEnv("HTTP_SERVER_PORT", ":8080")
	log.Printf("Starting server on port %s", port)

	if err := r.Run(port); err != nil {
		log.Fatalf("Port %s already in use: %v", port, err)
	}
}
