package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"

	"github.com/Grivvus/ym/internal/api"
	"github.com/Grivvus/ym/internal/audio"
	"github.com/Grivvus/ym/internal/db"
	"github.com/Grivvus/ym/internal/storage"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

type TrackUploadParams struct {
	ArtistID            int32
	AlbumID             *int32
	UploadBy            *int32
	IsGloballyAvailable bool
	IsSingle            bool
	Name                string
}

var ErrPresetCantBeSelected = errors.New("preset can't be selected for this track")

type StreamMeta struct {
	ContentLength uint
	ContentType   string
}

type TrackStream struct {
	Name        string
	ContentType string
	Reader      io.ReadSeekCloser
}

type TrackService struct {
	queries                *db.Queries
	st                     storage.Storage
	logger                 *slog.Logger
	transcodingQueueSignal chan<- struct{}
}

func NewTrackService(
	q *db.Queries, st storage.Storage, logger *slog.Logger,
	transcodingQueueSignal chan<- struct{},
) TrackService {
	return TrackService{
		queries:                q,
		st:                     st,
		logger:                 logger,
		transcodingQueueSignal: transcodingQueueSignal,
	}
}

func (s *TrackService) UploadTrack(
	ctx context.Context, params TrackUploadParams,
	trackFileHeader *multipart.FileHeader,
) (api.TrackUploadSuccessResponse, error) {
	var ret api.TrackUploadSuccessResponse
	var userID int32
	if params.UploadBy != nil {
		userID = *params.UploadBy
	}
	track, err := s.queries.CreateTrack(ctx, db.CreateTrackParams{
		Name:                params.Name,
		ArtistID:            params.ArtistID,
		IsGloballyAvailable: params.IsGloballyAvailable,
		UploadByUser:        pgtype.Int4{Valid: params.UploadBy != nil, Int32: userID},
	})
	if err != nil {
		return ret, fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}

	ret.TrackId = track.ID

	var albumID int32
	if params.IsSingle {
		single, err := s.queries.CreateAlbum(ctx, db.CreateAlbumParams{
			Name:     track.Name,
			ArtistID: track.ArtistID,
		})
		if err != nil {
			if e, ok := errors.AsType[*pgconn.PgError](err); ok && e.Code == "23505" {
				return ret, fmt.Errorf(
					"%w: album with this name already exists",
					NewErrAlreadyExists("album", track.Name),
				)
			}
			return ret, fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
		}
		albumID = single.ID
	} else {
		if params.AlbumID == nil {
			return ret, fmt.Errorf(
				"%w: albumID is required if track is not a single", ErrBadParams,
			)
		}
		albumID = *params.AlbumID
	}

	err = s.queries.AddTrackToAlbum(ctx, db.AddTrackToAlbumParams{
		TrackID: track.ID,
		AlbumID: albumID,
	})
	if err != nil {
		if e, ok := errors.AsType[*pgconn.PgError](err); ok && e.Code == "23505" {
			return ret, fmt.Errorf(
				"%w: album already has this track",
				NewErrAlreadyExists("track", track.ID),
			)
		}
		return ret, fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}

	rc, err := trackFileHeader.Open()
	if err != nil {
		return ret, fmt.Errorf("assertion, should never happen: %w", err)
	}
	defer func() { _ = rc.Close() }()

	tmpFname := originalTrackStorageKey(track.ID)
	err = s.st.PutTrack(ctx, tmpFname, rc, trackFileHeader.Size)
	if err != nil {
		s.logger.Warn("error mapping should be more precise")
		return ret, fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}

	_, err = s.queries.AddToTranscodingQueue(
		ctx, db.AddToTranscodingQueueParams{
			TrackID:               track.ID,
			TrackOriginalFileName: tmpFname,
		})
	if err != nil {
		return ret, fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}

	s.transcodingQueueSignal <- struct{}{}

	return ret, nil
}

func (s *TrackService) DeleteTrack(ctx context.Context, trackID int32) error {
	metadata, err := s.GetMeta(ctx, trackID)
	if err != nil {
		return err
	}
	errs := make([]error, 0, 5)
	if metadata.TrackFastPreset != nil {
		errs = append(errs, s.st.RemoveTrack(ctx, *metadata.TrackFastPreset))
	}
	if metadata.TrackStandardPreset != nil {
		errs = append(errs, s.st.RemoveTrack(ctx, *metadata.TrackStandardPreset))
	}
	if metadata.TrackHighPreset != nil {
		errs = append(errs, s.st.RemoveTrack(ctx, *metadata.TrackHighPreset))
	}
	if metadata.TrackLosslessPreset != nil {
		errs = append(errs, s.st.RemoveTrack(ctx, *metadata.TrackLosslessPreset))
	}
	errs = append(errs, s.st.RemoveTrack(ctx, metadata.Name))
	for _, err := range errs {
		if err != nil {
			return fmt.Errorf("can't remove track file: %w", err)
		}
	}
	err = s.st.RemoveTrack(ctx, originalTrackStorageKey(metadata.TrackId))
	if err != nil {
		return fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}
	return nil
}

func (s *TrackService) GetMeta(
	ctx context.Context, trackID int32,
) (api.TrackMetadata, error) {
	trackInfo, err := s.queries.GetTrack(ctx, trackID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return api.TrackMetadata{}, NewErrNotFound("track", trackID)
		}
		return api.TrackMetadata{}, fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}

	var (
		fastPreset     *string
		standardPreset *string
		highPreset     *string
		losslessPreset *string
	)
	if trackInfo.FastPresetFname.Valid {
		fastPreset = &trackInfo.FastPresetFname.String
	}
	if trackInfo.StandardPresetFname.Valid {
		standardPreset = &trackInfo.StandardPresetFname.String
	}
	if trackInfo.HighPresetFname.Valid {
		highPreset = &trackInfo.HighPresetFname.String
	}
	if trackInfo.LosslessPresetFname.Valid {
		losslessPreset = &trackInfo.LosslessPresetFname.String
	}
	s.logger.Warn("remove hardcoded values")
	trackCoverURL := fmt.Sprintf("0.0.0.0:8000/albums/%d/cover", trackInfo.AlbumID)

	return api.TrackMetadata{
		TrackId:             trackInfo.ID,
		ArtistId:            trackInfo.ArtistID,
		AlbumId:             trackInfo.AlbumID,
		CoverUrl:            &trackCoverURL,
		Name:                trackInfo.Name,
		DurationMs:          trackInfo.DurationMs.Int32,
		TrackFastPreset:     fastPreset,
		TrackStandardPreset: standardPreset,
		TrackHighPreset:     highPreset,
		TrackLosslessPreset: losslessPreset,
	}, nil
}

func (s *TrackService) GetUserTracks(ctx context.Context, userID int32) ([]api.TrackMetadata, error) {
	tracks, err := s.queries.GetUserTracks(ctx, pgtype.Int4{Int32: userID, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}
	ret := make([]api.TrackMetadata, len(tracks))
	for i, track := range tracks {
		ret[i] = api.TrackMetadata{
			ArtistId:            track.ArtistID,
			Name:                track.Name,
			DurationMs:          track.DurationMs.Int32,
			TrackId:             track.ID,
			TrackFastPreset:     &track.FastPresetFname.String,
			TrackStandardPreset: &track.StandardPresetFname.String,
			TrackHighPreset:     &track.HighPresetFname.String,
			TrackLosslessPreset: nil,
		}
	}
	return ret, nil
}

func (s *TrackService) GetStream(
	ctx context.Context, trackID int32, trackQuality string,
) (TrackStream, error) {
	preset, err := audio.PresetFromString(trackQuality)
	if err != nil {
		return TrackStream{}, fmt.Errorf("%w: invalid name of trackQuality: %v", ErrBadParams, trackQuality)
	}
	track, trackExist, err := s.trackExists(ctx, trackID)
	if err != nil {
		return TrackStream{}, err
	}
	if !trackExist {
		return TrackStream{}, NewErrNotFound("track", trackID)
	}
	trackKey, err := findClosestExistingTrackKey(track, preset)
	if err != nil {
		return TrackStream{}, err
	}

	stream, err := s.st.GetTrack(ctx, trackKey)
	if err != nil {
		return TrackStream{}, fmt.Errorf("can't get track stream: %w", err)
	}

	_, ctype, err := s.st.GetTrackInfo(ctx, trackKey)
	if err != nil {
		_ = stream.Close()
		return TrackStream{}, fmt.Errorf("can't fetch track info: %w", err)
	}

	return TrackStream{
		Name:        trackKey,
		ContentType: ctype,
		Reader:      stream,
	}, nil
}

func (s *TrackService) trackExists(
	ctx context.Context, trackID int32,
) (db.GetTrackRow, bool, error) {
	track, err := s.queries.GetTrack(ctx, trackID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return track, false, nil
		}
		return track, false, fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}
	return track, true, nil
}

func findClosestExistingTrackKey(
	track db.GetTrackRow, chosenPreset audio.Preset,
) (string, error) {
	switch chosenPreset {
	// lossless->orig
	case audio.PresetLossless:
		if track.LosslessPresetFname.Valid {
			return track.LosslessPresetFname.String, nil
		}
		return originalTrackStorageKey(track.ID), nil
	// H->S->F->orig
	case audio.PresetHigh:
		if track.HighPresetFname.Valid {
			return track.HighPresetFname.String, nil
		} else if track.StandardPresetFname.Valid {
			return track.StandardPresetFname.String, nil
		} else if track.FastPresetFname.Valid {
			return track.FastPresetFname.String, nil
		}
		return originalTrackStorageKey(track.ID), nil
	// S->F->H->orig
	case audio.PresetStandard:
		if track.StandardPresetFname.Valid {
			return track.StandardPresetFname.String, nil
		} else if track.FastPresetFname.Valid {
			return track.FastPresetFname.String, nil
		} else if track.HighPresetFname.Valid {
			return track.HighPresetFname.String, nil
		}
		return originalTrackStorageKey(track.ID), nil
	// F->S->H->orig
	case audio.PresetFast:
		if track.FastPresetFname.Valid {
			return track.FastPresetFname.String, nil
		} else if track.StandardPresetFname.Valid {
			return track.StandardPresetFname.String, nil
		} else if track.HighPresetFname.Valid {
			return track.HighPresetFname.String, nil
		}
		return originalTrackStorageKey(track.ID), nil
	default:
		return "", ErrPresetCantBeSelected
	}
}
