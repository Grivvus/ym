package tests

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Grivvus/ym/internal/api"
	"github.com/Grivvus/ym/internal/db"
	"github.com/Grivvus/ym/internal/handlers"
	"github.com/Grivvus/ym/internal/repository"
	"github.com/Grivvus/ym/internal/service"
	"github.com/Grivvus/ym/internal/storage"
	"github.com/Grivvus/ym/internal/utils"
	"github.com/Grivvus/ym/tests/testenv"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgtype"
	openapi_types "github.com/oapi-codegen/runtime/types"
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
	albumService := service.NewAlbumService(albumRepo, env.Storage, logger)
	trackService := service.NewTrackService(
		trackRepo, userRepo, env.Storage, logger, queueNotificationChan,
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
	s.Require().NoError(env.Storage.PutTrack(
		ctx, objectID, bytes.NewReader(payload),
		storage.PutTrackOptions{Size: int64(len(payload))},
	))
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

func (s *IntegrationTestSuite) TestDownloadTrackHeadReturnsDownloadMetadata() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userResp := s.registerUser(api.UserAuth{
		Username: "download-user",
		Password: "password-1",
	})
	s.Equal(http.StatusCreated, userResp.StatusCode)

	trackID, checksum := s.createDownloadableTrack(ctx, userResp.Body.UserId)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodHead,
		fmt.Sprintf("%s/tracks/%d/download?quality=standard", s.server.URL, trackID),
		nil,
	)
	s.Require().NoError(err)
	req.Header.Set("Authorization", "Bearer "+userResp.Body.AccessToken)

	resp, err := s.client.Do(req)
	s.Require().NoError(err)
	defer func() {
		s.Require().NoError(resp.Body.Close())
	}()

	s.Equal(http.StatusOK, resp.StatusCode)
	s.Equal("audio/ogg", resp.Header.Get("Content-Type"))
	s.Equal("bytes", resp.Header.Get("Accept-Ranges"))
	s.Equal("standard", resp.Header.Get("X-Track-Quality-Requested"))
	s.Equal("fast", resp.Header.Get("X-Track-Quality-Resolved"))
	s.Equal(checksum, resp.Header.Get("X-Track-Checksum-Sha256"))
	s.NotEmpty(resp.Header.Get("ETag"))
	s.True(
		strings.Contains(
			resp.Header.Get("Content-Disposition"),
			fmt.Sprintf("filename=track-%d-fast.opus", trackID),
		),
	)
}

func (s *IntegrationTestSuite) TestDownloadTrackReturnsFileAndHeaders() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userResp := s.registerUser(api.UserAuth{
		Username: "download-user",
		Password: "password-1",
	})
	s.Equal(http.StatusCreated, userResp.StatusCode)

	trackID, checksum := s.createDownloadableTrack(ctx, userResp.Body.UserId)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf("%s/tracks/%d/download?quality=fast", s.server.URL, trackID),
		nil,
	)
	s.Require().NoError(err)
	req.Header.Set("Authorization", "Bearer "+userResp.Body.AccessToken)

	resp, err := s.client.Do(req)
	s.Require().NoError(err)
	defer func() {
		s.Require().NoError(resp.Body.Close())
	}()

	body, err := io.ReadAll(resp.Body)
	s.Require().NoError(err)

	s.Equal(http.StatusOK, resp.StatusCode)
	s.Equal([]byte("fast-track-payload"), body)
	s.Equal("fast", resp.Header.Get("X-Track-Quality-Resolved"))
	s.Equal(checksum, resp.Header.Get("X-Track-Checksum-Sha256"))
	s.True(
		strings.Contains(
			resp.Header.Get("Content-Disposition"),
			fmt.Sprintf("filename=track-%d-fast.opus", trackID),
		),
	)
}

func (s *IntegrationTestSuite) TestDownloadTrackForbidsPrivateTrackForAnotherUser() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ownerResp := s.registerUser(api.UserAuth{
		Username: "download-owner",
		Password: "password-1",
	})
	otherResp := s.registerUser(api.UserAuth{
		Username: "download-other",
		Password: "password-1",
	})
	s.Equal(http.StatusCreated, ownerResp.StatusCode)
	s.Equal(http.StatusCreated, otherResp.StatusCode)

	trackID, _ := s.createDownloadableTrack(ctx, ownerResp.Body.UserId)

	statusCode, respBody := s.performJSONRequest(
		http.MethodGet,
		fmt.Sprintf("/tracks/%d/download?quality=fast", trackID),
		nil,
		otherResp.Body.AccessToken,
	)

	var errorResp api.ErrorResponse
	s.Require().NoError(json.Unmarshal(respBody, &errorResp))
	s.Equal(http.StatusForbidden, statusCode)
	s.Contains(errorResp.Error, "user can't have access to this track")
}

func (s *IntegrationTestSuite) TestDeleteTrackOnlySuperuserCanDeleteAndCleansRelations() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	superResp := s.registerUser(api.UserAuth{
		Username: "delete-super",
		Password: "password-1",
	})
	ownerResp := s.registerUser(api.UserAuth{
		Username: "delete-owner",
		Password: "password-1",
	})
	s.Equal(http.StatusCreated, superResp.StatusCode)
	s.Equal(http.StatusCreated, ownerResp.StatusCode)

	trackID, _ := s.createDownloadableTrack(ctx, ownerResp.Body.UserId)
	albumID, err := s.env.Queries.GetAlbumByTrackID(ctx, trackID)
	s.Require().NoError(err)
	_ = s.createPlaylistWithTrack(ctx, ownerResp.Body.UserId, trackID, false)

	statusCode, respBody := s.performJSONRequest(
		http.MethodDelete,
		fmt.Sprintf("/tracks/%d", trackID),
		nil,
		ownerResp.Body.AccessToken,
	)
	s.Equal(http.StatusForbidden, statusCode)
	var errorResp api.ErrorResponse
	s.Require().NoError(json.Unmarshal(respBody, &errorResp))
	s.Contains(errorResp.Error, "required superuser rights")

	statusCode, _ = s.performJSONRequest(
		http.MethodDelete,
		fmt.Sprintf("/tracks/%d", trackID),
		nil,
		superResp.Body.AccessToken,
	)
	s.Equal(http.StatusOK, statusCode)

	s.Equal(0, s.rowCount(ctx, `SELECT COUNT(*) FROM "track" WHERE id = $1`, trackID))
	s.Equal(0, s.rowCount(ctx, `SELECT COUNT(*) FROM "track_album" WHERE track_id = $1`, trackID))
	s.Equal(0, s.rowCount(ctx, `SELECT COUNT(*) FROM "track_playlist" WHERE track_id = $1`, trackID))
	s.Equal(1, s.rowCount(ctx, `SELECT COUNT(*) FROM "album" WHERE id = $1`, albumID))
}

func (s *IntegrationTestSuite) TestDeleteTrackDeletesSingleAlbum() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	superResp := s.registerUser(api.UserAuth{
		Username: "delete-single-super",
		Password: "password-1",
	})
	s.Equal(http.StatusCreated, superResp.StatusCode)

	artist, err := s.env.Queries.CreateArtist(ctx, "delete-single-artist")
	s.Require().NoError(err)

	album, err := s.env.Queries.CreateAlbum(ctx, db.CreateAlbumParams{
		Name:     "delete-single-track",
		ArtistID: artist.ID,
	})
	s.Require().NoError(err)

	track, err := s.env.Queries.CreateTrack(ctx, db.CreateTrackParams{
		Name:                "delete-single-track",
		ArtistID:            artist.ID,
		IsGloballyAvailable: false,
		UploadByUser:        pgtype.Int4{Int32: superResp.Body.UserId, Valid: true},
	})
	s.Require().NoError(err)
	s.Require().NoError(s.env.Queries.AddTrackToAlbum(ctx, db.AddTrackToAlbumParams{
		TrackID: track.ID,
		AlbumID: album.ID,
	}))
	_ = s.createPlaylistWithTrack(ctx, superResp.Body.UserId, track.ID, false)

	originalTrackKey := fmt.Sprintf("track%d", track.ID)
	s.Require().NoError(s.env.Storage.PutTrack(
		ctx, originalTrackKey, bytes.NewReader([]byte("single-track-payload")),
		storage.PutTrackOptions{Size: int64(len("single-track-payload"))},
	))

	statusCode, _ := s.performJSONRequest(
		http.MethodDelete,
		fmt.Sprintf("/tracks/%d", track.ID),
		nil,
		superResp.Body.AccessToken,
	)
	s.Equal(http.StatusOK, statusCode)

	s.Equal(0, s.rowCount(ctx, `SELECT COUNT(*) FROM "track" WHERE id = $1`, track.ID))
	s.Equal(0, s.rowCount(ctx, `SELECT COUNT(*) FROM "track_album" WHERE track_id = $1`, track.ID))
	s.Equal(0, s.rowCount(ctx, `SELECT COUNT(*) FROM "track_playlist" WHERE track_id = $1`, track.ID))
	s.Equal(0, s.rowCount(ctx, `SELECT COUNT(*) FROM "album" WHERE id = $1`, album.ID))
}

func (s *IntegrationTestSuite) TestSharedPlaylistGrantsAndRevokesPrivateTrackAccess() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ownerResp := s.registerUser(api.UserAuth{
		Username: "shared-owner",
		Password: "password-1",
	})
	otherResp := s.registerUser(api.UserAuth{
		Username: "shared-other",
		Password: "password-1",
	})
	s.Equal(http.StatusCreated, ownerResp.StatusCode)
	s.Equal(http.StatusCreated, otherResp.StatusCode)

	trackID, _ := s.createDownloadableTrack(ctx, ownerResp.Body.UserId)
	playlistID := s.createPlaylistWithTrack(ctx, ownerResp.Body.UserId, trackID, false)

	statusCode, _ := s.performJSONRequest(
		http.MethodGet,
		fmt.Sprintf("/tracks/%d/download?quality=fast", trackID),
		nil,
		otherResp.Body.AccessToken,
	)
	s.Equal(http.StatusForbidden, statusCode)

	statusCode, _ = s.performJSONRequest(
		http.MethodPost,
		fmt.Sprintf("/playlists/%d/share", playlistID),
		api.PlaylistShareRequest{
			ShareWithUsers:     []int32{otherResp.Body.UserId},
			HasWritePermission: false,
		},
		ownerResp.Body.AccessToken,
	)
	s.Equal(http.StatusOK, statusCode)

	statusCode, respBody := s.performJSONRequest(
		http.MethodGet,
		"/tracks",
		nil,
		otherResp.Body.AccessToken,
	)
	s.Equal(http.StatusOK, statusCode)
	var tracks []api.TrackMetadata
	s.Require().NoError(json.Unmarshal(respBody, &tracks))
	s.True(trackListContains(tracks, trackID))

	statusCode, respBody = s.performJSONRequest(
		http.MethodGet,
		fmt.Sprintf("/tracks/%d/download?quality=fast", trackID),
		nil,
		otherResp.Body.AccessToken,
	)
	s.Equal(http.StatusOK, statusCode)
	s.Equal([]byte("fast-track-payload"), respBody)

	statusCode, _ = s.performJSONRequest(
		http.MethodPost,
		fmt.Sprintf("/playlists/%d/revoke", playlistID),
		api.PlaylistRevokeAccessRequest{UserId: otherResp.Body.UserId},
		ownerResp.Body.AccessToken,
	)
	s.Equal(http.StatusOK, statusCode)

	statusCode, respBody = s.performJSONRequest(
		http.MethodGet,
		"/tracks",
		nil,
		otherResp.Body.AccessToken,
	)
	s.Equal(http.StatusOK, statusCode)
	s.Require().NoError(json.Unmarshal(respBody, &tracks))
	s.False(trackListContains(tracks, trackID))

	statusCode, _ = s.performJSONRequest(
		http.MethodGet,
		fmt.Sprintf("/tracks/%d/download?quality=fast", trackID),
		nil,
		otherResp.Body.AccessToken,
	)
	s.Equal(http.StatusForbidden, statusCode)
}

func (s *IntegrationTestSuite) TestPublicPlaylistGrantsPrivateTrackAccess() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ownerResp := s.registerUser(api.UserAuth{
		Username: "public-owner",
		Password: "password-1",
	})
	otherResp := s.registerUser(api.UserAuth{
		Username: "public-other",
		Password: "password-1",
	})
	s.Equal(http.StatusCreated, ownerResp.StatusCode)
	s.Equal(http.StatusCreated, otherResp.StatusCode)

	trackID, _ := s.createDownloadableTrack(ctx, ownerResp.Body.UserId)
	playlistID := s.createPlaylistWithTrack(ctx, ownerResp.Body.UserId, trackID, true)

	statusCode, respBody := s.performJSONRequest(
		http.MethodGet,
		"/tracks",
		nil,
		otherResp.Body.AccessToken,
	)
	s.Equal(http.StatusOK, statusCode)
	var tracks []api.TrackMetadata
	s.Require().NoError(json.Unmarshal(respBody, &tracks))
	s.True(trackListContains(tracks, trackID))

	statusCode, respBody = s.performJSONRequest(
		http.MethodGet,
		fmt.Sprintf("/playlists/%d", playlistID),
		nil,
		otherResp.Body.AccessToken,
	)
	s.Equal(http.StatusOK, statusCode)
	var playlist api.PlaylistWithTracksResponse
	s.Require().NoError(json.Unmarshal(respBody, &playlist))
	s.Equal(playlistID, playlist.PlaylistId)
	s.Contains(playlist.Tracks, trackID)

	statusCode, respBody = s.performJSONRequest(
		http.MethodGet,
		fmt.Sprintf("/tracks/%d/download?quality=fast", trackID),
		nil,
		otherResp.Body.AccessToken,
	)
	s.Equal(http.StatusOK, statusCode)
	s.Equal([]byte("fast-track-payload"), respBody)
}

func (s *IntegrationTestSuite) TestReadOnlyPlaylistShareCannotAddTrack() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ownerResp := s.registerUser(api.UserAuth{
		Username: "readonly-owner",
		Password: "password-1",
	})
	otherResp := s.registerUser(api.UserAuth{
		Username: "readonly-other",
		Password: "password-1",
	})
	s.Equal(http.StatusCreated, ownerResp.StatusCode)
	s.Equal(http.StatusCreated, otherResp.StatusCode)

	trackID, _ := s.createDownloadableTrack(ctx, ownerResp.Body.UserId)
	playlistID := s.createPlaylistWithTrack(ctx, ownerResp.Body.UserId, trackID, false)

	statusCode, _ := s.performJSONRequest(
		http.MethodPost,
		fmt.Sprintf("/playlists/%d/share", playlistID),
		api.PlaylistShareRequest{
			ShareWithUsers:     []int32{otherResp.Body.UserId},
			HasWritePermission: false,
		},
		ownerResp.Body.AccessToken,
	)
	s.Equal(http.StatusOK, statusCode)

	statusCode, _ = s.performJSONRequest(
		http.MethodPost,
		fmt.Sprintf("/playlists/%d", playlistID),
		api.AddTrackToPlaylistJSONBody{TrackId: trackID},
		otherResp.Body.AccessToken,
	)
	s.Equal(http.StatusForbidden, statusCode)
}

func (s *IntegrationTestSuite) TestWritePlaylistShareCannotAddInaccessibleTrack() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ownerResp := s.registerUser(api.UserAuth{
		Username: "write-owner",
		Password: "password-1",
	})
	writerResp := s.registerUser(api.UserAuth{
		Username: "write-user",
		Password: "password-1",
	})
	trackOwnerResp := s.registerUser(api.UserAuth{
		Username: "private-track-owner",
		Password: "password-1",
	})
	s.Equal(http.StatusCreated, ownerResp.StatusCode)
	s.Equal(http.StatusCreated, writerResp.StatusCode)
	s.Equal(http.StatusCreated, trackOwnerResp.StatusCode)

	playlistTrackID, _ := s.createDownloadableTrack(ctx, ownerResp.Body.UserId)
	playlistID := s.createPlaylistWithTrack(ctx, ownerResp.Body.UserId, playlistTrackID, false)
	privateTrackID, _ := s.createDownloadableTrack(ctx, trackOwnerResp.Body.UserId)

	statusCode, _ := s.performJSONRequest(
		http.MethodPost,
		fmt.Sprintf("/playlists/%d/share", playlistID),
		api.PlaylistShareRequest{
			ShareWithUsers:     []int32{writerResp.Body.UserId},
			HasWritePermission: true,
		},
		ownerResp.Body.AccessToken,
	)
	s.Equal(http.StatusOK, statusCode)

	statusCode, _ = s.performJSONRequest(
		http.MethodPost,
		fmt.Sprintf("/playlists/%d", playlistID),
		api.AddTrackToPlaylistJSONBody{TrackId: privateTrackID},
		writerResp.Body.AccessToken,
	)
	s.Equal(http.StatusForbidden, statusCode)
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

	sharedUser, err := s.env.Queries.CreateUser(ctx, db.CreateUserParams{
		Username:    "backup-shared-user",
		Email:       pgtype.Text{String: "backup-shared@example.com", Valid: true},
		Password:    []byte("password-hash"),
		Salt:        []byte("password-salt"),
		IsSuperuser: false,
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
		OwnerID:  user.ID,
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
	s.Require().NoError(s.env.Queries.SharePlaylistWith(ctx, db.SharePlaylistWithParams{
		PlaylistID:         playlist.ID,
		SharedWithUser:     sharedUser.ID,
		HasWritePermission: true,
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
		ctx, originalTrackKey, bytes.NewReader(originalTrack),
		storage.PutTrackOptions{Size: int64(len(originalTrack))},
	))
	s.Require().NoError(s.env.Storage.PutTrack(
		ctx, fastTrackKey, bytes.NewReader(fastTrack),
		storage.PutTrackOptions{Size: int64(len(fastTrack))},
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
	s.Len(users, 2)
	s.Equal("backup-user", users[0].Username)

	playlistShares, err := s.env.Queries.GetAllPlaylistSharesForBackup(ctx)
	s.Require().NoError(err)
	s.Len(playlistShares, 1)
	s.Equal(playlist.ID, playlistShares[0].PlaylistID)
	s.Equal(sharedUser.ID, playlistShares[0].SharedWithUser)
	s.True(playlistShares[0].HasWritePermission)

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

func (s *IntegrationTestSuite) TestBackupEndpointStartsAsyncOperationAndDownloadsArchive() {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	userResp := s.registerUser(api.UserAuth{
		Username: "async-backup-user",
		Password: "password",
	})
	s.Require().Equal(http.StatusCreated, userResp.StatusCode)

	artist, err := s.env.Queries.CreateArtist(ctx, "async-backup-artist")
	s.Require().NoError(err)

	track, err := s.env.Queries.CreateTrack(ctx, db.CreateTrackParams{
		Name:                "async-backup-track",
		ArtistID:            artist.ID,
		IsGloballyAvailable: false,
		UploadByUser:        pgtype.Int4{Int32: userResp.Body.UserId, Valid: true},
	})
	s.Require().NoError(err)

	originalTrackKey := fmt.Sprintf("track%d", track.ID)
	originalTrackPayload := []byte("async-original-track-payload")
	s.Require().NoError(s.env.Storage.PutTrack(
		ctx,
		originalTrackKey,
		bytes.NewReader(originalTrackPayload),
		storage.PutTrackOptions{Size: int64(len(originalTrackPayload))},
	))
	s.T().Cleanup(func() {
		_ = s.env.Storage.RemoveTrack(context.Background(), originalTrackKey)
	})

	statusCode, respBody := s.performJSONRequest(
		http.MethodPost,
		"/backup?include_transcoded_tracks=false",
		nil,
		userResp.Body.AccessToken,
	)
	s.Equal(http.StatusAccepted, statusCode)

	var startResp api.BackupStatusResponse
	s.Require().NoError(json.Unmarshal(respBody, &startResp))
	s.NotEmpty(startResp.BackupId)
	s.Equal(api.Pending, startResp.Status)
	s.False(startResp.IncludeImages)
	s.False(startResp.IncludeTranscodedTracks)

	s.T().Cleanup(func() {
		status, err := s.env.Queries.GetBackupStatus(context.Background(), startResp.BackupId)
		if err == nil && status.ArchivePath.Valid {
			_ = os.Remove(status.ArchivePath.String)
		}
	})

	var statusResp api.BackupStatusResponse
	s.Eventually(func() bool {
		statusCode, respBody := s.performJSONRequest(
			http.MethodGet,
			"/backup/"+startResp.BackupId,
			nil,
			userResp.Body.AccessToken,
		)
		s.Require().Equal(http.StatusOK, statusCode)
		s.Require().NoError(json.Unmarshal(respBody, &statusResp))
		if statusResp.Status == api.Error {
			if statusResp.Error != nil {
				s.T().Logf("backup failed: %s", *statusResp.Error)
			}
			return true
		}
		return statusResp.Status == api.Finished
	}, 10*time.Second, 100*time.Millisecond)
	s.Equal(api.Finished, statusResp.Status)
	s.Require().NotNil(statusResp.SizeBytes)
	s.Positive(*statusResp.SizeBytes)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		s.server.URL+"/backup/"+startResp.BackupId+"/download",
		nil,
	)
	s.Require().NoError(err)
	req.Header.Set("Authorization", "Bearer "+userResp.Body.AccessToken)

	downloadResp, err := s.client.Do(req)
	s.Require().NoError(err)
	defer func() { s.Require().NoError(downloadResp.Body.Close()) }()

	s.Equal(http.StatusOK, downloadResp.StatusCode)
	s.Equal("application/zip", downloadResp.Header.Get("Content-Type"))
	archiveBytes, err := io.ReadAll(downloadResp.Body)
	s.Require().NoError(err)
	s.Positive(len(archiveBytes))

	archive, err := zip.NewReader(bytes.NewReader(archiveBytes), int64(len(archiveBytes)))
	s.Require().NoError(err)
	s.NotNil(findZipEntry(archive, backupManifestPathForTest))
	s.NotNil(findZipEntry(archive, backupDBPathForTest))
	s.NotNil(findZipEntry(archive, "tracks/original/"+originalTrackKey))
}

func (s *IntegrationTestSuite) TestRestoreQueuesTracksWhenTranscodedFilesMissingFromBackup() {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	backupService := service.NewBackupService(logger, s.env.Queries, s.env.Storage)

	user, err := s.env.Queries.CreateUser(ctx, db.CreateUserParams{
		Username: "missing-transcoded-user",
		Password: []byte("password-hash"),
		Salt:     []byte("password-salt"),
	})
	s.Require().NoError(err)

	artist, err := s.env.Queries.CreateArtist(ctx, "missing-transcoded-artist")
	s.Require().NoError(err)

	album, err := s.env.Queries.CreateAlbum(ctx, db.CreateAlbumParams{
		Name:     "missing-transcoded-album",
		ArtistID: artist.ID,
	})
	s.Require().NoError(err)

	track, err := s.env.Queries.CreateTrack(ctx, db.CreateTrackParams{
		Name:                "missing-transcoded-track",
		ArtistID:            artist.ID,
		IsGloballyAvailable: false,
		UploadByUser:        pgtype.Int4{Int32: user.ID, Valid: true},
	})
	s.Require().NoError(err)

	s.Require().NoError(s.env.Queries.AddTrackToAlbum(ctx, db.AddTrackToAlbumParams{
		TrackID: track.ID,
		AlbumID: album.ID,
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

	s.Require().NoError(s.env.Storage.PutTrack(
		ctx,
		originalTrackKey,
		bytes.NewReader([]byte("original-track-payload")),
		storage.PutTrackOptions{Size: int64(len("original-track-payload"))},
	))
	s.Require().NoError(s.env.Storage.PutTrack(
		ctx,
		fastTrackKey,
		bytes.NewReader([]byte("fast-track-payload")),
		storage.PutTrackOptions{Size: int64(len("fast-track-payload"))},
	))

	backupReader, _, err := backupService.MakeBackup(ctx, service.BackupSettings{
		IncludeTranscodedTracks: true,
	})
	s.Require().NoError(err)
	archiveBytes, err := io.ReadAll(backupReader)
	s.Require().NoError(err)
	s.Require().NoError(backupReader.Close())

	archiveBytes = s.zipWithoutEntriesWithPrefix(archiveBytes, "tracks/transcoded/")

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

	tracks, err := s.env.Queries.GetAllTracksForBackup(ctx)
	s.Require().NoError(err)
	s.Len(tracks, 1)
	s.False(tracks[0].FastPresetFname.Valid)
	s.False(tracks[0].StandardPresetFname.Valid)
	s.False(tracks[0].HighPresetFname.Valid)
	s.False(tracks[0].LosslessPresetFname.Valid)

	queueRows, err := s.env.Queries.GetAllTranscodingQueueForBackup(ctx)
	s.Require().NoError(err)
	s.Len(queueRows, 1)
	s.Equal(track.ID, queueRows[0].TrackID)
	s.Equal(originalTrackKey, queueRows[0].TrackOriginalFileName)
	s.False(queueRows[0].WasFailed)
}

func (s *IntegrationTestSuite) TestPasswordReset_RequestReturnsAcceptedForUnknownEmail() {
	resp := s.requestPasswordReset("unknown@example.com")

	s.Equal(http.StatusAccepted, resp.StatusCode)
	s.Equal(
		"if an account with that email exists, a reset code has been sent",
		resp.Body.Msg,
	)
	_, exists := s.resetMailer.LastCode("unknown@example.com")
	s.False(exists)
}

func (s *IntegrationTestSuite) TestPasswordReset_ConfirmChangesPasswordAndInvalidatesRefreshToken() {
	registerResp := s.registerUser(api.UserAuth{
		Username: "reset-user",
		Password: "password-1",
	})
	s.Equal(http.StatusCreated, registerResp.StatusCode)
	s.setUserEmail(registerResp.Body.UserId, "reset@example.com", "reset-user")

	requestResp := s.requestPasswordReset("reset@example.com")
	s.Equal(http.StatusAccepted, requestResp.StatusCode)

	code, exists := s.resetMailer.LastCode("reset@example.com")
	s.True(exists)
	s.Len(code, 6)

	oldAccessToken := registerResp.Body.AccessToken
	oldRefreshToken := registerResp.Body.RefreshToken

	confirmResp := s.confirmPasswordReset("reset@example.com", code, "password-2")
	s.Equal(http.StatusOK, confirmResp.StatusCode)
	s.Equal("password was successfully reset", confirmResp.Body.Msg)

	refreshResp := s.refreshTokens(oldRefreshToken)
	s.Equal(http.StatusUnauthorized, refreshResp.StatusCode)
	s.Equal("invalid refresh token", refreshResp.Error.Error)

	loginOldPasswordResp := s.loginUser(api.UserAuth{
		Username: "reset-user",
		Password: "password-1",
	})
	s.Equal(http.StatusUnauthorized, loginOldPasswordResp.StatusCode)
	s.Equal("invalid credentials", loginOldPasswordResp.Error.Error)

	loginNewPasswordResp := s.loginUser(api.UserAuth{
		Username: "reset-user",
		Password: "password-2",
	})
	s.Equal(http.StatusOK, loginNewPasswordResp.StatusCode)
	s.NotEmpty(loginNewPasswordResp.Body.AccessToken)

	statusCode, body := s.performJSONRequest(
		http.MethodGet,
		fmt.Sprintf("/users/%d", registerResp.Body.UserId),
		nil,
		oldAccessToken,
	)
	s.Equal(http.StatusOK, statusCode)

	var user api.UserReturn
	s.Require().NoError(json.Unmarshal(body, &user))
	s.Equal("reset@example.com", *user.Email)
}

func (s *IntegrationTestSuite) createDownloadableTrack(
	ctx context.Context, ownerID int32,
) (int32, string) {
	suffix := fmt.Sprintf("%d-%d", ownerID, time.Now().UnixNano())
	artist, err := s.env.Queries.CreateArtist(ctx, "download-artist-"+suffix)
	s.Require().NoError(err)

	album, err := s.env.Queries.CreateAlbum(ctx, db.CreateAlbumParams{
		Name:     "download-album-" + suffix,
		ArtistID: artist.ID,
	})
	s.Require().NoError(err)

	track, err := s.env.Queries.CreateTrack(ctx, db.CreateTrackParams{
		Name:                "download-track",
		ArtistID:            artist.ID,
		IsGloballyAvailable: false,
		UploadByUser:        pgtype.Int4{Int32: ownerID, Valid: true},
	})
	s.Require().NoError(err)

	s.Require().NoError(s.env.Queries.AddTrackToAlbum(ctx, db.AddTrackToAlbumParams{
		TrackID: track.ID,
		AlbumID: album.ID,
	}))

	fastTrackKey := fmt.Sprintf("track%d_fast", track.ID)
	_, err = s.env.Queries.AddPostTranscodingInfo(ctx, db.AddPostTranscodingInfoParams{
		ID:                  track.ID,
		DurationMs:          pgtype.Int4{Int32: 12345, Valid: true},
		FastPresetFname:     pgtype.Text{String: fastTrackKey, Valid: true},
		StandardPresetFname: pgtype.Text{Valid: false},
		HighPresetFname:     pgtype.Text{Valid: false},
		LosslessPresetFname: pgtype.Text{Valid: false},
	})
	s.Require().NoError(err)

	payload := []byte("fast-track-payload")
	checksum, err := utils.SHA256HexFromReadSeeker(bytes.NewReader(payload))
	s.Require().NoError(err)

	s.Require().NoError(s.env.Storage.PutTrack(
		ctx,
		fastTrackKey,
		bytes.NewReader(payload),
		storage.PutTrackOptions{
			Size:           int64(len(payload)),
			ContentType:    "audio/ogg",
			ChecksumSHA256: checksum,
		},
	))
	s.T().Cleanup(func() {
		s.Require().NoError(s.env.Storage.RemoveTrack(context.Background(), fastTrackKey))
	})

	return track.ID, checksum
}

func (s *IntegrationTestSuite) createPlaylistWithTrack(
	ctx context.Context, ownerID, trackID int32, isPublic bool,
) int32 {
	playlist, err := s.env.Queries.CreatePlaylist(ctx, db.CreatePlaylistParams{
		Name:     fmt.Sprintf("playlist-%d-%d", ownerID, trackID),
		IsPublic: isPublic,
		OwnerID:  ownerID,
	})
	s.Require().NoError(err)

	s.Require().NoError(s.env.Queries.AddTrackToPlaylist(ctx, db.AddTrackToPlaylistParams{
		TrackID:    trackID,
		PlaylistID: playlist.ID,
	}))

	return playlist.ID
}

func trackListContains(tracks []api.TrackMetadata, trackID int32) bool {
	for _, track := range tracks {
		if track.TrackId == trackID {
			return true
		}
	}
	return false
}

func (s *IntegrationTestSuite) zipWithoutEntriesWithPrefix(payload []byte, prefix string) []byte {
	reader, err := zip.NewReader(bytes.NewReader(payload), int64(len(payload)))
	s.Require().NoError(err)

	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	for _, file := range reader.File {
		if strings.HasPrefix(file.Name, prefix) {
			continue
		}

		header := file.FileHeader
		entryWriter, err := writer.CreateHeader(&header)
		s.Require().NoError(err)

		entryReader, err := file.Open()
		s.Require().NoError(err)
		_, copyErr := io.Copy(entryWriter, entryReader)
		closeErr := entryReader.Close()
		s.Require().NoError(copyErr)
		s.Require().NoError(closeErr)
	}
	s.Require().NoError(writer.Close())

	return output.Bytes()
}

const (
	backupManifestPathForTest = "manifest.json"
	backupDBPathForTest       = "db/full.json"
)

func findZipEntry(archive *zip.Reader, name string) *zip.File {
	for _, file := range archive.File {
		if file.Name == name {
			return file
		}
	}
	return nil
}

type tokenResponse struct {
	StatusCode int
	Body       api.TokenResponse
	Error      api.ErrorResponse
}

type messageResponse struct {
	StatusCode int
	Body       api.MessageResponse
	Error      api.ErrorResponse
}

func (s *IntegrationTestSuite) registerUser(user api.UserAuth) tokenResponse {
	statusCode, respBody := s.performJSONRequest(http.MethodPost, "/auth/register", user, "")

	var tokenResp api.TokenResponse
	s.Require().NoError(json.Unmarshal(respBody, &tokenResp))

	return tokenResponse{
		StatusCode: statusCode,
		Body:       tokenResp,
	}
}

func (s *IntegrationTestSuite) loginUser(user api.UserAuth) tokenResponse {
	statusCode, respBody := s.performJSONRequest(http.MethodPost, "/auth/login", user, "")
	response := tokenResponse{StatusCode: statusCode}
	if statusCode == http.StatusOK {
		s.Require().NoError(json.Unmarshal(respBody, &response.Body))
	} else {
		s.Require().NoError(json.Unmarshal(respBody, &response.Error))
	}
	return response
}

func (s *IntegrationTestSuite) refreshTokens(refreshToken string) tokenResponse {
	statusCode, respBody := s.performJSONRequest(
		http.MethodPost,
		"/auth/refresh",
		api.UpdateTokenRequest{RefreshToken: refreshToken},
		"",
	)
	response := tokenResponse{StatusCode: statusCode}
	if statusCode == http.StatusOK {
		s.Require().NoError(json.Unmarshal(respBody, &response.Body))
	} else {
		s.Require().NoError(json.Unmarshal(respBody, &response.Error))
	}
	return response
}

func (s *IntegrationTestSuite) requestPasswordReset(email string) messageResponse {
	statusCode, respBody := s.performJSONRequest(
		http.MethodPost,
		"/auth/password-reset/request",
		api.PasswordResetRequest{Email: openapi_types.Email(email)},
		"",
	)
	response := messageResponse{StatusCode: statusCode}
	if statusCode == http.StatusAccepted {
		s.Require().NoError(json.Unmarshal(respBody, &response.Body))
	} else {
		s.Require().NoError(json.Unmarshal(respBody, &response.Error))
	}
	return response
}

func (s *IntegrationTestSuite) confirmPasswordReset(
	email string, code string, newPassword string,
) messageResponse {
	statusCode, respBody := s.performJSONRequest(
		http.MethodPost,
		"/auth/password-reset/confirm",
		api.PasswordResetConfirmRequest{
			Email:       openapi_types.Email(email),
			Code:        code,
			NewPassword: newPassword,
		},
		"",
	)
	response := messageResponse{StatusCode: statusCode}
	if statusCode == http.StatusOK {
		s.Require().NoError(json.Unmarshal(respBody, &response.Body))
	} else {
		s.Require().NoError(json.Unmarshal(respBody, &response.Error))
	}
	return response
}

func (s *IntegrationTestSuite) performJSONRequest(
	method string, path string, payload any, accessToken string,
) (int, []byte) {
	var bodyReader io.Reader
	if payload != nil {
		body, err := json.Marshal(payload)
		s.Require().NoError(err)
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(
		context.Background(),
		method,
		s.server.URL+path,
		bodyReader,
	)
	s.Require().NoError(err)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}

	resp, err := s.client.Do(req)
	s.Require().NoError(err)
	defer func() {
		s.Require().NoError(resp.Body.Close())
	}()

	respBody, err := io.ReadAll(resp.Body)
	s.Require().NoError(err)

	return resp.StatusCode, respBody
}

func (s *IntegrationTestSuite) setUserEmail(userID int32, email string, username string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := s.env.Queries.UpdateUser(ctx, db.UpdateUserParams{
		ID:       userID,
		Username: username,
		Email:    pgtype.Text{String: email, Valid: true},
	})
	s.Require().NoError(err)
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

func (s *IntegrationTestSuite) rowCount(ctx context.Context, query string, args ...any) int {
	var count int
	err := s.env.DB.QueryRow(ctx, query, args...).Scan(&count)
	s.Require().NoError(err)
	return count
}

type testPasswordResetMailer struct {
	mu    sync.Mutex
	codes map[string]string
}

func newTestPasswordResetMailer() *testPasswordResetMailer {
	return &testPasswordResetMailer{codes: map[string]string{}}
}

func (m *testPasswordResetMailer) SendPasswordResetCode(
	_ context.Context, recipient string, code string, _ time.Duration,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.codes[recipient] = code
	return nil
}

func (m *testPasswordResetMailer) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.codes = map[string]string{}
}

func (m *testPasswordResetMailer) LastCode(recipient string) (string, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	code, ok := m.codes[recipient]
	return code, ok
}
