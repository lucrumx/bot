// Package main implements the entry point for the API server.
package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/lucrumx/bot/internal/http/handlers"
	"github.com/lucrumx/bot/internal/http/middleware"
	"github.com/lucrumx/bot/internal/storage"
	"github.com/lucrumx/bot/internal/utils"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, hope environment variables are set")
	}

	db := storage.InitDB()

	r := gin.Default()
	userHandler := handlers.NewUserHandler(db)
	r.POST("/users", userHandler.CreateUser)
	r.POST("/auth/login", userHandler.Login)

	private := r.Group("/")
	private.Use(middleware.JwtAuth())
	{
		private.GET("/users/me", userHandler.GetMe)
	}

	port := utils.GetEnv("HTTP_SERVER_PORT", ":8080")
	log.Printf("Starting server on port %s", port)

	if err := r.Run(port); err != nil {
		log.Fatalf("Port %s already in use: %v", port, err)
	}
}
