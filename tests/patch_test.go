package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Grivvus/ym/internal/api"
	"github.com/Grivvus/ym/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
)

func (s *IntegrationTestSuite) TestPatchArtistRequiresSuperuserAndDetectsConflict() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	superResp := s.registerUser(api.UserAuth{
		Username: "patch-artist-super",
		Password: "password-1",
	})
	userResp := s.registerUser(api.UserAuth{
		Username: "patch-artist-user",
		Password: "password-1",
	})
	s.Equal(http.StatusCreated, superResp.StatusCode)
	s.Equal(http.StatusCreated, userResp.StatusCode)

	artist, err := s.env.Queries.CreateArtist(ctx, "patch-artist-old")
	s.Require().NoError(err)
	conflictingArtist, err := s.env.Queries.CreateArtist(ctx, "patch-artist-conflict")
	s.Require().NoError(err)

	statusCode, _ := s.performJSONRequest(
		http.MethodPatch,
		fmt.Sprintf("/artists/%d", artist.ID),
		map[string]any{"artist_name": "patch-artist-denied"},
		userResp.Body.AccessToken,
	)
	s.Equal(http.StatusForbidden, statusCode)

	statusCode, respBody := s.performJSONRequest(
		http.MethodPatch,
		fmt.Sprintf("/artists/%d", artist.ID),
		map[string]any{"artist_name": "patch-artist-new"},
		superResp.Body.AccessToken,
	)
	s.Equal(http.StatusOK, statusCode)
	var artistResp api.ArtistInfoResponse
	s.Require().NoError(json.Unmarshal(respBody, &artistResp))
	s.Equal(artist.ID, artistResp.ArtistId)
	s.Equal("patch-artist-new", artistResp.ArtistName)

	statusCode, _ = s.performJSONRequest(
		http.MethodPatch,
		fmt.Sprintf("/artists/%d", artist.ID),
		map[string]any{"artist_name": conflictingArtist.Name},
		superResp.Body.AccessToken,
	)
	s.Equal(http.StatusConflict, statusCode)
}

func (s *IntegrationTestSuite) TestPatchAlbumUpdatesNullableFieldsAndRequiresSuperuser() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	superResp := s.registerUser(api.UserAuth{
		Username: "patch-album-super",
		Password: "password-1",
	})
	userResp := s.registerUser(api.UserAuth{
		Username: "patch-album-user",
		Password: "password-1",
	})
	s.Equal(http.StatusCreated, superResp.StatusCode)
	s.Equal(http.StatusCreated, userResp.StatusCode)

	artist, err := s.env.Queries.CreateArtist(ctx, "patch-album-artist-old")
	s.Require().NoError(err)
	newArtist, err := s.env.Queries.CreateArtist(ctx, "patch-album-artist-new")
	s.Require().NoError(err)
	album, err := s.env.Queries.CreateAlbum(ctx, db.CreateAlbumParams{
		Name:            "patch-album-old",
		ArtistID:        artist.ID,
		ReleaseYear:     pgtype.Int4{Int32: 2001, Valid: true},
		ReleaseFullDate: pgtype.Date{Time: time.Date(2001, 2, 3, 0, 0, 0, 0, time.UTC), Valid: true},
	})
	s.Require().NoError(err)

	statusCode, _ := s.performJSONRequest(
		http.MethodPatch,
		fmt.Sprintf("/albums/%d", album.ID),
		map[string]any{
			"album_name":        "patch-album-denied",
			"artist_id":         artist.ID,
			"release_year":      2001,
			"release_full_date": "2001-02-03",
		},
		userResp.Body.AccessToken,
	)
	s.Equal(http.StatusForbidden, statusCode)

	statusCode, respBody := s.performJSONRequest(
		http.MethodPatch,
		fmt.Sprintf("/albums/%d", album.ID),
		map[string]any{
			"album_name":        "patch-album-new",
			"artist_id":         newArtist.ID,
			"release_year":      nil,
			"release_full_date": nil,
		},
		superResp.Body.AccessToken,
	)
	s.Equal(http.StatusOK, statusCode)
	var albumResp api.AlbumInfoResponse
	s.Require().NoError(json.Unmarshal(respBody, &albumResp))
	s.Equal(album.ID, albumResp.AlbumId)
	s.Equal("patch-album-new", albumResp.AlbumName)
	s.Equal(newArtist.ID, albumResp.ArtistId)
	s.Nil(albumResp.ReleaseYear)
	s.Nil(albumResp.ReleaseFullDate)
}

func (s *IntegrationTestSuite) TestPatchTrackUpdatesMetadataAndRequiresSuperuser() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	superResp := s.registerUser(api.UserAuth{
		Username: "patch-track-super",
		Password: "password-1",
	})
	userResp := s.registerUser(api.UserAuth{
		Username: "patch-track-user",
		Password: "password-1",
	})
	s.Equal(http.StatusCreated, superResp.StatusCode)
	s.Equal(http.StatusCreated, userResp.StatusCode)

	artist, err := s.env.Queries.CreateArtist(ctx, "patch-track-artist-old")
	s.Require().NoError(err)
	newArtist, err := s.env.Queries.CreateArtist(ctx, "patch-track-artist-new")
	s.Require().NoError(err)
	album, err := s.env.Queries.CreateAlbum(ctx, db.CreateAlbumParams{
		Name:     "patch-track-album-old",
		ArtistID: artist.ID,
	})
	s.Require().NoError(err)
	newAlbum, err := s.env.Queries.CreateAlbum(ctx, db.CreateAlbumParams{
		Name:     "patch-track-album-new",
		ArtistID: newArtist.ID,
	})
	s.Require().NoError(err)
	track, err := s.env.Queries.CreateTrack(ctx, db.CreateTrackParams{
		Name:                "patch-track-old",
		ArtistID:            artist.ID,
		IsGloballyAvailable: false,
		UploadByUser:        pgtype.Int4{Int32: superResp.Body.UserId, Valid: true},
	})
	s.Require().NoError(err)
	s.Require().NoError(s.env.Queries.AddTrackToAlbum(ctx, db.AddTrackToAlbumParams{
		TrackID: track.ID,
		AlbumID: album.ID,
	}))

	statusCode, _ := s.performJSONRequest(
		http.MethodPatch,
		fmt.Sprintf("/tracks/%d", track.ID),
		map[string]any{
			"name":                  "patch-track-denied",
			"artist_id":             artist.ID,
			"album_id":              album.ID,
			"is_globally_available": false,
		},
		userResp.Body.AccessToken,
	)
	s.Equal(http.StatusForbidden, statusCode)

	statusCode, respBody := s.performJSONRequest(
		http.MethodPatch,
		fmt.Sprintf("/tracks/%d", track.ID),
		map[string]any{
			"name":                  "patch-track-new",
			"artist_id":             newArtist.ID,
			"album_id":              newAlbum.ID,
			"is_globally_available": true,
		},
		superResp.Body.AccessToken,
	)
	s.Equal(http.StatusOK, statusCode)
	var trackResp api.TrackMetadata
	s.Require().NoError(json.Unmarshal(respBody, &trackResp))
	s.Equal(track.ID, trackResp.TrackId)
	s.Equal("patch-track-new", trackResp.Name)
	s.Equal(newArtist.ID, trackResp.ArtistId)
	s.Equal(newAlbum.ID, trackResp.AlbumId)
	s.True(trackResp.IsGloballyAvailable)
	s.Equal(1, s.rowCount(
		ctx,
		`SELECT COUNT(*) FROM "track_album" WHERE track_id = $1 AND album_id = $2`,
		track.ID, newAlbum.ID,
	))
}

func (s *IntegrationTestSuite) TestPatchPlaylistCanTogglePublicAndRename() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ownerResp := s.registerUser(api.UserAuth{
		Username: "patch-playlist-owner",
		Password: "password-1",
	})
	s.Equal(http.StatusCreated, ownerResp.StatusCode)

	playlist, err := s.env.Queries.CreatePlaylist(ctx, db.CreatePlaylistParams{
		Name:     "patch-playlist-old",
		IsPublic: false,
		OwnerID:  ownerResp.Body.UserId,
	})
	s.Require().NoError(err)

	statusCode, respBody := s.performJSONRequest(
		http.MethodPatch,
		fmt.Sprintf("/playlists/%d", playlist.ID),
		map[string]any{
			"playlist_name": "patch-playlist-old",
			"is_public":     true,
		},
		ownerResp.Body.AccessToken,
	)
	s.Equal(http.StatusOK, statusCode)
	var playlistResp api.PlaylistResponse
	s.Require().NoError(json.Unmarshal(respBody, &playlistResp))
	s.Equal(playlist.ID, playlistResp.PlaylistId)
	s.Equal("patch-playlist-old", playlistResp.PlaylistName)
	s.True(playlistResp.IsPublic)

	statusCode, respBody = s.performJSONRequest(
		http.MethodPatch,
		fmt.Sprintf("/playlists/%d", playlist.ID),
		map[string]any{
			"playlist_name": "patch-playlist-new",
			"is_public":     true,
		},
		ownerResp.Body.AccessToken,
	)
	s.Equal(http.StatusOK, statusCode)
	s.Require().NoError(json.Unmarshal(respBody, &playlistResp))
	s.Equal("patch-playlist-new", playlistResp.PlaylistName)
	s.True(playlistResp.IsPublic)
}
