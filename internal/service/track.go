package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/Grivvus/ym/internal/api"
	"github.com/Grivvus/ym/internal/db"
	"github.com/Grivvus/ym/internal/storage"
	"github.com/Grivvus/ym/internal/transcoder"
	"github.com/Grivvus/ym/internal/utils"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var ErrPresetCantBeSelected error = errors.New("Preset can't be selected for this track")

type StreamMeta struct {
	ContentLength uint
	ContentType   string
}

type TrackService struct {
	queries *db.Queries
	st      storage.Storage
}

func NewTrackService(q *db.Queries, st storage.Storage) TrackService {
	return TrackService{
		queries: q,
		st:      st,
	}
}

func (s *TrackService) UploadTrack(
	ctx context.Context, data api.TrackUploadRequest,
) (api.TrackUploadSuccessResponse, error) {
	var ret api.TrackUploadSuccessResponse
	track, err := s.queries.CreateTrack(ctx, db.CreateTrackParams{
		Name:     data.Name,
		ArtistID: int32(data.ArtistId),
		Duration: pgtype.Int4{Valid: false},
	})
	if err != nil {
		return ret, fmt.Errorf("can't create new record in db: %w", err)
	}

	ret.TrackId = int(track.ID)

	err = s.queries.AddTrackToAlbum(ctx, db.AddTrackToAlbumParams{
		TrackID: track.ID,
		AlbumID: int32(data.AlbumId),
	})
	if err != nil {
		return ret, fmt.Errorf("can't add this track to album: %w", err)
	}

	r, err := data.File.Reader()
	if err != nil {
		panic(err)
	}
	tmpFname := s.tmpFileName(int(track.ID))
	err = utils.SaveAsFile(r, tmpFname)
	if err != nil {
		return ret, fmt.Errorf("can't create tmp file: %w", err)
	}

	presetsFiles, err := transcoder.TranscodeConcurrent(ctx, tmpFname)
	if err != nil {
		return ret, fmt.Errorf("error in transcoding process: %w", err)
	}
	for _, v := range presetsFiles {
		f, err := os.Open(v)
		defer func() { _ = f.Close() }()

		if err != nil {
			return ret, fmt.Errorf("can't find file with transcoded samples: %w", err)
		}
		err = s.st.PutTrack(ctx, v, f)
		if err != nil {
			return ret, fmt.Errorf("can't save transcoded sample: %w", err)
		}
	}
	_, err = s.queries.AddTrackPresets(
		ctx, db.AddTrackPresetsParams{
			ID: int32(ret.TrackId),
			FastPresetFname: pgtype.Text{
				String: presetsFiles[transcoder.PresetFast],
				Valid:  true,
			},
			StandardPresetFname: pgtype.Text{
				String: presetsFiles[transcoder.PresetStandard],
				Valid:  true,
			},
			HighPresetFname: pgtype.Text{
				String: presetsFiles[transcoder.PresetHigh],
				Valid:  true,
			},
			LosslessPresetFname: pgtype.Text{
				String: presetsFiles[transcoder.PresetLossless],
				Valid:  true,
			},
		},
	)
	if err != nil {
		return ret, fmt.Errorf("can't add track-presets to a record: %w", err)
	}
	return ret, nil

}

func (s *TrackService) DeleteTrack(ctx context.Context, trackID int) error {
	metadata, err := s.GetMeta(ctx, trackID)
	if err != nil {
		return fmt.Errorf("can't fetch track metadata: %w", err)
	}
	errs := make([]error, 0, 4)
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
	for _, err := range errs {
		if err != nil {
			return fmt.Errorf("can't remove track file: %w", err)
		}
	}
	err = s.queries.DeleteTrack(ctx, int32(trackID))
	if err != nil {
		return fmt.Errorf("can't remove track record: %w", err)
	}
	return nil
}

func (s *TrackService) GetMeta(
	ctx context.Context, trackID int,
) (api.TrackMetadata, error) {
	trackInfo, err := s.queries.GetTrack(ctx, int32(trackID))
	if err != nil {
		return api.TrackMetadata{}, fmt.Errorf("can't fetch info about track: %w", err)
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

	return api.TrackMetadata{
		TrackId:             int(trackInfo.ID),
		ArtistId:            int(trackInfo.ArtistID),
		CoverUrl:            nil,
		Name:                trackInfo.Name,
		TrackFastPreset:     fastPreset,
		TrackStandardPreset: standardPreset,
		TrackHighPreset:     highPreset,
		TrackLosslessPreset: losslessPreset,
	}, nil
}

func (s *TrackService) GetStreamMeta(
	ctx context.Context, trackId int, trackQuality string,
) (StreamMeta, error) {
	preset, err := transcoder.PresetFromString(trackQuality)
	if err != nil {
		return StreamMeta{}, fmt.Errorf("Invalid name of trackQuality: %v", trackQuality)
	}
	fullTrackName := transcoder.TranscodedName(s.tmpFileName(trackId), preset)
	clen, ctype, err := s.st.GetTrackInfo(ctx, fullTrackName)
	if err != nil {
		return StreamMeta{}, fmt.Errorf("can't fetch track info: %w", err)
	}
	return StreamMeta{
		ContentLength: clen,
		ContentType:   ctype,
	}, nil
}

func (s *TrackService) GetStream(
	ctx context.Context, trackID int, trackQuality string,
) (io.ReadCloser, error) {
	preset, err := transcoder.PresetFromString(trackQuality)
	if err != nil {
		return nil, fmt.Errorf("Invalid name of trackQuality: %v", trackQuality)
	}
	track, trackExist, err := s.trackExists(ctx, trackID)
	if err != nil {
		return nil, err
	}
	if !trackExist {
		return nil, fmt.Errorf("track doesn't exist")
	}
	preset, err = s.findClosestExistingPreset(track, preset)
	if err != nil {
		return nil, err
	}

	fullTrackName := transcoder.TranscodedName(s.tmpFileName(trackID), preset)
	return s.st.GetTrack(ctx, fullTrackName)
}

func (s *TrackService) trackExists(
	ctx context.Context, trackID int,
) (db.Track, bool, error) {
	track, err := s.queries.GetTrack(ctx, int32(trackID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return track, false, nil
		} else {
			return track, false, fmt.Errorf("unexpected db error: %w", err)
		}
	}
	return track, true, nil
}

func (s *TrackService) findClosestExistingPreset(
	track db.Track, chosenPreset transcoder.Preset,
) (transcoder.Preset, error) {
	switch chosenPreset {
	// if we're looking for Lossless:
	// find exact lossless
	case transcoder.PresetLossless:
		if track.LosslessPresetFname.Valid {
			return transcoder.PresetLossless, nil
		} else {
			return transcoder.Preset(0), ErrPresetCantBeSelected
		}
		// if we're looking for high
		// searching H->S->F
	case transcoder.PresetHigh:
		if track.HighPresetFname.Valid {
			return transcoder.PresetHigh, nil
		} else if track.StandardPresetFname.Valid {
			return transcoder.PresetStandard, nil
		} else if track.FastPresetFname.Valid {
			return transcoder.PresetFast, nil
		} else {
			return transcoder.Preset(0), ErrPresetCantBeSelected
		}
		// if we're looking for standard
		// searching S->F->H
	case transcoder.PresetStandard:
		if track.StandardPresetFname.Valid {
			return transcoder.PresetStandard, nil
		} else if track.FastPresetFname.Valid {
			return transcoder.PresetFast, nil
		} else if track.HighPresetFname.Valid {
			return transcoder.PresetHigh, nil
		} else {
			return transcoder.Preset(0), ErrPresetCantBeSelected
		}
		// if we're looking for fast
		// searching F->S->H
	case transcoder.PresetFast:
		if track.FastPresetFname.Valid {
			return transcoder.PresetFast, nil
		} else if track.StandardPresetFname.Valid {
			return transcoder.PresetStandard, nil
		} else if track.HighPresetFname.Valid {
			return transcoder.PresetHigh, nil
		} else {
			return transcoder.Preset(0), ErrPresetCantBeSelected
		}
	default:
		return transcoder.Preset(0), ErrPresetCantBeSelected
	}
}

func (s *TrackService) tmpFileName(trackID int) string {
	return "track" + strconv.Itoa(trackID)
}
