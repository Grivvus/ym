package tests

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/Grivvus/ym/internal/api"
	"github.com/Grivvus/ym/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
)

func (s *IntegrationTestSuite) TestPutTrackToAlbumIsIdempotentAndRequiresSuperuser() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	superResp := s.registerUser(api.UserAuth{
		Username: "put-album-track-super",
		Password: "password-1",
	})
	userResp := s.registerUser(api.UserAuth{
		Username: "put-album-track-user",
		Password: "password-1",
	})
	s.Equal(http.StatusCreated, superResp.StatusCode)
	s.Equal(http.StatusCreated, userResp.StatusCode)

	artist, err := s.env.Queries.CreateArtist(ctx, "put-album-track-artist")
	s.Require().NoError(err)
	sourceAlbum, err := s.env.Queries.CreateAlbum(ctx, db.CreateAlbumParams{
		Name:     "put-album-track-source",
		ArtistID: artist.ID,
	})
	s.Require().NoError(err)
	targetAlbum, err := s.env.Queries.CreateAlbum(ctx, db.CreateAlbumParams{
		Name:     "put-album-track-target",
		ArtistID: artist.ID,
	})
	s.Require().NoError(err)
	track, err := s.env.Queries.CreateTrack(ctx, db.CreateTrackParams{
		Name:                "put-album-track-track",
		ArtistID:            artist.ID,
		IsGloballyAvailable: false,
		UploadByUser:        pgtype.Int4{Int32: superResp.Body.UserId, Valid: true},
	})
	s.Require().NoError(err)
	s.Require().NoError(s.env.Queries.AddTrackToAlbum(ctx, db.AddTrackToAlbumParams{
		TrackID: track.ID,
		AlbumID: sourceAlbum.ID,
	}))

	statusCode, _ := s.performJSONRequest(
		http.MethodPut,
		fmt.Sprintf("/albums/%d/tracks/%d", targetAlbum.ID, track.ID),
		nil,
		userResp.Body.AccessToken,
	)
	s.Equal(http.StatusForbidden, statusCode)
	s.Equal(0, s.rowCount(
		ctx,
		`SELECT COUNT(*) FROM "track_album" WHERE album_id = $1 AND track_id = $2`,
		targetAlbum.ID, track.ID,
	))

	statusCode, _ = s.performJSONRequest(
		http.MethodPut,
		fmt.Sprintf("/albums/%d/tracks/%d", targetAlbum.ID, track.ID),
		nil,
		superResp.Body.AccessToken,
	)
	s.Equal(http.StatusOK, statusCode)
	s.Equal(1, s.rowCount(
		ctx,
		`SELECT COUNT(*) FROM "track_album" WHERE album_id = $1 AND track_id = $2`,
		targetAlbum.ID, track.ID,
	))

	statusCode, _ = s.performJSONRequest(
		http.MethodPut,
		fmt.Sprintf("/albums/%d/tracks/%d", targetAlbum.ID, track.ID),
		nil,
		superResp.Body.AccessToken,
	)
	s.Equal(http.StatusOK, statusCode)
	s.Equal(1, s.rowCount(
		ctx,
		`SELECT COUNT(*) FROM "track_album" WHERE album_id = $1 AND track_id = $2`,
		targetAlbum.ID, track.ID,
	))
}

func (s *IntegrationTestSuite) TestDeleteTrackFromAlbumIsIdempotentAndRequiresSuperuser() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	superResp := s.registerUser(api.UserAuth{
		Username: "delete-album-track-super",
		Password: "password-1",
	})
	userResp := s.registerUser(api.UserAuth{
		Username: "delete-album-track-user",
		Password: "password-1",
	})
	s.Equal(http.StatusCreated, superResp.StatusCode)
	s.Equal(http.StatusCreated, userResp.StatusCode)

	artist, err := s.env.Queries.CreateArtist(ctx, "delete-album-track-artist")
	s.Require().NoError(err)

	album, err := s.env.Queries.CreateAlbum(ctx, db.CreateAlbumParams{
		Name:     "delete-album-track-album",
		ArtistID: artist.ID,
	})
	s.Require().NoError(err)

	track, err := s.env.Queries.CreateTrack(ctx, db.CreateTrackParams{
		Name:                "delete-album-track-track",
		ArtistID:            artist.ID,
		IsGloballyAvailable: false,
		UploadByUser:        pgtype.Int4{Int32: userResp.Body.UserId, Valid: true},
	})
	s.Require().NoError(err)
	s.Require().NoError(s.env.Queries.AddTrackToAlbum(ctx, db.AddTrackToAlbumParams{
		TrackID: track.ID,
		AlbumID: album.ID,
	}))

	statusCode, _ := s.performJSONRequest(
		http.MethodDelete,
		fmt.Sprintf("/albums/%d/tracks/%d", album.ID, track.ID),
		nil,
		userResp.Body.AccessToken,
	)
	s.Equal(http.StatusForbidden, statusCode)
	s.Equal(1, s.rowCount(
		ctx,
		`SELECT COUNT(*) FROM "track_album" WHERE album_id = $1 AND track_id = $2`,
		album.ID, track.ID,
	))

	statusCode, _ = s.performJSONRequest(
		http.MethodDelete,
		fmt.Sprintf("/albums/%d/tracks/%d", album.ID, track.ID),
		nil,
		superResp.Body.AccessToken,
	)
	s.Equal(http.StatusOK, statusCode)
	s.Equal(0, s.rowCount(
		ctx,
		`SELECT COUNT(*) FROM "track_album" WHERE album_id = $1 AND track_id = $2`,
		album.ID, track.ID,
	))
	s.Equal(1, s.rowCount(ctx, `SELECT COUNT(*) FROM "track" WHERE id = $1`, track.ID))

	statusCode, _ = s.performJSONRequest(
		http.MethodDelete,
		fmt.Sprintf("/albums/%d/tracks/%d", album.ID, track.ID),
		nil,
		superResp.Body.AccessToken,
	)
	s.Equal(http.StatusOK, statusCode)

	statusCode, _ = s.performJSONRequest(
		http.MethodDelete,
		fmt.Sprintf("/albums/%d/tracks/%d", album.ID+1000, track.ID),
		nil,
		superResp.Body.AccessToken,
	)
	s.Equal(http.StatusNotFound, statusCode)
}
