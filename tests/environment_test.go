package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Grivvus/ym/internal/api"
	"github.com/Grivvus/ym/internal/db"
	"github.com/Grivvus/ym/internal/handlers"
	"github.com/Grivvus/ym/internal/service"
	"github.com/Grivvus/ym/tests/testenv"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgtype"
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
	backupService := service.NewBackupService(logger, env.Queries, env.Storage)

	rootHandler := handlers.NewRootHandler(
		logger,
		authService, userService,
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
			"transcoding_queue",
			"track_playlist",
			"track_album",
			"track",
			"playlist",
			"album",
			"artist",
			"user",
			"restore_status"
		RESTART IDENTITY CASCADE
	`)
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

func (s *IntegrationTestSuite) TestBackupAndRestore_RestoresDatabaseAndStorage() {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	backupService := service.NewBackupService(logger, s.env.Queries, s.env.Storage)

	user, err := s.env.Queries.CreateUser(ctx, db.CreateUserParams{
		Username:    "backup-user",
		Email:       pgtype.Text{String: "backup@example.com", Valid: true},
		Password:    []byte("password-hash"),
		Salt:        []byte("password-salt"),
		IsSuperuser: true,
	})
	s.Require().NoError(err)

	artist, err := s.env.Queries.CreateArtist(ctx, "backup-artist")
	s.Require().NoError(err)

	album, err := s.env.Queries.CreateAlbum(ctx, db.CreateAlbumParams{
		Name:     "backup-album",
		ArtistID: artist.ID,
	})
	s.Require().NoError(err)

	playlist, err := s.env.Queries.CreatePlaylist(ctx, db.CreatePlaylistParams{
		Name:     "backup-playlist",
		IsPublic: true,
		OwnerID:  pgtype.Int4{Int32: user.ID, Valid: true},
	})
	s.Require().NoError(err)

	track, err := s.env.Queries.CreateTrack(ctx, db.CreateTrackParams{
		Name:                "backup-track",
		ArtistID:            artist.ID,
		IsGloballyAvailable: true,
		UploadByUser:        pgtype.Int4{Int32: user.ID, Valid: true},
	})
	s.Require().NoError(err)

	s.Require().NoError(s.env.Queries.AddTrackToAlbum(ctx, db.AddTrackToAlbumParams{
		TrackID: track.ID,
		AlbumID: album.ID,
	}))
	s.Require().NoError(s.env.Queries.AddTrackToPlaylist(ctx, db.AddTrackToPlaylistParams{
		TrackID:    track.ID,
		PlaylistID: playlist.ID,
	}))

	originalTrackKey := fmt.Sprintf("track%d", track.ID)
	fastTrackKey := originalTrackKey + "_fast"

	_, err = s.env.Queries.AddPostTranscodingInfo(ctx, db.AddPostTranscodingInfoParams{
		ID:                  track.ID,
		DurationMs:          pgtype.Int4{Int32: 12345, Valid: true},
		FastPresetFname:     pgtype.Text{String: fastTrackKey, Valid: true},
		StandardPresetFname: pgtype.Text{Valid: false},
		HighPresetFname:     pgtype.Text{Valid: false},
		LosslessPresetFname: pgtype.Text{Valid: false},
	})
	s.Require().NoError(err)

	userAvatarKey := service.ArtworkOwner{Kind: "user", ID: user.ID, Name: user.Username}.Key()
	artistImageKey := service.ArtworkOwner{Kind: "artist", ID: artist.ID, Name: artist.Name}.Key()
	albumImageKey := service.ArtworkOwner{Kind: "album", ID: album.ID, Name: album.Name}.Key()
	playlistImageKey := service.ArtworkOwner{Kind: "playlist", ID: playlist.ID, Name: playlist.Name}.Key()

	userAvatar := []byte("user-avatar")
	artistImage := []byte("artist-image")
	albumImage := []byte("album-image")
	playlistImage := []byte("playlist-image")
	originalTrack := []byte("original-track-payload")
	fastTrack := []byte("fast-track-payload")

	s.Require().NoError(s.env.Storage.PutImage(ctx, userAvatarKey, bytes.NewReader(userAvatar)))
	s.Require().NoError(s.env.Storage.PutImage(ctx, artistImageKey, bytes.NewReader(artistImage)))
	s.Require().NoError(s.env.Storage.PutImage(ctx, albumImageKey, bytes.NewReader(albumImage)))
	s.Require().NoError(s.env.Storage.PutImage(ctx, playlistImageKey, bytes.NewReader(playlistImage)))
	s.Require().NoError(s.env.Storage.PutTrack(
		ctx, originalTrackKey, bytes.NewReader(originalTrack), int64(len(originalTrack)),
	))
	s.Require().NoError(s.env.Storage.PutTrack(
		ctx, fastTrackKey, bytes.NewReader(fastTrack), int64(len(fastTrack)),
	))

	backupReader, _, err := backupService.MakeBackup(ctx, service.BackupSettings{
		IncludeImages:           true,
		IncludeTranscodedTracks: true,
	})
	s.Require().NoError(err)
	archiveBytes, err := io.ReadAll(backupReader)
	s.Require().NoError(err)
	s.Require().NoError(backupReader.Close())

	_, err = s.env.DB.Exec(ctx, `
		TRUNCATE TABLE
			"transcoding_queue",
			"track_playlist",
			"track_album",
			"track",
			"playlist",
			"album",
			"artist",
			"user",
			"restore_status"
		RESTART IDENTITY CASCADE
	`)
	s.Require().NoError(err)

	s.Require().NoError(s.env.Storage.RemoveImage(ctx, userAvatarKey))
	s.Require().NoError(s.env.Storage.RemoveImage(ctx, artistImageKey))
	s.Require().NoError(s.env.Storage.RemoveImage(ctx, albumImageKey))
	s.Require().NoError(s.env.Storage.RemoveImage(ctx, playlistImageKey))
	s.Require().NoError(s.env.Storage.RemoveTrack(ctx, originalTrackKey))
	s.Require().NoError(s.env.Storage.RemoveTrack(ctx, fastTrackKey))

	restoreID, err := backupService.StartRestoreOperation(ctx, bytes.NewReader(archiveBytes))
	s.Require().NoError(err)

	var status api.RestoreStatusResponse
	s.Eventually(func() bool {
		var statusErr error
		status, statusErr = backupService.CheckRestoreOperation(context.Background(), restoreID)
		s.Require().NoError(statusErr)
		if status.Status == api.Error {
			s.T().Logf("restore failed: %s", *status.Error)
			return true
		}
		return status.Status == api.Finished
	}, 10*time.Second, 100*time.Millisecond)
	s.Equal(api.Finished, status.Status)

	users, err := s.env.Queries.GetAllUsersForBackup(ctx)
	s.Require().NoError(err)
	s.Len(users, 1)
	s.Equal("backup-user", users[0].Username)

	tracks, err := s.env.Queries.GetAllTracksForBackup(ctx)
	s.Require().NoError(err)
	s.Len(tracks, 1)
	s.Equal("backup-track", tracks[0].Name)
	s.True(tracks[0].FastPresetFname.Valid)
	s.Equal(fastTrackKey, tracks[0].FastPresetFname.String)

	gotAvatar, err := s.env.Storage.GetImage(ctx, userAvatarKey)
	s.Require().NoError(err)
	s.Equal(userAvatar, gotAvatar)

	originalReader, err := s.env.Storage.GetTrack(ctx, originalTrackKey)
	s.Require().NoError(err)
	defer func() { s.Require().NoError(originalReader.Close()) }()
	originalPayload, err := io.ReadAll(originalReader)
	s.Require().NoError(err)
	s.Equal(originalTrack, originalPayload)

	fastReader, err := s.env.Storage.GetTrack(ctx, fastTrackKey)
	s.Require().NoError(err)
	defer func() { s.Require().NoError(fastReader.Close()) }()
	fastPayload, err := io.ReadAll(fastReader)
	s.Require().NoError(err)
	s.Equal(fastTrack, fastPayload)
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
