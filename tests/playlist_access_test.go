package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Grivvus/ym/internal/api"
)

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

	statusCode, _ = s.performJSONRequest(
		http.MethodPut,
		fmt.Sprintf("/playlists/%d/tracks/%d", playlistID, trackID),
		nil,
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

	statusCode, _ = s.performJSONRequest(
		http.MethodPut,
		fmt.Sprintf("/playlists/%d/tracks/%d", playlistID, privateTrackID),
		nil,
		writerResp.Body.AccessToken,
	)
	s.Equal(http.StatusForbidden, statusCode)
}
