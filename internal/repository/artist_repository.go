package repository

import (
	"context"

	"github.com/Grivvus/ym/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ArtistRepository interface {
	GetArtist(ctx context.Context, artistID int32) (Artist, error)
	GetArtistInfo(ctx context.Context, artistID int32) (ArtistInfo, error)
	GetAllArtists(ctx context.Context) ([]ArtistInfo, error)
	GetArtistsWithFilter(ctx context.Context, nameStartsWith string, limit int) ([]ArtistInfo, error)
	CreateArtist(ctx context.Context, artistName string) (Artist, error)
	DeleteArtist(ctx context.Context, artistID int32) error
}

type Artist struct {
	ID   int32
	Name string
}

type ArtistInfo struct {
	ID       int32
	Name     string
	AlbumIDs []int32
}

type PostgresArtistRepository struct {
	queries *db.Queries
}

func NewArtistRepository(pool *pgxpool.Pool) *PostgresArtistRepository {
	return &PostgresArtistRepository{
		queries: db.New(pool),
	}
}

func (repo *PostgresArtistRepository) GetArtist(
	ctx context.Context, artistID int32,
) (Artist, error) {
	artist, err := repo.queries.GetArtist(ctx, artistID)
	if err != nil {
		return Artist{}, wrapDBError(err)
	}
	return Artist{
		ID:   artist.ID,
		Name: artist.Name,
	}, nil
}

func (repo *PostgresArtistRepository) GetArtistInfo(
	ctx context.Context, artistID int32,
) (ArtistInfo, error) {
	artistAlbums, err := repo.queries.GetArtistWithAlbums(ctx, artistID)
	if err != nil {
		return ArtistInfo{}, wrapDBError(err)
	}
	if len(artistAlbums) == 0 {
		artist, err := repo.GetArtist(ctx, artistID)
		if err != nil {
			return ArtistInfo{}, err
		}
		return ArtistInfo{
			ID:       artist.ID,
			Name:     artist.Name,
			AlbumIDs: []int32{},
		}, nil
	}

	albumIDs := make([]int32, len(artistAlbums))
	for i, album := range artistAlbums {
		albumIDs[i] = album.AlbumID
	}
	return ArtistInfo{
		ID:       artistAlbums[0].ArtistID,
		Name:     artistAlbums[0].ArtistName,
		AlbumIDs: albumIDs,
	}, nil
}

func (repo *PostgresArtistRepository) GetAllArtists(ctx context.Context) ([]ArtistInfo, error) {
	artists, err := repo.queries.GetAllArtists(ctx)
	if err != nil {
		return nil, wrapDBError(err)
	}
	result := make([]Artist, len(artists))
	for i, artist := range artists {
		result[i] = Artist{
			ID:   artist.ID,
			Name: artist.Name,
		}
	}
	return repo.artistInfosFromArtists(ctx, result)
}

func (repo *PostgresArtistRepository) GetArtistsWithFilter(
	ctx context.Context, nameStartsWith string, limit int,
) ([]ArtistInfo, error) {
	artists, err := repo.queries.GetArtistsWithFilter(ctx, db.GetArtistsWithFilterParams{
		Column1: pgtype.Text{String: nameStartsWith, Valid: true},
		Limit:   int32(limit),
	})
	if err != nil {
		return nil, wrapDBError(err)
	}
	result := make([]Artist, len(artists))
	for i, artist := range artists {
		result[i] = Artist{
			ID:   artist.ID,
			Name: artist.Name,
		}
	}
	return repo.artistInfosFromArtists(ctx, result)
}

func (repo *PostgresArtistRepository) CreateArtist(
	ctx context.Context, artistName string,
) (Artist, error) {
	artist, err := repo.queries.CreateArtist(ctx, artistName)
	if err != nil {
		return Artist{}, wrapDBError(err)
	}
	return Artist{
		ID:   artist.ID,
		Name: artist.Name,
	}, nil
}

func (repo *PostgresArtistRepository) DeleteArtist(ctx context.Context, artistID int32) error {
	err := repo.queries.DeleteArtist(ctx, artistID)
	return wrapDBError(err)
}

func (repo *PostgresArtistRepository) artistInfosFromArtists(
	ctx context.Context, artists []Artist,
) ([]ArtistInfo, error) {
	artistInfos := make([]ArtistInfo, len(artists))
	for i, artist := range artists {
		albumIDs, err := repo.getArtistAlbumIDs(ctx, artist.ID)
		if err != nil {
			return nil, err
		}
		artistInfos[i] = ArtistInfo{
			ID:       artist.ID,
			Name:     artist.Name,
			AlbumIDs: albumIDs,
		}
	}
	return artistInfos, nil
}

func (repo *PostgresArtistRepository) getArtistAlbumIDs(
	ctx context.Context, artistID int32,
) ([]int32, error) {
	albums, err := repo.queries.GetArtistWithAlbums(ctx, artistID)
	if err != nil {
		return nil, wrapDBError(err)
	}
	albumIDs := make([]int32, len(albums))
	for i, album := range albums {
		albumIDs[i] = album.AlbumID
	}
	return albumIDs, nil
}
