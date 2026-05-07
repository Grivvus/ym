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
	"github.com/Grivvus/ym/internal/mailer"
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

	for _, path := range []string{".env", ".env.minio"} {
		err := godotenv.Load(path)
		if err == nil || os.IsNotExist(err) {
			continue
		}

		logger.Warn("can't load env file, continuing with process environment", "path", path, "err", err)
	}

	cfg, err := utils.NewConfig()
	if err != nil {
		logger.Error("can't create config", "err", err)
		exitCode = 1
		return
	}
	smtpCfg, err := utils.NewSMTPConfig()
	if err != nil {
		logger.Error("can't create smtp config", "err", err)
		exitCode = 1
		return
	}
	passwordResetCfg, err := utils.NewPasswordResetConfig()
	if err != nil {
		logger.Error("can't create password reset config", "err", err)
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

	var passwordResetMailer mailer.Mailer
	if passwordResetCfg.Enabled {
		validateErr := passwordResetCfg.Validate()
		switch {
		case validateErr != nil:
			logger.Warn("password reset is disabled due to invalid config", "err", validateErr)
		case !smtpCfg.IsConfigured():
			logger.Warn("password reset is disabled because SMTP config is incomplete")
		default:
			smtpMailer, err := mailer.NewSMTPMailer(*smtpCfg, logger)
			if err != nil {
				logger.Warn("password reset is disabled because SMTP config is invalid", "err", err)
			} else {
				passwordResetMailer = smtpMailer
			}
		}
	}

	queueNotificationChan := make(chan struct{})

	transcoderRepo := repository.NewTranscodingQueueRepository(pool, dbInst)
	tcoder := transcoder.NewTranscoder(
		logger, storageClient, transcoderRepo, queueNotificationChan,
	)
	tcoder.StartListener(ctx)

	playlistRepo := repository.NewPlaylistRepository(pool)
	authRepo := repository.NewAuthRepository(pool)
	userRepo := repository.NewUserRepository(pool)
	artistRepo := repository.NewArtistRepository(pool)

	authService := service.NewAuthService(authRepo, logger, cfg)
	passwordResetService := service.NewPasswordResetService(
		dbInst, logger, passwordResetMailer, passwordResetCfg,
	)
	userService := service.NewUserService(userRepo, storageClient, logger)
	albumService := service.NewAlbumService(dbInst, storageClient, logger)
	playlistService := service.NewPlaylistService(playlistRepo, storageClient, logger)
	trackService := service.NewTrackService(dbInst, storageClient, logger, queueNotificationChan)
	artistService := service.NewArtistService(artistRepo, storageClient, logger)
	backupService := service.NewBackupService(logger, dbInst, storageClient)

	var server api.ServerInterface = handlers.NewRootHandler(
		logger,
		authService, passwordResetService, userService,
		albumService, artistService,
		trackService, playlistService,
		backupService,
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
		err := s.ListenAndServe()
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
