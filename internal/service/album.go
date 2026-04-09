package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"

	"github.com/Grivvus/ym/internal/api"
	"github.com/Grivvus/ym/internal/db"
	"github.com/Grivvus/ym/internal/storage"
	"github.com/jackc/pgx/v5"
)

type AlbumCreateParams struct {
	ArtistID int32
	Name     string
}

type AlbumService struct {
	queries        *db.Queries
	st             storage.Storage
	logger         *slog.Logger
	artworkService ArtworkManager
}

func NewAlbumService(q *db.Queries, st storage.Storage, logger *slog.Logger) AlbumService {
	svc := AlbumService{
		queries: q,
		st:      st,
		logger:  logger,
	}
	svc.artworkService = NewArtworkManager(st, svc.loadArtworkOwner, logger)

	return svc
}

func (s *AlbumService) loadArtworkOwner(
	ctx context.Context, albumID int32,
) (ArtworkOwner, error) {
	album, err := s.queries.GetAlbum(ctx, albumID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ArtworkOwner{}, NewErrNotFound("album", albumID)
		}
		return ArtworkOwner{}, fmt.Errorf("%w, cause: %w", ErrUnknownDBError, err)
	}
	return ArtworkOwner{ID: album.ID, Name: album.Name, Kind: "album"}, nil
}

func (s *AlbumService) Create(
	ctx context.Context, albumInfo AlbumCreateParams,
	coverFileHeader *multipart.FileHeader,
) (api.AlbumCreateResponse, error) {
	var ret api.AlbumCreateResponse
	var album = db.CreateAlbumParams{
		Name:     albumInfo.Name,
		ArtistID: albumInfo.ArtistID,
	}
	albumRet, err := s.queries.CreateAlbum(ctx, album)
	if err != nil {
		return ret, fmt.Errorf("unknown server error: %w", err)
	}
	ret.AlbumId = albumRet.ID

	if coverFileHeader == nil {
		return ret, nil
	}

	rc, err := coverFileHeader.Open()
	if err != nil {
		return ret, fmt.Errorf("%w: assertion, must be nil", err)
	}
	defer func() { _ = rc.Close() }()

	err = s.UploadCover(ctx, albumRet.ID, rc)
	if err != nil {
		go func() {
			_ = s.queries.DeleteAlbum(ctx, int32(ret.AlbumId))
		}()
		return ret, fmt.Errorf("error while uploading album cover: %w", err)
	}
	return ret, nil
}

func (s *AlbumService) Get(
	ctx context.Context, albumID int32,
) (api.AlbumInfoResponse, error) {
	var ret api.AlbumInfoResponse
	albumTracks, err := s.queries.GetAlbumWithTracks(ctx, albumID)
	if err != nil {
		// no tracks in album, but album exists
		// could be valid state
		if errors.Is(err, pgx.ErrNoRows) {
			return ret, NewErrNotFound("album", albumID)
		}
		return ret, fmt.Errorf("unknown server error: %w", err)
	}
	ret.AlbumId = albumID
	ret.AlbumName = albumTracks[0].Name
	for _, t := range albumTracks {
		ret.Tracks = append(ret.Tracks, t.TrackID)
	}
	return ret, nil
}

func (s *AlbumService) Delete(
	ctx context.Context, albumID int32,
) (api.AlbumDeleteResponse, error) {
	var ret = api.AlbumDeleteResponse{AlbumId: albumID}
	err := s.artworkService.Delete(ctx, albumID)
	if err != nil {
		return ret, err
	}
	err = s.queries.DeleteAlbum(ctx, albumID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ret, NewErrNotFound("album", albumID)
		}
		return ret, fmt.Errorf("%w, cause: %w", ErrUnknownDBError, err)
	}
	return ret, nil
}

func (s *AlbumService) DeleteCover(
	ctx context.Context, albumID int32,
) error {
	return s.artworkService.Delete(ctx, albumID)
}

func (s *AlbumService) UploadCover(
	ctx context.Context, albumID int32, cover io.Reader,
) error {
	return s.artworkService.Upload(ctx, albumID, cover)
}

func (s *AlbumService) GetCover(
	ctx context.Context, albumID int32,
) ([]byte, error) {
	return s.artworkService.Get(ctx, albumID)
}
