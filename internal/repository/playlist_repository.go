package repository

import (
	"context"

	"github.com/Grivvus/ym/internal/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PlaylistRepository struct {
	pool *pgxpool.Pool
	q    *db.Queries
}

func NewPlaylistRepository(pool *pgxpool.Pool) *PlaylistRepository {
	return &PlaylistRepository{
		pool: pool,
		q:    db.New(pool),
	}
}

func (repo *PlaylistRepository) SharePlaylistWithMany(
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

func (repo *PlaylistRepository) SharePlaylist(
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

func (repo *PlaylistRepository) RevokePlaylistAccess(
	ctx context.Context, playlistID, userID int32,
) error {
	_, err := repo.q.RevokePlaylistAccess(ctx, db.RevokePlaylistAccessParams{
		PlaylistID:     playlistID,
		SharedWithUser: userID,
	})
	return wrapDBError(err)
}
