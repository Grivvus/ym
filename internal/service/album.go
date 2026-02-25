package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"

	"github.com/Grivvus/ym/internal/api"
	"github.com/Grivvus/ym/internal/db"
	"github.com/Grivvus/ym/internal/storage"
	"github.com/Grivvus/ym/internal/transcoder"
	"github.com/jackc/pgx/v5"
)

type AlbumCreateParams struct {
	ArtistID int
	Name     string
}

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
	ctx context.Context, albumInfo AlbumCreateParams,
	coverFileHeader *multipart.FileHeader,
) (api.AlbumCreateResponse, error) {
	var ret api.AlbumCreateResponse
	var album = db.CreateAlbumParams{
		Name:     albumInfo.Name,
		ArtistID: int32(albumInfo.ArtistID),
	}
	albumRet, err := s.queries.CreateAlbum(ctx, album)
	if err != nil {
		return ret, fmt.Errorf("unkown server error: %w", err)
	}
	ret.AlbumId = int(albumRet.ID)

	if coverFileHeader == nil {
		return ret, nil
	}

	rc, err := coverFileHeader.Open()
	if err != nil {
		panic(err)
	}
	defer func() { _ = rc.Close() }()

	err = s.UploadCover(ctx, int(albumRet.ID), rc)
	if err != nil {
		go func() {
			_ = s.queries.DeleteAlbum(ctx, int32(ret.AlbumId))
		}()
		return ret, fmt.Errorf("error while uploading album cover: %w", err)
	}
	return ret, nil
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
		} else {
			return ret, fmt.Errorf("unkown server error: %w", err)
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

func (s *AlbumService) DeleteCover(
	ctx context.Context, albumID int,
) error {
	album, err := s.queries.GetAlbum(ctx, int32(albumID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("can't delete image, cause: %w", err)
	}
	err = s.st.RemoveImage(ctx, ImageID("album", albumID, album.Name))
	if err != nil {
		return fmt.Errorf("can't delete image, cause: %w", err)
	}
	return nil
}

func (s *AlbumService) UploadCover(
	ctx context.Context, albumID int, cover io.Reader,
) error {
	album, err := s.queries.GetAlbum(ctx, int32(albumID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		} else {
			return fmt.Errorf("can't upload image, cause: %w", err)
		}
	}
	rcTranscoded, err := transcoder.ToWebp(cover)
	if err != nil {
		return fmt.Errorf("can't upload image, cause: %w", err)
	}
	defer func() { _ = rcTranscoded.Close() }()

	err = s.st.PutImage(ctx, ImageID("album", int(album.ID), album.Name), rcTranscoded)
	if err != nil {
		return fmt.Errorf("can't upload image, cause: %w", err)
	}
	return nil
}

func (s *AlbumService) GetCover(
	ctx context.Context, albumID int,
) ([]byte, error) {
	album, err := s.queries.GetAlbum(ctx, int32(albumID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, NewErrNotFound("album", albumID)
		}
		return nil, fmt.Errorf("unkown server error: %w", err)
	}
	bimage, err := s.st.GetImage(ctx, ImageID("album", int(album.ID), album.Name))
	if err != nil {
		return nil, fmt.Errorf("unkown server error: %w", err)
	}
	return bimage, nil
}
