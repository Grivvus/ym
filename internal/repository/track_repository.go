package repository

import (
	"context"

	"github.com/Grivvus/ym/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TrackRepository interface {
	CreateTrackWithAlbum(ctx context.Context, params CreateTrackParams) (Track, error)
	AddToTranscodingQueue(ctx context.Context, trackID int32, originalFileName string) error
	GetTrack(ctx context.Context, trackID int32) (Track, error)
	GetAllTracks(ctx context.Context) ([]Track, error)
	GetAlbumIDByTrackID(ctx context.Context, trackID int32) (int32, error)
}

type CreateTrackParams struct {
	Name                string
	ArtistID            int32
	UploadBy            *int32
	IsGloballyAvailable bool
	AlbumID             *int32
	NewAlbum            *CreateTrackAlbumParams
}

type CreateTrackAlbumParams struct {
	Name     string
	ArtistID int32
}

type Track struct {
	ID                  int32
	Name                string
	ArtistID            int32
	ArtistName          string
	AlbumID             int32
	DurationMs          int32
	FastPresetName      *string
	StandardPresetName  *string
	HighPresetName      *string
	LosslessPresetName  *string
	IsGloballyAvailable bool
	UploadBy            *int32
}

type PostgresTrackRepository struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

func NewTrackRepository(pool *pgxpool.Pool) *PostgresTrackRepository {
	return &PostgresTrackRepository{
		pool:    pool,
		queries: db.New(pool),
	}
}

func (repo *PostgresTrackRepository) CreateTrackWithAlbum(
	ctx context.Context, params CreateTrackParams,
) (Track, error) {
	track, err := withTx(ctx, repo.pool, repo.queries, func(q *db.Queries) (Track, error) {
		createdTrack, err := q.CreateTrack(ctx, db.CreateTrackParams{
			Name:                params.Name,
			ArtistID:            params.ArtistID,
			UploadByUser:        pgIntFromInt32Ptr(params.UploadBy),
			IsGloballyAvailable: params.IsGloballyAvailable,
		})
		if err != nil {
			return Track{}, err
		}

		albumID := int32(0)
		if params.NewAlbum != nil {
			album, err := q.CreateAlbum(ctx, db.CreateAlbumParams{
				Name:     params.NewAlbum.Name,
				ArtistID: params.NewAlbum.ArtistID,
			})
			if err != nil {
				return Track{}, err
			}
			albumID = album.ID
		} else if params.AlbumID != nil {
			albumID = *params.AlbumID
		}

		err = q.AddTrackToAlbum(ctx, db.AddTrackToAlbumParams{
			TrackID: createdTrack.ID,
			AlbumID: albumID,
		})
		if err != nil {
			return Track{}, err
		}

		track := trackFromDBTrack(createdTrack)
		track.AlbumID = albumID
		return track, nil
	})
	if err != nil {
		return Track{}, wrapDBError(err)
	}
	return track, nil
}

func (repo *PostgresTrackRepository) AddToTranscodingQueue(
	ctx context.Context, trackID int32, originalFileName string,
) error {
	_, err := repo.queries.AddToTranscodingQueue(ctx, db.AddToTranscodingQueueParams{
		TrackID:               trackID,
		TrackOriginalFileName: originalFileName,
	})
	return wrapDBError(err)
}

func (repo *PostgresTrackRepository) GetTrack(ctx context.Context, trackID int32) (Track, error) {
	track, err := repo.queries.GetTrack(ctx, trackID)
	if err != nil {
		return Track{}, wrapDBError(err)
	}
	return trackFromGetTrackRow(track), nil
}

func (repo *PostgresTrackRepository) GetAllTracks(ctx context.Context) ([]Track, error) {
	tracks, err := repo.queries.GetAllTracks(ctx)
	if err != nil {
		return nil, wrapDBError(err)
	}
	result := make([]Track, len(tracks))
	for i, track := range tracks {
		result[i] = trackFromGetAllTracksRow(track)
	}
	return result, nil
}

func (repo *PostgresTrackRepository) GetAlbumIDByTrackID(
	ctx context.Context, trackID int32,
) (int32, error) {
	albumID, err := repo.queries.GetAlbumByTrackID(ctx, trackID)
	if err != nil {
		return 0, wrapDBError(err)
	}
	return albumID, nil
}

func trackFromDBTrack(track db.Track) Track {
	return Track{
		ID:                  track.ID,
		Name:                track.Name,
		ArtistID:            track.ArtistID,
		DurationMs:          int32FromPGInt(track.DurationMs),
		FastPresetName:      stringPtrFromPGText(track.FastPresetFname),
		StandardPresetName:  stringPtrFromPGText(track.StandardPresetFname),
		HighPresetName:      stringPtrFromPGText(track.HighPresetFname),
		LosslessPresetName:  stringPtrFromPGText(track.LosslessPresetFname),
		IsGloballyAvailable: track.IsGloballyAvailable,
		UploadBy:            int32PtrFromPGInt(track.UploadByUser),
	}
}

func trackFromGetTrackRow(track db.GetTrackRow) Track {
	return Track{
		ID:                  track.ID,
		Name:                track.Name,
		ArtistID:            track.ArtistID,
		ArtistName:          track.ArtistName,
		AlbumID:             track.AlbumID,
		DurationMs:          int32FromPGInt(track.DurationMs),
		FastPresetName:      stringPtrFromPGText(track.FastPresetFname),
		StandardPresetName:  stringPtrFromPGText(track.StandardPresetFname),
		HighPresetName:      stringPtrFromPGText(track.HighPresetFname),
		LosslessPresetName:  stringPtrFromPGText(track.LosslessPresetFname),
		IsGloballyAvailable: track.IsGloballyAvailable,
		UploadBy:            int32PtrFromPGInt(track.UploadByUser),
	}
}

func trackFromGetAllTracksRow(track db.GetAllTracksRow) Track {
	return Track{
		ID:                 track.ID,
		Name:               track.Name,
		ArtistID:           track.ArtistID,
		DurationMs:         int32FromPGInt(track.DurationMs),
		FastPresetName:     stringPtrFromPGText(track.FastPresetFname),
		StandardPresetName: stringPtrFromPGText(track.StandardPresetFname),
		HighPresetName:     stringPtrFromPGText(track.HighPresetFname),
		LosslessPresetName: stringPtrFromPGText(track.LosslessPresetFname),
	}
}

func int32FromPGInt(value pgtype.Int4) int32 {
	if !value.Valid {
		return 0
	}
	return value.Int32
}
