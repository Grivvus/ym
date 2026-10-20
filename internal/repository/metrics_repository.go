package repository

import (
	"context"

	"github.com/Grivvus/ym/internal/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MetricsRepository interface {
	TotalUniqueTracks(context.Context) (uint, error)
	TotalUsersRegistered(context.Context) (uint, error)
	CurrentTranscodingQueueSize(context.Context) (uint, error)
	CurrentTranscodingQueueFailed(context.Context) (uint, error)
}

func NewMetricsRepository(pool *pgxpool.Pool) MetricsRepository {
	return &postgresMetricsRepository{
		pool:    pool,
		queries: db.New(pool),
	}
}

type postgresMetricsRepository struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

func (r *postgresMetricsRepository) TotalUniqueTracks(
	ctx context.Context,
) (uint, error) {
	cnt, err := r.queries.GetTracksCount(ctx)
	return uint(cnt), wrapDBError(err)
}

func (r *postgresMetricsRepository) TotalUsersRegistered(
	ctx context.Context,
) (uint, error) {
	cnt, err := r.queries.GetUserCount(ctx)
	return uint(cnt), wrapDBError(err)
}

func (r *postgresMetricsRepository) CurrentTranscodingQueueSize(
	ctx context.Context,
) (uint, error) {
	cnt, err := r.queries.TranscodingQueueSize(ctx)
	return uint(cnt), wrapDBError(err)
}

func (r *postgresMetricsRepository) CurrentTranscodingQueueFailed(
	ctx context.Context,
) (uint, error) {
	cnt, err := r.queries.TranscodingQueueFailed(ctx)
	return uint(cnt), wrapDBError(err)
}
