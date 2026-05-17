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
	"github.com/Grivvus/ym/internal/repository"
	"github.com/Grivvus/ym/internal/storage"
	"github.com/oapi-codegen/runtime/types"
)

type AlbumCreateParams struct {
	ArtistID    int32
	Name        string
	ReleaseYear *int32
	ReleaseDate *time.Time
}

type AlbumService struct {
	repo           repository.AlbumRepository
	objStorage     storage.Storage
	logger         *slog.Logger
	artworkService ArtworkManager
}

func NewAlbumService(
	repo repository.AlbumRepository, st storage.Storage, logger *slog.Logger,
) AlbumService {
	svc := AlbumService{
		repo:       repo,
		objStorage: st,
		logger:     logger,
	}
	svc.artworkService = NewArtworkManager(st, svc.loadArtworkOwner, logger)

	return svc
}

func (s *AlbumService) loadArtworkOwner(
	ctx context.Context, albumID int32,
) (ArtworkOwner, error) {
	album, err := s.repo.GetAlbum(ctx, albumID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
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
	var album = repository.CreateAlbumParams{
		Name:        albumInfo.Name,
		ArtistID:    albumInfo.ArtistID,
		ReleaseYear: albumInfo.ReleaseYear,
		ReleaseDate: albumInfo.ReleaseDate,
	}
	albumRet, err := s.repo.CreateAlbum(ctx, album)
	if err != nil {
		if errors.Is(err, repository.ErrAlreadyExists) {
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

	album, err := s.repo.GetAlbumInfo(ctx, albumID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ret, NewErrNotFound("album", albumID)
		}
		return ret, fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}

	ret.AlbumId = album.ID
	ret.AlbumName = album.Name
	ret.ReleaseYear = album.ReleaseYear
	ret.ReleaseFullDate = timePtrToSwaggerDate(album.ReleaseDate)
	ret.Tracks = album.TrackIDs
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
	err = s.repo.DeleteAlbum(ctx, albumID)
	if err != nil {
		return ret, fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}
	return ret, nil
}

func (s *AlbumService) DeleteTrack(ctx context.Context, albumID, trackID int32) error {
	if _, err := s.repo.GetAlbum(ctx, albumID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return NewErrNotFound("album", albumID)
		}
		return fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}
	err := s.repo.DeleteTrackFromAlbum(ctx, albumID, trackID)
	if err != nil {
		return fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}
	return nil
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

func timePtrToSwaggerDate(dt *time.Time) *types.Date {
	if dt == nil {
		return nil
	}
	return &types.Date{
		Time: *dt,
	}
}
