package repository

import (
	"context"
	"errors"

	"github.com/Grivvus/ym/internal/audio"
	"github.com/Grivvus/ym/internal/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const pageSize = 10

type TranscodingQueueRepository struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

func NewTranscodingQueueRepository(
	pool *pgxpool.Pool, queries *db.Queries,
) *TranscodingQueueRepository {
	return &TranscodingQueueRepository{
		pool:    pool,
		queries: queries,
	}
}

func (repo *TranscodingQueueRepository) GetTranscodingQueue(
	ctx context.Context,
) (<-chan db.GetTranscodingQueueRow, <-chan error) {
	queue := make(chan db.GetTranscodingQueueRow)
	errc := make(chan error, 1)
	go func() {
		defer close(queue)
		defer close(errc)
		lastElementID := int32(0)
		for {
			rows, err := repo.queries.GetTranscodingQueue(ctx, db.GetTranscodingQueueParams{
				ID:    lastElementID,
				Limit: pageSize,
			})
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				errc <- err
				return
			}

			for _, row := range rows {
				queue <- row
				lastElementID = row.ID
			}

			if len(rows) < pageSize {
				return
			}
		}
	}()
	return queue, errc
}

func (repo *TranscodingQueueRepository) RemoveFromQueueAndUpdateTrack(
	ctx context.Context, queueID int32, trackDuration int32,
	presetsToName map[audio.Preset]string,
) error {
	_, err := withTx[int32](
		ctx, repo.pool, repo.queries, func(q *db.Queries) (int32, error) {
			queue, err := q.DeleteFromTranscodingQueue(ctx, queueID)
			if err != nil {
				return 0, err
			}

			_, err = q.AddPostTranscodingInfo(ctx, db.AddPostTranscodingInfoParams{
				ID:         queue.TrackID,
				DurationMs: pgtype.Int4{Int32: trackDuration, Valid: true},
				FastPresetFname: pgtype.Text{
					String: presetsToName[audio.PresetFast],
					Valid:  presetsToName[audio.PresetFast] != "",
				},
				StandardPresetFname: pgtype.Text{
					String: presetsToName[audio.PresetStandard],
					Valid:  presetsToName[audio.PresetStandard] != "",
				},
				HighPresetFname: pgtype.Text{
					String: presetsToName[audio.PresetHigh],
					Valid:  presetsToName[audio.PresetHigh] != "",
				},
				LosslessPresetFname: pgtype.Text{
					String: presetsToName[audio.PresetLossless],
					Valid:  presetsToName[audio.PresetLossless] != "",
				},
			})
			return queue.ID, err
		})
	return err
}

func (repo *TranscodingQueueRepository) OnFailedTranscoding(
	ctx context.Context, queueID int32, errorMsg error,
) error {
	_, err := withTx(ctx, repo.pool, repo.queries, func(q *db.Queries) (int32, error) {
		queue, err := q.DeleteFromTranscodingQueue(ctx, queueID)
		if err != nil {
			return 0, err
		}
		_, err = q.AddToTranscodingQueue(ctx, db.AddToTranscodingQueueParams{
			TrackID:               queue.TrackID,
			TrackOriginalFileName: queue.TrackOriginalFileName,
			WasFailed:             true,
			ErrorMsg:              pgtype.Text{String: errorMsg.Error(), Valid: true},
		})
		return 0, err
	})
	return err
}
