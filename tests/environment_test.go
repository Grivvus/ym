package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Grivvus/ym/internal/api"
	"github.com/Grivvus/ym/internal/handlers"
	"github.com/Grivvus/ym/internal/service"
	"github.com/Grivvus/ym/tests/testenv"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
)

type IntegrationTestSuite struct {
	suite.Suite
	env    *testenv.Environment
	server *httptest.Server
	client *http.Client
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

	authService := service.NewAuthService(env.Queries, logger, &env.Config)
	userService := service.NewUserService(env.Queries, env.Storage, logger)
	albumService := service.NewAlbumService(env.Queries, env.Storage, logger)
	playlistService := service.NewPlaylistService(env.Queries, env.Storage, logger)
	trackService := service.NewTrackService(env.Queries, env.Storage, logger, queueNotificationChan)
	artistService := service.NewArtistService(env.Queries, env.Storage, logger)

	rootHandler := handlers.NewRootHandler(
		logger,
		authService, userService,
		albumService, artistService,
		trackService, playlistService,
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

	_, err := s.env.DB.Exec(ctx, `TRUNCATE TABLE "user" RESTART IDENTITY CASCADE`)
	s.Require().NoError(err)
}

func (s *IntegrationTestSuite) TestEnvironmentBootstrapsDatabaseAndStorage() {
	env := s.env

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var exists bool
	err := env.DB.QueryRow(
		ctx,
		`SELECT EXISTS (
			SELECT 1
			FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = 'user'
		)`,
	).Scan(&exists)
	s.Require().NoError(err)
	s.True(exists)

	const objectID = "integration-smoke-track"

	payload := []byte("integration-smoke-payload")
	s.Require().NoError(env.Storage.PutTrack(ctx, objectID, bytes.NewReader(payload), int64(len(payload))))
	s.T().Cleanup(func() {
		s.Require().NoError(env.Storage.RemoveTrack(context.Background(), objectID))
	})

	reader, err := env.Storage.GetTrack(ctx, objectID)
	s.Require().NoError(err)
	s.T().Cleanup(func() {
		s.Require().NoError(reader.Close())
	})

	actual, err := io.ReadAll(reader)
	s.Require().NoError(err)
	s.Equal(payload, actual)
}

func (s *IntegrationTestSuite) TestRegister_FirstUserBecomesSuperuser() {
	resp := s.registerUser(api.UserAuth{
		Username: "first-user",
		Password: "password-1",
	})

	s.Equal(http.StatusCreated, resp.StatusCode)
	s.NotZero(resp.Body.UserId)
	s.NotEmpty(resp.Body.AccessToken)
	s.NotEmpty(resp.Body.RefreshToken)
	s.Equal("bearer", resp.Body.TokenType)
	s.True(s.userIsSuperuser(resp.Body.UserId))
}

func (s *IntegrationTestSuite) TestRegister_SubsequentUserDoesNotBecomeSuperuser() {
	firstResp := s.registerUser(api.UserAuth{
		Username: "first-user",
		Password: "password-1",
	})
	secondResp := s.registerUser(api.UserAuth{
		Username: "second-user",
		Password: "password-2",
	})

	s.Equal(http.StatusCreated, firstResp.StatusCode)
	s.Equal(http.StatusCreated, secondResp.StatusCode)
	s.True(s.userIsSuperuser(firstResp.Body.UserId))
	s.False(s.userIsSuperuser(secondResp.Body.UserId))
}

type registerResponse struct {
	StatusCode int
	Body       api.TokenResponse
}

func (s *IntegrationTestSuite) registerUser(user api.UserAuth) registerResponse {
	body, err := json.Marshal(user)
	s.Require().NoError(err)

	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		s.server.URL+"/auth/register",
		bytes.NewReader(body),
	)
	s.Require().NoError(err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	s.Require().NoError(err)
	s.T().Cleanup(func() {
		s.Require().NoError(resp.Body.Close())
	})

	respBody, err := io.ReadAll(resp.Body)
	s.Require().NoError(err)

	var tokenResp api.TokenResponse
	err = json.Unmarshal(respBody, &tokenResp)
	s.Require().NoError(err)

	return registerResponse{
		StatusCode: resp.StatusCode,
		Body:       tokenResp,
	}
}

func (s *IntegrationTestSuite) userIsSuperuser(userID int32) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var isSuperuser bool
	err := s.env.DB.QueryRow(
		ctx,
		`SELECT is_superuser FROM "user" WHERE id = $1`,
		userID,
	).Scan(&isSuperuser)
	s.Require().NoError(err)

	return isSuperuser
}
