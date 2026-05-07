package repository

import (
	"context"

	"github.com/Grivvus/ym/internal/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PlaylistRepository interface {
	SharePlaylistWithMany(
		ctx context.Context, playlistID int32,
		hasWritePermission bool, users []int32,
	) error
	SharePlaylist(
		ctx context.Context, playlistID int32,
		hasWritePermission bool, userID int32,
	) error
	RevokePlaylistAccess(ctx context.Context, playlistID, userID int32) error
}

type PostgresPlaylistRepository struct {
	pool *pgxpool.Pool
	q    *db.Queries
}

func NewPlaylistRepository(pool *pgxpool.Pool) *PostgresPlaylistRepository {
	return &PostgresPlaylistRepository{
		pool: pool,
		q:    db.New(pool),
	}
}

func (repo *PostgresPlaylistRepository) SharePlaylistWithMany(
	ctx context.Context, playlistID int32,
	hasWritePermission bool, users []int32,
) error {
	_, err := withTx(ctx, repo.pool, repo.q, func(q *db.Queries) (int, error) {
		for _, userID := range users {
			err := q.SharePlaylistWith(ctx, db.SharePlaylistWithParams{
				SharedWithUser:     userID,
				PlaylistID:         playlistID,
				HasWritePermission: hasWritePermission,
			})
			if err != nil {
				return 0, err
			}
		}
		return 0, nil
	})
	return wrapDBError(err)
}

func (repo *PostgresPlaylistRepository) SharePlaylist(
	ctx context.Context, playlistID int32,
	hasWritePermission bool, userID int32,
) error {
	err := repo.q.SharePlaylistWith(ctx, db.SharePlaylistWithParams{
		SharedWithUser:     userID,
		PlaylistID:         playlistID,
		HasWritePermission: hasWritePermission,
	})
	return wrapDBError(err)
}

func (repo *PostgresPlaylistRepository) RevokePlaylistAccess(
	ctx context.Context, playlistID, userID int32,
) error {
	_, err := repo.q.RevokePlaylistAccess(ctx, db.RevokePlaylistAccessParams{
		PlaylistID:     playlistID,
		SharedWithUser: userID,
	})
	return wrapDBError(err)
}
