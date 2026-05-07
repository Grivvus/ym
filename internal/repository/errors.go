package repository

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var ErrNotFound = errors.New("repository: not found")
var ErrAlreadyExists = errors.New("repository: already exists")
var ErrUnknownDBError = errors.New("repository: unknown database error")

func wrapDBError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrNotFound) ||
		errors.Is(err, ErrAlreadyExists) ||
		errors.Is(err, ErrUnknownDBError) {
		return err
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%w caused by: %w", ErrNotFound, err)
	}
	if isUniqueViolation(err) {
		return fmt.Errorf("%w caused by: %w", ErrAlreadyExists, err)
	}
	return fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
}

func isUniqueViolation(err error) bool {
	pgErr, ok := errors.AsType[*pgconn.PgError](err)
	return ok && pgErr.Code == "23505"
}
