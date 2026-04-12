package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/Grivvus/ym/internal/api"
	"github.com/Grivvus/ym/internal/db"
	"github.com/Grivvus/ym/internal/storage"
	"github.com/jackc/pgx/v5"
)

type BackupSettings struct {
	IncludeImages           bool
	IncludeTranscodedTracks bool
}

type BackupService struct {
	logger  *slog.Logger
	queries *db.Queries
	storage storage.Storage
}

func NewBackupService(
	logger *slog.Logger, queries *db.Queries, storage storage.Storage,
) BackupService {
	return BackupService{logger: logger}
}

func (service BackupService) MakeBackup(
	ctx context.Context, settings BackupSettings,
) (backup io.ReadCloser, clen uint, err error) {
	panic("not implemented")
}

func (service BackupService) StartRestoreOperation(
	ctx context.Context, backup io.Reader,
) (restoreID string, err error) {
	// make only 1 running backup possible
	panic("not implemented")
}

func (service BackupService) CheckRestoreOperation(
	ctx context.Context, restoreID string,
) (response api.RestoreStatusResponse, err error) {
	status, err := service.queries.GetRestoreStatus(ctx, restoreID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return response, NewErrNotFound("restore_status", restoreID)
		}
		return response, fmt.Errorf("%w cased by: %w", ErrUnknownDBError, err)
	}
	response.RestoreId = status.ID

	switch status.Status {
	case db.StatusPending:
		response.Status = api.Pending
	case db.StatusStarted:
		response.Status = api.Started
	case db.StatusFinished:
		response.Status = api.Finished
	case db.StatusError:
		response.Status = api.Error
		response.Error = &status.Error.String
	default:
		return response, fmt.Errorf(
			"%w: invalid status - %v", ErrBadParams, status.Status,
		)
	}

	return response, nil
}
