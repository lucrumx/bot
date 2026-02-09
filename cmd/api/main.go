// Package main implements the entry point for the API server.
package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/lucrumx/bot/internal/config"

	"github.com/lucrumx/bot/internal/middleware"

	"github.com/lucrumx/bot/internal/storage"

	authHandler "github.com/lucrumx/bot/internal/auth/http"
	authService "github.com/lucrumx/bot/internal/auth/services"
	usersHandler "github.com/lucrumx/bot/internal/users/http"
	userService "github.com/lucrumx/bot/internal/users/services"
)

func main() {
	log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).
		With().
		Timestamp().
		Logger()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("Error loading config")
	}

	db := storage.InitDB(cfg)

	// users
	usersRepo := userService.CreateUserRepo(db)
	usersSrv := userService.Create(usersRepo)
	usersH := usersHandler.Create(usersSrv)

	// auth
	authSrv := authService.Create(usersSrv, cfg)
	authH := authHandler.Create(authSrv)

	r := gin.Default()

	r.POST("/users", usersH.CreateUser)
	r.POST("/auth", authH.Auth)

	private := r.Group("/")
	private.Use(middleware.JwtAuth(cfg))
	{
		private.GET("/users/me", usersH.GetMe)
	}

	port := cfg.HTTP.HTTPServerPort
	log.Printf("Starting server on port %s", port)

	srv := &http.Server{
		Addr:    port,
		Handler: r,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal().Err(err).Msg("Failed to start server")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	log.Info().Msg("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("Failed to gracefully shutdown the server, server forced to shutdown")
	}

	log.Info().Msg("Server shut down successfully. Bye!")
}
