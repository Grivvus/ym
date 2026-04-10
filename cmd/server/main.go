package main

import (
	"context"
	"errors"
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
	"github.com/Grivvus/ym/internal/repository"
	"github.com/Grivvus/ym/internal/service"
	"github.com/Grivvus/ym/internal/storage"
	"github.com/Grivvus/ym/internal/transcoder"
	"github.com/Grivvus/ym/internal/utils"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	httpSwagger "github.com/swaggo/http-swagger"
)

func main() {
	var exitCode = 0
	defer func() {
		os.Exit(exitCode)
	}()
	ctx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt, os.Kill, syscall.SIGTERM,
	)
	defer cancel()

	logger := slog.New(slog.NewTextHandler(
		os.Stdout, &slog.HandlerOptions{AddSource: true},
	))

	err := godotenv.Load(".env.minio", ".env")
	if err != nil {
		logger.Error("Can't load .env file", "err", err)
		exitCode = 1
		return
	}

	cfg, err := utils.NewConfig()
	if err != nil {
		logger.Error("can't create config", "err", err)
		exitCode = 1
		return
	}

	pool, err := pgxpool.New(ctx, cfg.DBConnString())
	if err != nil {
		logger.Error("Can't create connection pool to a database", "err", err)
		exitCode = 1
		return
	}
	logger.Info("connection pool to the database was created")

	storageClient, err := storage.New(ctx, *cfg, logger)
	if err != nil {
		logger.Error("Can't create connection to a storage", "err", err)
		exitCode = 1
		return
	}
	logger.Info("connection to the storage was created")

	dbInst := db.New(pool)

	queueNotificationChan := make(chan struct{})

	transcoderRepo := repository.NewTranscodingQueueRepository(pool, dbInst)
	tcoder := transcoder.NewTranscoder(
		logger, storageClient, transcoderRepo, queueNotificationChan,
	)
	tcoder.StartListener(ctx)

	authService := service.NewAuthService(dbInst, logger, cfg)
	userService := service.NewUserService(dbInst, storageClient, logger)
	albumService := service.NewAlbumService(dbInst, storageClient, logger)
	playlistService := service.NewPlaylistService(dbInst, storageClient, logger)
	trackService := service.NewTrackService(dbInst, storageClient, logger, queueNotificationChan)
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
		var err error
		if cfg.TLSCertFile != "" && cfg.TLSKeyFile != "" {
			err = s.ListenAndServeTLS(cfg.TLSCertFile, cfg.TLSKeyFile)
		} else {
			err = s.ListenAndServe()
		}
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("listen failed", "err", err)
		}
	}()

	<-ctx.Done()
	logger.Info("shutdown server")

	timeoutCtx, timeoutCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer timeoutCancel()
	if err := s.Shutdown(timeoutCtx); err != nil {
		logger.Error("server shutdown failed", "err", err)
	}
	logger.Info("server exiting")
}
