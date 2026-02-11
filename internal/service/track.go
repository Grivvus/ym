package service

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/Grivvus/ym/internal/api"
	"github.com/Grivvus/ym/internal/db"
	"github.com/Grivvus/ym/internal/storage"
	"github.com/Grivvus/ym/internal/transcoder"
	"github.com/Grivvus/ym/internal/utils"
	"github.com/jackc/pgx/v5/pgtype"
)

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
	errs = append(errs, s.st.RemoveTrack(ctx, metadata.TrackFastPreset))
	errs = append(errs, s.st.RemoveTrack(ctx, metadata.TrackStandardPreset))
	errs = append(errs, s.st.RemoveTrack(ctx, metadata.TrackHighPreset))
	errs = append(errs, s.st.RemoveTrack(ctx, metadata.TrackLosslessPreset))
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

	if !trackInfo.FastPresetFname.Valid ||
		!trackInfo.StandardPresetFname.Valid ||
		!trackInfo.HighPresetFname.Valid ||
		!trackInfo.LosslessPresetFname.Valid {
		return api.TrackMetadata{}, fmt.Errorf("can't find existing presets for this track")
	}

	return api.TrackMetadata{
		TrackId:             int(trackInfo.ID),
		ArtistId:            int(trackInfo.ArtistID),
		CoverUrl:            nil,
		Name:                trackInfo.Name,
		TrackFastPreset:     trackInfo.FastPresetFname.String,
		TrackStandardPreset: trackInfo.StandardPresetFname.String,
		TrackHighPreset:     trackInfo.HighPresetFname.String,
		TrackLosslessPreset: trackInfo.LosslessPresetFname.String,
	}, nil
}

func (s *TrackService) tmpFileName(trackID int) string {
	return "track" + strconv.Itoa(trackID)
}
