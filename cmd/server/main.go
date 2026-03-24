package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Grivvus/ym/internal/api"
	"github.com/Grivvus/ym/internal/db"
	"github.com/Grivvus/ym/internal/handlers"
	"github.com/Grivvus/ym/internal/service"
	"github.com/Grivvus/ym/internal/storage"
	"github.com/Grivvus/ym/internal/utils"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	httpSwagger "github.com/swaggo/http-swagger"
)

func main() {
	logger := slog.New(slog.NewTextHandler(
		os.Stdout, &slog.HandlerOptions{AddSource: true},
	))

	err := godotenv.Load(".env.minio", ".env")
	if err != nil {
		logger.Error("Can't load .env file", "err", err)
		os.Exit(1)
	}

	cfg, err := utils.NewConfig()
	if err != nil {
		logger.Error("can't create config", "err", err)
		os.Exit(1)
	}

	pool, err := pgxpool.New(context.Background(), cfg.DBConnString())
	if err != nil {
		logger.Error("Can't create connection pool to a database", "err", err)
		os.Exit(1)
	}
	logger.Info("connection pool to the database was created")

	storageClient, err := storage.New(context.Background(), *cfg, logger)
	if err != nil {
		logger.Error("Can't create connection to a storage", "err", err)
		os.Exit(1)
	}
	logger.Info("connection to the storage was created")

	dbInst := db.New(pool)

	authService := service.NewAuthService(dbInst, logger, cfg)
	userService := service.NewUserService(dbInst, storageClient, logger)
	albumService := service.NewAlbumService(dbInst, storageClient, logger)
	playlistService := service.NewPlaylistService(dbInst, storageClient, logger)
	trackService := service.NewTrackService(dbInst, storageClient, logger)
	artistService := service.NewArtistService(dbInst, storageClient, logger)

	var server api.ServerInterface = handlers.NewRootHandler(
		logger,
		authService, userService,
		albumService, artistService,
		trackService, playlistService,
	)

	r := chi.NewMux()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(handlers.AuthMiddleware(logger, []byte(cfg.JWTSecret)))

	/* swagger-related routes */
	r.Get("/openapi.yml", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "/api/openapi.yml")
	})
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/openapi.yml"),
	))
	/* swagger-related routes end */

	h := api.HandlerFromMux(server, r)

	s := &http.Server{
		Addr:    fmt.Sprintf("%v:%v", cfg.Host, cfg.Port),
		Handler: h,
	}

	logger.Info("server was started on", "port", cfg.Port)

	go func() {
		if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("listen failed", "err", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server with
	// a timeout of 5 seconds.
	quit := make(chan os.Signal, 1)
	// kill (no params) by default sends syscall.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall.SIGKILL but can't be caught, so don't need add it
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("shutdown server")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.Shutdown(ctx); err != nil {
		logger.Error("server shutdown failed", "err", err)
	}
	logger.Info("server exiting")
}
