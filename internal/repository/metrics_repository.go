package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type MetricsRepository interface {
	TotalUniqueTracks(ctx context.Context) (uint, error)
	TotalUsersRegistered(ctx context.Context) (uint, error)
	CurrentTranscodingQueueSize(ctx context.Context) (uint, error)
}

func NewMetricsRepository(pool *pgxpool.Pool) MetricsRepository {}
