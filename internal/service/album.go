package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/Grivvus/ym/internal/api"
	"github.com/Grivvus/ym/internal/db"
	"github.com/Grivvus/ym/internal/storage"
	"github.com/jackc/pgx/v5"
)

type AlbumService struct {
	queries *db.Queries
	st      storage.Storage
}

func NewAlbumService(q *db.Queries, st storage.Storage) AlbumService {
	return AlbumService{
		queries: q,
		st:      st,
	}
}

func (s *AlbumService) Create(
	ctx context.Context, albumInfo api.AlbumCreateRequest,
) (api.AlbumCreateResponse, error) {
	var ret api.AlbumCreateResponse
	var album = db.CreateAlbumParams{
		Name:     albumInfo.AlbumName,
		ArtistID: int32(albumInfo.OwnerId),
	}
	albumRet, err := s.queries.CreateAlbum(ctx, album)
	if err != nil {
		return ret, fmt.Errorf("unkown server error: %w", err)
	}
	if albumInfo.AlbumCover != nil {
		// upload album cover
	}
	return api.AlbumCreateResponse{
		AlbumId: int(albumRet.ID),
	}, nil
}

func (s *AlbumService) Get(
	ctx context.Context, albumID int,
) (api.AlbumInfoResponse, error) {
	var ret api.AlbumInfoResponse
	albumTracks, err := s.queries.GetAlbumWithTracks(ctx, int32(albumID))
	if err != nil {
		// no tracks in album, but album exists
		// could be valid state
		if errors.Is(err, pgx.ErrNoRows) {
			return ret, NewErrNotFound("album", albumID)
		}
	}
	ret.AlbumId = albumID
	ret.AlbumName = albumTracks[0].Name
	for _, t := range albumTracks {
		ret.Tracks = append(ret.Tracks, int(t.TrackID))
	}
	return ret, nil
}

func (s *AlbumService) Delete(
	ctx context.Context, albumID int,
) (api.AlbumDeleteResponse, error) {
	var ret = api.AlbumDeleteResponse{AlbumId: albumID}
	album, err := s.queries.GetAlbum(ctx, int32(albumID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ret, nil
		}
	}
	err = s.st.RemoveImage(ctx, ImageID("album", int(album.ID), album.Name))
	if err != nil {
		return ret, fmt.Errorf("Can't delete image: %w", err)
	}
	err = s.queries.DeleteAlbum(ctx, int32(albumID))
	if err != nil {
		return ret, err
	}
	return ret, nil
}
