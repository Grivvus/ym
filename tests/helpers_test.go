package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/Grivvus/ym/internal/api"
	"github.com/Grivvus/ym/internal/db"
	"github.com/Grivvus/ym/internal/storage"
	"github.com/Grivvus/ym/internal/utils"
	"github.com/jackc/pgx/v5/pgtype"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

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
