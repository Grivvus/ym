package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"time"

	"github.com/Grivvus/ym/internal/api"
	"github.com/Grivvus/ym/internal/db"
	"github.com/Grivvus/ym/internal/storage"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/oapi-codegen/runtime/types"
)

type AlbumCreateParams struct {
	ArtistID    int32
	Name        string
	ReleaseYear *int32
	ReleaseDate *time.Time
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
		return ArtworkOwner{}, fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}
	return ArtworkOwner{ID: album.ID, Name: album.Name, Kind: "album"}, nil
}

func (s *AlbumService) Create(
	ctx context.Context, albumInfo AlbumCreateParams,
	coverFileHeader *multipart.FileHeader,
) (api.AlbumCreateResponse, error) {
	var ret api.AlbumCreateResponse
	var album = db.CreateAlbumParams{
		Name:            albumInfo.Name,
		ArtistID:        albumInfo.ArtistID,
		ReleaseYear:     intPtrToDBInt(albumInfo.ReleaseYear),
		ReleaseFullDate: datePtrToDBDate(albumInfo.ReleaseDate),
	}
	albumRet, err := s.queries.CreateAlbum(ctx, album)
	if err != nil {
		if e, ok := errors.AsType[*pgconn.PgError](err); ok && e.Code == "23505" {
			return ret, NewErrAlreadyExists("album", albumInfo.Name)
		}
		return ret, fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}
	ret.AlbumId = albumRet.ID
	ret.CoverUploaded = coverFileHeader != nil

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
		ret.CoverUploaded = false
	}
	return ret, nil
}

func (s *AlbumService) Get(
	ctx context.Context, albumID int32,
) (api.AlbumInfoResponse, error) {
	var ret api.AlbumInfoResponse

	// check if album exists
	album, err := s.queries.GetAlbum(ctx, albumID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ret, NewErrNotFound("album", albumID)
		}
		return ret, fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}

	albumTracks, err := s.queries.GetAlbumWithTracks(ctx, albumID)
	if err != nil {
		return ret, fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}
	ret.AlbumId = album.ID
	ret.AlbumName = album.Name
	ret.ReleaseYear = dbIntToIntPtr(album.ReleaseYear)
	ret.ReleaseFullDate = dbDateToSwaggerDate(album.ReleaseFullDate)
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
		if _, ok := errors.AsType[ErrNotFound](err); !ok {
			return ret, err
		}
	}
	err = s.queries.DeleteAlbum(ctx, albumID)
	if err != nil {
		return ret, fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
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

func intPtrToDBInt(iptr *int32) pgtype.Int4 {
	if iptr == nil {
		return pgtype.Int4{Valid: false}
	}
	return pgtype.Int4{Valid: true, Int32: *iptr}
}

func datePtrToDBDate(dt *time.Time) pgtype.Date {
	if dt == nil {
		return pgtype.Date{Valid: false}
	}
	return pgtype.Date{Valid: true, Time: *dt}
}

func dbDateToSwaggerDate(dt pgtype.Date) *types.Date {
	if !dt.Valid {
		return nil
	}
	return &types.Date{
		Time: dt.Time,
	}
}

func dbIntToIntPtr(i pgtype.Int4) *int32 {
	if !i.Valid {
		return nil
	}
	return &i.Int32
}
