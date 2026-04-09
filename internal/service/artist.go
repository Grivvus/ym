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
	"github.com/jackc/pgx/v5/pgtype"
)

type ArtistService struct {
	queries        *db.Queries
	st             storage.Storage
	logger         *slog.Logger
	artworkService ArtworkManager
}

func NewArtistService(q *db.Queries, st storage.Storage, logger *slog.Logger) ArtistService {
	svc := ArtistService{
		queries: q,
		st:      st,
		logger:  logger,
	}

	svc.artworkService = NewArtworkManager(st, svc.loadArtworkOwner, logger)

	return svc
}

func (s *ArtistService) loadArtworkOwner(
	ctx context.Context, artistID int32,
) (ArtworkOwner, error) {
	artist, err := s.queries.GetArtist(ctx, artistID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ArtworkOwner{}, NewErrNotFound("artist", artistID)
		}
		return ArtworkOwner{}, fmt.Errorf("%w, cause: %w", ErrUnknownDBError, err)
	}
	return ArtworkOwner{
		ID:   artist.ID,
		Name: artist.Name,
		Kind: "artist",
	}, nil
}

func (s *ArtistService) Get(ctx context.Context, id int32) (api.ArtistInfoResponse, error) {
	var ret api.ArtistInfoResponse

	var (
		artistID   int32
		artistName string
	)

	artistWithAlbums, err := s.queries.GetArtistWithAlbums(ctx, id)
	if len(artistWithAlbums) == 0 || errors.Is(err, pgx.ErrNoRows) {
		artist, err := s.queries.GetArtist(ctx, id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ret, NewErrNotFound("artist", id)
			}
			return ret, fmt.Errorf("unknown server error: %w", err)
		}
		artistID = artist.ID
		artistName = artist.Name
	} else if err != nil {
		return ret, fmt.Errorf("unknown server error: %w", err)
	}

	albums := make([]int32, len(artistWithAlbums))
	for i, album := range artistWithAlbums {
		albums[i] = album.AlbumID
	}

	if len(artistWithAlbums) > 0 {
		artistID = artistWithAlbums[0].ArtistID
		artistName = artistWithAlbums[0].ArtistName
	}

	ret.ArtistId = artistID
	ret.ArtistName = artistName
	ret.ArtistAlbums = albums

	return ret, nil
}

func (s *ArtistService) GetAll(ctx context.Context) ([]api.ArtistInfoResponse, error) {
	dbArtists, err := s.queries.GetAllArtists(ctx)
	if err != nil {
		return nil, fmt.Errorf("unknown db error: %w", err)
	}
	artists := make([]api.ArtistInfoResponse, len(dbArtists))
	for i, artist := range dbArtists {
		artists[i] = api.ArtistInfoResponse{
			ArtistId:     artist.ID,
			ArtistName:   artist.Name,
			ArtistAlbums: []int32{},
		}
	}
	return artists, nil
}

func (s *ArtistService) GetWithFilters(
	ctx context.Context, nameStartsWith string, limit int,
) ([]api.ArtistInfoResponse, error) {
	dbArtists, err := s.queries.GetArtistsWithFilter(ctx, db.GetArtistsWithFilterParams{
		Column1: pgtype.Text{Valid: true, String: nameStartsWith},
		Limit:   int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("unknown db error: %w", err)
	}
	artists := make([]api.ArtistInfoResponse, len(dbArtists))
	for i, artist := range dbArtists {
		artists[i] = api.ArtistInfoResponse{
			ArtistId:     artist.ID,
			ArtistName:   artist.Name,
			ArtistAlbums: []int32{},
		}
	}
	return artists, nil
}

func (s *ArtistService) Delete(ctx context.Context, id int32) (api.ArtistDeleteResponse, error) {
	ret := api.ArtistDeleteResponse{ArtistId: id}
	_, err := s.queries.GetArtist(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ret, nil
		}
		return ret, fmt.Errorf("%w, cause: %w", ErrUnknownDBError, err)
	}
	err = s.artworkService.Delete(ctx, id)
	if err != nil {
		return ret, nil
	}
	err = s.queries.DeleteArtist(ctx, id)
	if err != nil {
		return ret, fmt.Errorf("%w, cause: %w", ErrUnknownDBError, err)
	}
	return ret, nil
}

func (s *ArtistService) Create(
	ctx context.Context, artistName string,
	artistImage *multipart.FileHeader,
) (api.ArtistCreateResponse, error) {
	var ret api.ArtistCreateResponse
	artist, err := s.queries.CreateArtist(ctx, artistName)
	if err != nil {
		return ret, fmt.Errorf("unknown server error: %w", err)
	}

	ret.ArtistId = artist.ID
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
		go func() {
			_ = s.queries.DeleteArtist(ctx, ret.ArtistId)
		}()
		return ret, err
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
