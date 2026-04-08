package repository

import (
	"context"
	"fmt"

	"github.com/Grivvus/ym/internal/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func withTx[T any](
	ctx context.Context, pool *pgxpool.Pool, queries *db.Queries, fn func(queries *db.Queries) (T, error),
) (T, error) {
	var zero T

	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	qtx := queries.WithTx(tx)
	if err != nil {
		return zero, fmt.Errorf(
			"error on executing query: %w", err,
		)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	value, err := fn(qtx)
	if err != nil {
		return zero, err
	}

	if err := tx.Commit(ctx); err != nil {
		return zero, fmt.Errorf(
			"error on commit: %w", err,
		)
	}

	return value, nil
}
