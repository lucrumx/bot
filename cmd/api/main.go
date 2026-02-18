// Package main implements the entry point for the API server.
package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"path"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/lucrumx/bot/internal/config"
	"github.com/lucrumx/bot/internal/exchange/arbitragebot"
	"github.com/lucrumx/bot/internal/ui"

	"github.com/lucrumx/bot/internal/middleware"

	"github.com/lucrumx/bot/internal/storage"

	authHandler "github.com/lucrumx/bot/internal/auth/http"
	authService "github.com/lucrumx/bot/internal/auth/services"
	usersHandler "github.com/lucrumx/bot/internal/users/http"
	userService "github.com/lucrumx/bot/internal/users/services"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	logger := log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	cfg, err := config.Load(logger)
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

	//arbitrage
	arbitrageSpreadRepo := arbitragebot.NewRepository(db)
	arbitrageH := arbitragebot.NewHTTPHandlers(arbitrageSpreadRepo)

	r := gin.Default()
	api := r.Group("/api")
	{

		api.POST("/users", usersH.CreateUser)
		api.POST("/auth", authH.Auth)

		private := api.Group("/")
		private.Use(middleware.JwtAuth(cfg))
		{
			private.GET("/users/me", usersH.GetMe)
			//
			private.GET("/arbitrage-spreads", arbitrageH.GetSpreadsHandler)
		}
	}

	feFs, err := ui.GetFileSystem()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to get frontend filesystem")
	}
	fileServer := http.FileServer(feFs)
	r.NoRoute(func(c *gin.Context) {
		requestPath := c.Request.URL.Path
		if strings.HasPrefix(requestPath, "/api/") {
			// for routes like /api/something that are not defined, return 404 instead of serving index.html
			c.Status(http.StatusNotFound)
			return
		}

		assetPath := resolveFrontendAssetPath(feFs, requestPath)
		c.Request.URL.Path = assetPath
		fileServer.ServeHTTP(c.Writer, c.Request)
	})

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

func resolveFrontendAssetPath(staticFS http.FileSystem, requestPath string) string {
	cleanPath := path.Clean("/" + requestPath)

	if strings.Contains(path.Base(cleanPath), ".") {
		return cleanPath
	}

	if cleanPath == "/" {
		return "/"
	}

	if fileExists(staticFS, cleanPath) {
		return cleanPath
	}

	dirIndex := path.Join(cleanPath, "index.html")
	if fileExists(staticFS, dirIndex) {
		if strings.HasSuffix(cleanPath, "/") {
			return cleanPath
		}
		return cleanPath + "/"
	}

	return "/"
}

func fileExists(staticFS http.FileSystem, filePath string) bool {
	file, err := staticFS.Open(filePath)
	if err != nil {
		return false
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Warn().Err(err).Msg("Failed to close file")
		}
	}()

	stat, err := file.Stat()
	if err != nil {
		return false
	}

	return !stat.IsDir()
}
