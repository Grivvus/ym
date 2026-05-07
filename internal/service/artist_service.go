package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"

	"github.com/Grivvus/ym/internal/api"
	"github.com/Grivvus/ym/internal/repository"
	"github.com/Grivvus/ym/internal/storage"
)

type ArtistService struct {
	repo           repository.ArtistRepository
	objStorage     storage.Storage
	logger         *slog.Logger
	artworkService ArtworkManager
}

func NewArtistService(
	repo repository.ArtistRepository, st storage.Storage, logger *slog.Logger,
) ArtistService {
	svc := ArtistService{
		repo:       repo,
		objStorage: st,
		logger:     logger,
	}

	svc.artworkService = NewArtworkManager(st, svc.loadArtworkOwner, logger)

	return svc
}

func (s *ArtistService) loadArtworkOwner(
	ctx context.Context, artistID int32,
) (ArtworkOwner, error) {
	artist, err := s.repo.GetArtist(ctx, artistID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ArtworkOwner{}, NewErrNotFound("artist", artistID)
		}
		return ArtworkOwner{}, fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}
	return ArtworkOwner{
		ID:   artist.ID,
		Name: artist.Name,
		Kind: "artist",
	}, nil
}

func (s *ArtistService) Get(ctx context.Context, id int32) (api.ArtistInfoResponse, error) {
	var ret api.ArtistInfoResponse

	artist, err := s.repo.GetArtistInfo(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ret, NewErrNotFound("artist", id)
		}
		return ret, fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}
	return apiArtistInfoFromRepositoryArtistInfo(artist), nil
}

func (s *ArtistService) GetAll(ctx context.Context) ([]api.ArtistInfoResponse, error) {
	artists, err := s.repo.GetAllArtists(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}
	return apiArtistsFromRepositoryArtists(artists), nil
}

func (s *ArtistService) GetWithFilters(
	ctx context.Context, nameStartsWith string, limit int,
) ([]api.ArtistInfoResponse, error) {
	artists, err := s.repo.GetArtistsWithFilter(ctx, nameStartsWith, limit)
	if err != nil {
		return nil, fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}
	return apiArtistsFromRepositoryArtists(artists), nil
}

func (s *ArtistService) Delete(ctx context.Context, id int32) (api.ArtistDeleteResponse, error) {
	ret := api.ArtistDeleteResponse{ArtistId: id}
	err := s.artworkService.Delete(ctx, id)
	if err != nil {
		if _, ok := errors.AsType[ErrNotFound](err); !ok {
			return ret, err
		}
	}
	err = s.repo.DeleteArtist(ctx, id)
	if err != nil {
		return ret, fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}
	return ret, nil
}

func (s *ArtistService) Create(
	ctx context.Context, artistName string,
	artistImage *multipart.FileHeader,
) (api.ArtistCreateResponse, error) {
	var ret api.ArtistCreateResponse
	artist, err := s.repo.CreateArtist(ctx, artistName)
	if err != nil {
		if errors.Is(err, repository.ErrAlreadyExists) {
			return ret, NewErrAlreadyExists("artist", artistName)
		}
		return ret, fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}

	ret.ArtistId = artist.ID
	ret.CoverUploaded = artistImage != nil
	if artistImage == nil {
		return ret, nil
	}

	rc, err := artistImage.Open()
	if err != nil {
		return ret, fmt.Errorf("%w: assertion, must be nil", err)
	}
	defer func() { _ = rc.Close() }()

	err = s.UploadImage(ctx, ret.ArtistId, rc)
	if err != nil {
		ret.CoverUploaded = false
	}
	return ret, nil
}

func (s *ArtistService) UploadImage(
	ctx context.Context, artistID int32, file io.Reader,
) error {
	return s.artworkService.Upload(ctx, artistID, file)
}

func (s *ArtistService) DeleteImage(
	ctx context.Context, artistID int32,
) error {
	return s.artworkService.Delete(ctx, artistID)
}

func (s *ArtistService) GetImage(
	ctx context.Context, artistID int32,
) ([]byte, error) {
	return s.artworkService.Get(ctx, artistID)
}

func apiArtistsFromRepositoryArtists(
	artists []repository.ArtistInfo,
) []api.ArtistInfoResponse {
	result := make([]api.ArtistInfoResponse, len(artists))
	for i, artist := range artists {
		result[i] = apiArtistInfoFromRepositoryArtistInfo(artist)
	}
	return result
}

func apiArtistInfoFromRepositoryArtistInfo(
	artist repository.ArtistInfo,
) api.ArtistInfoResponse {
	return api.ArtistInfoResponse{
		ArtistId:     artist.ID,
		ArtistName:   artist.Name,
		ArtistAlbums: artist.AlbumIDs,
	}
}
