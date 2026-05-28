package tests

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Grivvus/ym/internal/api"
	"github.com/Grivvus/ym/internal/handlers"
	"github.com/Grivvus/ym/internal/repository"
	"github.com/Grivvus/ym/internal/service"
	"github.com/Grivvus/ym/internal/utils"
	"github.com/Grivvus/ym/tests/testenv"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
)

type IntegrationTestSuite struct {
	suite.Suite
	env         *testenv.Environment
	server      *httptest.Server
	client      *http.Client
	resetMailer *testPasswordResetMailer
}

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}

func (s *IntegrationTestSuite) SetupSuite() {
	testcontainers.SkipIfProviderIsNotHealthy(s.T())

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	env, err := testenv.Start(ctx)
	s.Require().NoError(err)
	s.env = env

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	queueNotificationChan := make(chan struct{}, 1)
	s.resetMailer = newTestPasswordResetMailer()
	passwordResetCfg := &utils.PasswordResetConfig{
		Enabled:         true,
		CodeSecret:      "integration-password-reset-secret",
		CodeTTL:         15 * time.Minute,
		ResendCooldown:  time.Minute,
		MaxAttempts:     5,
		CodeLength:      6,
		AcceptedMessage: "if an account with that email exists, a reset code has been sent",
	}

	authRepo := repository.NewAuthRepository(env.DB)
	playlistRepo := repository.NewPlaylistRepository(env.DB)
	userRepo := repository.NewUserRepository(env.DB)
	artistRepo := repository.NewArtistRepository(env.DB)
	albumRepo := repository.NewAlbumRepository(env.DB)
	trackRepo := repository.NewTrackRepository(env.DB)
	passwordResetRepo := repository.NewPasswordResetRepository(env.DB)

	authService := service.NewAuthService(authRepo, logger, &env.Config)
	passwordResetService := service.NewPasswordResetService(
		passwordResetRepo, logger, s.resetMailer, passwordResetCfg,
	)
	userService := service.NewUserService(userRepo, env.Storage, logger)
	albumService := service.NewAlbumService(albumRepo, artistRepo, trackRepo, env.Storage, logger)
	trackService := service.NewTrackService(
		trackRepo, userRepo, artistRepo, albumRepo, env.Storage, logger, queueNotificationChan,
	)
	playlistService := service.NewPlaylistService(playlistRepo, trackRepo, env.Storage, logger)
	artistService := service.NewArtistService(artistRepo, env.Storage, logger)
	backupService := service.NewBackupService(logger, env.Queries, env.Storage)

	rootHandler := handlers.NewRootHandler(
		logger,
		authService, passwordResetService, userService,
		albumService, artistService,
		trackService, playlistService,
		backupService,
	)

	r := chi.NewMux()
	r.Use(middleware.Recoverer)
	r.Use(handlers.AuthMiddleware(logger, []byte(env.Config.JWTSecret)))

	httpHandler := api.HandlerFromMux(rootHandler, r)
	s.server = httptest.NewServer(httpHandler)
	s.client = s.server.Client()
}

func (s *IntegrationTestSuite) TearDownSuite() {
	if s.server != nil {
		s.server.Close()
	}

	if s.env == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	s.Require().NoError(s.env.Close(ctx))
}

func (s *IntegrationTestSuite) SetupTest() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := s.env.DB.Exec(ctx, `
		TRUNCATE TABLE
			"password_reset_code",
			"transcoding_queue",
			"track_playlist",
			"track_album",
			"track",
			"playlist",
			"album",
			"artist",
			"user",
			"backup_status",
			"restore_status"
		RESTART IDENTITY CASCADE
	`)
	s.Require().NoError(err)
	s.resetMailer.Reset()
}
