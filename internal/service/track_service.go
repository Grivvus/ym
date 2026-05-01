package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/Grivvus/ym/internal/api"
	"github.com/Grivvus/ym/internal/audio"
	"github.com/Grivvus/ym/internal/db"
	"github.com/Grivvus/ym/internal/storage"
	"github.com/Grivvus/ym/internal/utils"
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

const (
	defaultTrackQuality     = "standard"
	resolvedQualityOriginal = "original"
)

type StreamMeta struct {
	ContentLength uint
	ContentType   string
}

type TrackStream struct {
	Name             string
	ContentType      string
	ContentLength    int64
	ETag             string
	ChecksumSHA256   string
	RequestedQuality string
	ResolvedQuality  string
	DownloadName     string
	Reader           io.ReadSeekCloser
}

type TrackService struct {
	queries                *db.Queries
	objStorage             storage.Storage
	logger                 *slog.Logger
	transcodingQueueSignal chan<- struct{}
}

func NewTrackService(
	q *db.Queries, st storage.Storage, logger *slog.Logger,
	transcodingQueueSignal chan<- struct{},
) TrackService {
	return TrackService{
		queries:                q,
		objStorage:             st,
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

	checksum, err := utils.SHA256HexFromReadSeeker(rc)
	if err != nil {
		return ret, fmt.Errorf("can't checksum track file: %w", err)
	}
	contentType, err := detectTrackUploadContentType(trackFileHeader, rc)
	if err != nil {
		return ret, fmt.Errorf("can't detect track content type: %w", err)
	}

	tmpFname := originalTrackStorageKey(track.ID)
	err = s.objStorage.PutTrack(ctx, tmpFname, rc, storage.PutTrackOptions{
		Size:           trackFileHeader.Size,
		ContentType:    contentType,
		ChecksumSHA256: checksum,
	})
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
		errs = append(errs, s.objStorage.RemoveTrack(ctx, *metadata.TrackFastPreset))
	}
	if metadata.TrackStandardPreset != nil {
		errs = append(errs, s.objStorage.RemoveTrack(ctx, *metadata.TrackStandardPreset))
	}
	if metadata.TrackHighPreset != nil {
		errs = append(errs, s.objStorage.RemoveTrack(ctx, *metadata.TrackHighPreset))
	}
	if metadata.TrackLosslessPreset != nil {
		errs = append(errs, s.objStorage.RemoveTrack(ctx, *metadata.TrackLosslessPreset))
	}
	errs = append(errs, s.objStorage.RemoveTrack(ctx, metadata.Name))
	for _, err := range errs {
		if err != nil {
			return fmt.Errorf("can't remove track file: %w", err)
		}
	}
	err = s.objStorage.RemoveTrack(ctx, originalTrackStorageKey(metadata.TrackId))
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

	return api.TrackMetadata{
		TrackId:             trackInfo.ID,
		ArtistId:            trackInfo.ArtistID,
		AlbumId:             trackInfo.AlbumID,
		Name:                trackInfo.Name,
		DurationMs:          trackInfo.DurationMs.Int32,
		TrackFastPreset:     fastPreset,
		TrackStandardPreset: standardPreset,
		TrackHighPreset:     highPreset,
		TrackLosslessPreset: losslessPreset,
	}, nil
}

func (s *TrackService) GetUserTracks(
	ctx context.Context, userID int32,
) ([]api.TrackMetadata, error) {
	// tracks, err := s.queries.GetUserTracks(ctx, pgtype.Int4{Int32: userID, Valid: true})
	tracks, err := s.queries.GetAllTracks(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}
	ret := make([]api.TrackMetadata, len(tracks))
	for i, track := range tracks {
		albumID, err := s.queries.GetAlbumByTrackID(ctx, track.ID)
		if err != nil {
			if errors.Is(pgx.ErrNoRows, err) {
				s.logger.Warn(
					"track without assosiated album",
					slog.Int("trackID", int(track.ID)),
				)
				continue
			}
			return nil, fmt.Errorf("%w: exact - %w", ErrUnknownDBError, err)
		}
		ret[i] = api.TrackMetadata{
			ArtistId:            track.ArtistID,
			Name:                track.Name,
			AlbumId:             albumID,
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
	ctx context.Context, userID, trackID int32, trackQuality string,
) (TrackStream, error) {
	return s.getTrackFile(ctx, userID, trackID, trackQuality, true, false)
}

func (s *TrackService) GetDownload(
	ctx context.Context, userID, trackID int32, trackQuality string,
) (TrackStream, error) {
	return s.getTrackFile(ctx, userID, trackID, trackQuality, true, true)
}

func (s *TrackService) GetDownloadMeta(
	ctx context.Context, userID, trackID int32, trackQuality string,
) (TrackStream, error) {
	return s.getTrackFile(ctx, userID, trackID, trackQuality, false, true)
}

func (s *TrackService) getTrackFile(
	ctx context.Context, userID, trackID int32, trackQuality string,
	includeReader bool, includeChecksum bool,
) (TrackStream, error) {
	requestedQuality := trackQuality
	if requestedQuality == "" {
		requestedQuality = defaultTrackQuality
	}
	preset, err := audio.PresetFromString(requestedQuality)
	if err != nil {
		return TrackStream{}, fmt.Errorf(
			"%w: invalid name of trackQuality: %v", ErrBadParams, requestedQuality,
		)
	}
	track, err := s.getTrack(ctx, trackID)
	if err != nil {
		return TrackStream{}, err
	}
	if !canAccessTrack(track, userID) {
		return TrackStream{}, fmt.Errorf("%w: user can't have access to this track", ErrUnauthorized)
	}
	selected, err := findClosestExistingTrackFile(track, preset)
	if err != nil {
		return TrackStream{}, err
	}

	info, err := s.objStorage.GetTrackInfo(ctx, selected.key)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotFound) {
			return TrackStream{}, fmt.Errorf(
				"%w: track not found, exact - %w",
				NewErrNotFound("track", trackID), err,
			)
		}
		return TrackStream{}, fmt.Errorf("%w: can't fetch track info: %w", ErrUnknownDBError, err)
	}

	if includeChecksum && info.ChecksumSHA256 == "" {
		info.ChecksumSHA256, err = s.calculateTrackChecksum(ctx, trackID, selected.key)
		if err != nil {
			return TrackStream{}, err
		}
	}

	contentType := normalizeTrackContentType(info.ContentType, selected.quality)

	ret := TrackStream{
		Name:             selected.key,
		ContentType:      contentType,
		ContentLength:    info.Size,
		ETag:             info.ETag,
		ChecksumSHA256:   info.ChecksumSHA256,
		RequestedQuality: requestedQuality,
		ResolvedQuality:  selected.quality,
		DownloadName:     trackDownloadFileName(track.ID, selected.quality, contentType),
	}

	if !includeReader {
		return ret, nil
	}

	stream, err := s.objStorage.GetTrack(ctx, selected.key)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotFound) {
			return TrackStream{}, fmt.Errorf(
				"%w: track not found, exact - %w",
				NewErrNotFound("track", trackID), err,
			)
		}
		return TrackStream{}, fmt.Errorf(
			"%w: can't get track stream: %w", ErrUnknownDBError, err,
		)
	}
	ret.Reader = stream

	return ret, nil
}

func (s *TrackService) getTrack(ctx context.Context, trackID int32) (db.GetTrackRow, error) {
	track, err := s.queries.GetTrack(ctx, trackID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return track, NewErrNotFound("track", trackID)
		}
		return track, fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}
	return track, nil
}

func (s *TrackService) calculateTrackChecksum(
	ctx context.Context, trackID int32, trackKey string,
) (string, error) {
	stream, err := s.objStorage.GetTrack(ctx, trackKey)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotFound) {
			return "", fmt.Errorf(
				"%w: track not found, exact - %w",
				NewErrNotFound("track", trackID), err,
			)
		}
		return "", fmt.Errorf("%w: can't get track for checksum: %w", ErrUnknownDBError, err)
	}
	defer func() { _ = stream.Close() }()

	checksum, err := utils.SHA256HexFromReadSeeker(stream)
	if err != nil {
		return "", fmt.Errorf("%w: can't checksum track: %w", ErrUnknownDBError, err)
	}
	return checksum, nil
}

type selectedTrackFile struct {
	key     string
	quality string
}

func findClosestExistingTrackFile(
	track db.GetTrackRow, chosenPreset audio.Preset,
) (selectedTrackFile, error) {
	switch chosenPreset {
	// lossless->orig
	case audio.PresetLossless:
		if track.LosslessPresetFname.Valid {
			return selectedPresetTrackFile(track.LosslessPresetFname.String, audio.PresetLossless), nil
		}
		return selectedOriginalTrackFile(track.ID), nil
	// H->S->F->orig
	case audio.PresetHigh:
		if track.HighPresetFname.Valid {
			return selectedPresetTrackFile(track.HighPresetFname.String, audio.PresetHigh), nil
		} else if track.StandardPresetFname.Valid {
			return selectedPresetTrackFile(track.StandardPresetFname.String, audio.PresetStandard), nil
		} else if track.FastPresetFname.Valid {
			return selectedPresetTrackFile(track.FastPresetFname.String, audio.PresetFast), nil
		}
		return selectedOriginalTrackFile(track.ID), nil
	// S->F->H->orig
	case audio.PresetStandard:
		if track.StandardPresetFname.Valid {
			return selectedPresetTrackFile(track.StandardPresetFname.String, audio.PresetStandard), nil
		} else if track.FastPresetFname.Valid {
			return selectedPresetTrackFile(track.FastPresetFname.String, audio.PresetFast), nil
		} else if track.HighPresetFname.Valid {
			return selectedPresetTrackFile(track.HighPresetFname.String, audio.PresetHigh), nil
		}
		return selectedOriginalTrackFile(track.ID), nil
	// F->S->H->orig
	case audio.PresetFast:
		if track.FastPresetFname.Valid {
			return selectedPresetTrackFile(track.FastPresetFname.String, audio.PresetFast), nil
		} else if track.StandardPresetFname.Valid {
			return selectedPresetTrackFile(track.StandardPresetFname.String, audio.PresetStandard), nil
		} else if track.HighPresetFname.Valid {
			return selectedPresetTrackFile(track.HighPresetFname.String, audio.PresetHigh), nil
		}
		return selectedOriginalTrackFile(track.ID), nil
	default:
		return selectedTrackFile{}, ErrPresetCantBeSelected
	}
}

func findClosestExistingTrackKey(
	track db.GetTrackRow, chosenPreset audio.Preset,
) (string, error) {
	selected, err := findClosestExistingTrackFile(track, chosenPreset)
	if err != nil {
		return "", err
	}
	return selected.key, nil
}

func selectedPresetTrackFile(key string, preset audio.Preset) selectedTrackFile {
	return selectedTrackFile{key: key, quality: preset.String()}
}

func selectedOriginalTrackFile(trackID int32) selectedTrackFile {
	return selectedTrackFile{
		key:     originalTrackStorageKey(trackID),
		quality: resolvedQualityOriginal,
	}
}

func canAccessTrack(track db.GetTrackRow, userID int32) bool {
	if track.IsGloballyAvailable {
		return true
	}
	return track.UploadByUser.Valid && track.UploadByUser.Int32 == userID
}

func detectTrackUploadContentType(
	header *multipart.FileHeader, file multipart.File,
) (string, error) {
	contentType := header.Header.Get("Content-Type")
	if contentType != "" && contentType != "application/octet-stream" {
		return contentType, nil
	}

	var sniff [512]byte
	n, err := file.Read(sniff[:])
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	if n == 0 {
		return "application/octet-stream", nil
	}
	return http.DetectContentType(sniff[:n]), nil
}

func normalizeTrackContentType(contentType string, resolvedQuality string) string {
	if contentType != "" && contentType != "application/octet-stream" {
		return contentType
	}
	switch resolvedQuality {
	case audio.PresetFast.String(), audio.PresetStandard.String():
		return "audio/ogg"
	case audio.PresetHigh.String():
		return "audio/mp4"
	case audio.PresetLossless.String():
		return "audio/flac"
	default:
		return "application/octet-stream"
	}
}

func trackDownloadFileName(trackID int32, resolvedQuality string, contentType string) string {
	return fmt.Sprintf("track-%d-%s.%s", trackID, resolvedQuality, trackExtension(resolvedQuality, contentType))
}

func trackExtension(resolvedQuality string, contentType string) string {
	switch resolvedQuality {
	case audio.PresetFast.String(), audio.PresetStandard.String():
		return "opus"
	case audio.PresetHigh.String():
		return "m4a"
	case audio.PresetLossless.String():
		return "flac"
	}

	switch strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0])) {
	case "audio/mpeg", "audio/mp3":
		return "mp3"
	case "audio/mp4", "audio/x-m4a":
		return "m4a"
	case "audio/flac", "audio/x-flac":
		return "flac"
	case "audio/opus":
		return "opus"
	case "audio/ogg":
		return "ogg"
	case "audio/wav", "audio/x-wav":
		return "wav"
	default:
		return "bin"
	}
}
