package tests

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/Grivvus/ym/internal/api"
	"github.com/Grivvus/ym/internal/db"
)

func (s *IntegrationTestSuite) TestDeleteTrackFromPlaylistIsIdempotentAndRequiresWritePermission() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ownerResp := s.registerUser(api.UserAuth{
		Username: "delete-playlist-track-owner",
		Password: "password-1",
	})
	readonlyResp := s.registerUser(api.UserAuth{
		Username: "delete-playlist-track-readonly",
		Password: "password-1",
	})
	s.Equal(http.StatusCreated, ownerResp.StatusCode)
	s.Equal(http.StatusCreated, readonlyResp.StatusCode)

	trackID, _ := s.createDownloadableTrack(ctx, ownerResp.Body.UserId)
	playlistID := s.createPlaylistWithTrack(ctx, ownerResp.Body.UserId, trackID, false)

	statusCode, _ := s.performJSONRequest(
		http.MethodPost,
		fmt.Sprintf("/playlists/%d/share", playlistID),
		api.PlaylistShareRequest{
			ShareWithUsers:     []int32{readonlyResp.Body.UserId},
			HasWritePermission: false,
		},
		ownerResp.Body.AccessToken,
	)
	s.Equal(http.StatusOK, statusCode)

	statusCode, _ = s.performJSONRequest(
		http.MethodDelete,
		fmt.Sprintf("/playlists/%d/tracks/%d", playlistID, trackID),
		nil,
		readonlyResp.Body.AccessToken,
	)
	s.Equal(http.StatusForbidden, statusCode)
	s.Equal(1, s.rowCount(
		ctx,
		`SELECT COUNT(*) FROM "track_playlist" WHERE playlist_id = $1 AND track_id = $2`,
		playlistID, trackID,
	))

	statusCode, _ = s.performJSONRequest(
		http.MethodDelete,
		fmt.Sprintf("/playlists/%d/tracks/%d", playlistID, trackID),
		nil,
		ownerResp.Body.AccessToken,
	)
	s.Equal(http.StatusOK, statusCode)
	s.Equal(0, s.rowCount(
		ctx,
		`SELECT COUNT(*) FROM "track_playlist" WHERE playlist_id = $1 AND track_id = $2`,
		playlistID, trackID,
	))

	statusCode, _ = s.performJSONRequest(
		http.MethodDelete,
		fmt.Sprintf("/playlists/%d/tracks/%d", playlistID, trackID),
		nil,
		ownerResp.Body.AccessToken,
	)
	s.Equal(http.StatusOK, statusCode)
}

func (s *IntegrationTestSuite) TestPutTrackToPlaylistIsIdempotent() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ownerResp := s.registerUser(api.UserAuth{
		Username: "put-playlist-track-owner",
		Password: "password-1",
	})
	s.Equal(http.StatusCreated, ownerResp.StatusCode)

	trackID, _ := s.createDownloadableTrack(ctx, ownerResp.Body.UserId)
	playlist, err := s.env.Queries.CreatePlaylist(ctx, db.CreatePlaylistParams{
		Name:     "put-playlist-track",
		IsPublic: false,
		OwnerID:  ownerResp.Body.UserId,
	})
	s.Require().NoError(err)

	statusCode, _ := s.performJSONRequest(
		http.MethodPut,
		fmt.Sprintf("/playlists/%d/tracks/%d", playlist.ID, trackID),
		nil,
		ownerResp.Body.AccessToken,
	)
	s.Equal(http.StatusOK, statusCode)
	s.Equal(1, s.rowCount(
		ctx,
		`SELECT COUNT(*) FROM "track_playlist" WHERE playlist_id = $1 AND track_id = $2`,
		playlist.ID, trackID,
	))

	statusCode, _ = s.performJSONRequest(
		http.MethodPut,
		fmt.Sprintf("/playlists/%d/tracks/%d", playlist.ID, trackID),
		nil,
		ownerResp.Body.AccessToken,
	)
	s.Equal(http.StatusOK, statusCode)
	s.Equal(1, s.rowCount(
		ctx,
		`SELECT COUNT(*) FROM "track_playlist" WHERE playlist_id = $1 AND track_id = $2`,
		playlist.ID, trackID,
	))
}

func (s *IntegrationTestSuite) TestLegacyPostTrackToPlaylistStillWorks() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ownerResp := s.registerUser(api.UserAuth{
		Username: "legacy-add-playlist-track-owner",
		Password: "password-1",
	})
	s.Equal(http.StatusCreated, ownerResp.StatusCode)

	trackID, _ := s.createDownloadableTrack(ctx, ownerResp.Body.UserId)
	playlist, err := s.env.Queries.CreatePlaylist(ctx, db.CreatePlaylistParams{
		Name:     "legacy-add-playlist-track",
		IsPublic: false,
		OwnerID:  ownerResp.Body.UserId,
	})
	s.Require().NoError(err)

	statusCode, _ := s.performJSONRequest(
		http.MethodPost,
		fmt.Sprintf("/playlists/%d", playlist.ID),
		api.AddTrackToPlaylistJSONBody{TrackId: trackID},
		ownerResp.Body.AccessToken,
	)
	s.Equal(http.StatusOK, statusCode)
	s.Equal(1, s.rowCount(
		ctx,
		`SELECT COUNT(*) FROM "track_playlist" WHERE playlist_id = $1 AND track_id = $2`,
		playlist.ID, trackID,
	))

	statusCode, _ = s.performJSONRequest(
		http.MethodPost,
		fmt.Sprintf("/playlists/%d", playlist.ID),
		api.AddTrackToPlaylistJSONBody{TrackId: trackID},
		ownerResp.Body.AccessToken,
	)
	s.Equal(http.StatusConflict, statusCode)
}
