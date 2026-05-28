package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Grivvus/ym/internal/api"
	"github.com/Grivvus/ym/internal/db"
	"github.com/Grivvus/ym/internal/storage"
	"github.com/jackc/pgx/v5/pgtype"
)

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

func (s *IntegrationTestSuite) TestDeleteAlbumOnlySuperuserCanDeleteAndKeepsTracks() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	superResp := s.registerUser(api.UserAuth{
		Username: "delete-album-super",
		Password: "password-1",
	})
	userResp := s.registerUser(api.UserAuth{
		Username: "delete-album-user",
		Password: "password-1",
	})
	s.Equal(http.StatusCreated, superResp.StatusCode)
	s.Equal(http.StatusCreated, userResp.StatusCode)

	artist, err := s.env.Queries.CreateArtist(ctx, "delete-album-artist")
	s.Require().NoError(err)

	album, err := s.env.Queries.CreateAlbum(ctx, db.CreateAlbumParams{
		Name:     "delete-album",
		ArtistID: artist.ID,
	})
	s.Require().NoError(err)

	track, err := s.env.Queries.CreateTrack(ctx, db.CreateTrackParams{
		Name:                "delete-album-track",
		ArtistID:            artist.ID,
		IsGloballyAvailable: false,
		UploadByUser:        pgtype.Int4{Int32: userResp.Body.UserId, Valid: true},
	})
	s.Require().NoError(err)
	s.Require().NoError(s.env.Queries.AddTrackToAlbum(ctx, db.AddTrackToAlbumParams{
		TrackID: track.ID,
		AlbumID: album.ID,
	}))

	statusCode, respBody := s.performJSONRequest(
		http.MethodDelete,
		fmt.Sprintf("/albums/%d", album.ID),
		nil,
		userResp.Body.AccessToken,
	)
	s.Equal(http.StatusForbidden, statusCode)
	var errorResp api.ErrorResponse
	s.Require().NoError(json.Unmarshal(respBody, &errorResp))
	s.Contains(errorResp.Error, "required superuser rights")

	statusCode, _ = s.performJSONRequest(
		http.MethodDelete,
		fmt.Sprintf("/albums/%d", album.ID),
		nil,
		superResp.Body.AccessToken,
	)
	s.Equal(http.StatusOK, statusCode)

	s.Equal(0, s.rowCount(ctx, `SELECT COUNT(*) FROM "album" WHERE id = $1`, album.ID))
	s.Equal(0, s.rowCount(ctx, `SELECT COUNT(*) FROM "track_album" WHERE album_id = $1`, album.ID))
	s.Equal(1, s.rowCount(ctx, `SELECT COUNT(*) FROM "track" WHERE id = $1`, track.ID))
}
