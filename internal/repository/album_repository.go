package repository

import (
	"context"
	"time"

	"github.com/Grivvus/ym/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AlbumRepository interface {
	CreateAlbum(ctx context.Context, params CreateAlbumParams) (Album, error)
	GetAlbum(ctx context.Context, albumID int32) (Album, error)
	GetAlbumInfo(ctx context.Context, albumID int32) (Album, error)
	DeleteAlbum(ctx context.Context, albumID int32) error
	DeleteTrackFromAlbum(ctx context.Context, albumID, trackID int32) error
}

type CreateAlbumParams struct {
	ArtistID    int32
	Name        string
	ReleaseYear *int32
	ReleaseDate *time.Time
}

type Album struct {
	ID          int32
	Name        string
	ArtistID    int32
	ReleaseYear *int32
	ReleaseDate *time.Time
	TrackIDs    []int32
}

type PostgresAlbumRepository struct {
	queries *db.Queries
}

func NewAlbumRepository(pool *pgxpool.Pool) *PostgresAlbumRepository {
	return &PostgresAlbumRepository{
		queries: db.New(pool),
	}
}

func (repo *PostgresAlbumRepository) CreateAlbum(
	ctx context.Context, params CreateAlbumParams,
) (Album, error) {
	album, err := repo.queries.CreateAlbum(ctx, db.CreateAlbumParams{
		Name:            params.Name,
		ArtistID:        params.ArtistID,
		ReleaseYear:     pgIntFromInt32Ptr(params.ReleaseYear),
		ReleaseFullDate: pgDateFromTimePtr(params.ReleaseDate),
	})
	if err != nil {
		return Album{}, wrapDBError(err)
	}
	return albumFromDBAlbum(album), nil
}

func (repo *PostgresAlbumRepository) GetAlbum(ctx context.Context, albumID int32) (Album, error) {
	album, err := repo.queries.GetAlbum(ctx, albumID)
	if err != nil {
		return Album{}, wrapDBError(err)
	}
	return albumFromGetAlbumRow(album), nil
}

func (repo *PostgresAlbumRepository) GetAlbumInfo(
	ctx context.Context, albumID int32,
) (Album, error) {
	album, err := repo.GetAlbum(ctx, albumID)
	if err != nil {
		return Album{}, err
	}

	tracks, err := repo.queries.GetAlbumWithTracks(ctx, albumID)
	if err != nil {
		return Album{}, wrapDBError(err)
	}
	album.TrackIDs = make([]int32, len(tracks))
	for i, track := range tracks {
		album.TrackIDs[i] = track.TrackID
	}
	return album, nil
}

func (repo *PostgresAlbumRepository) DeleteAlbum(ctx context.Context, albumID int32) error {
	err := repo.queries.DeleteAlbum(ctx, albumID)
	return wrapDBError(err)
}

func (repo *PostgresAlbumRepository) DeleteTrackFromAlbum(
	ctx context.Context, albumID, trackID int32,
) error {
	err := repo.queries.DeleteTrackFromAlbumRelation(
		ctx,
		db.DeleteTrackFromAlbumRelationParams{
			AlbumID: albumID,
			TrackID: trackID,
		},
	)
	return wrapDBError(err)
}

func albumFromDBAlbum(album db.Album) Album {
	return Album{
		ID:          album.ID,
		Name:        album.Name,
		ArtistID:    album.ArtistID,
		ReleaseYear: int32PtrFromPGInt(album.ReleaseYear),
		ReleaseDate: timePtrFromPGDate(album.ReleaseFullDate),
	}
}

func albumFromGetAlbumRow(album db.GetAlbumRow) Album {
	return Album{
		ID:          album.ID,
		Name:        album.Name,
		ReleaseYear: int32PtrFromPGInt(album.ReleaseYear),
		ReleaseDate: timePtrFromPGDate(album.ReleaseFullDate),
	}
}

func pgIntFromInt32Ptr(value *int32) pgtype.Int4 {
	if value == nil {
		return pgtype.Int4{Valid: false}
	}
	return pgtype.Int4{Int32: *value, Valid: true}
}

func pgDateFromTimePtr(value *time.Time) pgtype.Date {
	if value == nil {
		return pgtype.Date{Valid: false}
	}
	return pgtype.Date{Time: *value, Valid: true}
}

func int32PtrFromPGInt(value pgtype.Int4) *int32 {
	if !value.Valid {
		return nil
	}
	result := value.Int32
	return &result
}

func timePtrFromPGDate(value pgtype.Date) *time.Time {
	if !value.Valid {
		return nil
	}
	result := value.Time
	return &result
}
