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
	"github.com/Grivvus/ym/internal/repository"
	"github.com/Grivvus/ym/internal/storage"
	"github.com/Grivvus/ym/internal/utils"
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
	repo                   repository.TrackRepository
	userRepo               repository.UserRepository
	objStorage             storage.Storage
	logger                 *slog.Logger
	transcodingQueueSignal chan<- struct{}
}

func NewTrackService(
	repo repository.TrackRepository, userRepo repository.UserRepository,
	st storage.Storage, logger *slog.Logger,
	transcodingQueueSignal chan<- struct{},
) TrackService {
	return TrackService{
		repo:                   repo,
		userRepo:               userRepo,
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
	var albumID *int32
	var newAlbum *repository.CreateTrackAlbumParams
	if params.IsSingle {
		newAlbum = &repository.CreateTrackAlbumParams{
			Name:     params.Name,
			ArtistID: params.ArtistID,
		}
	} else {
		if params.AlbumID == nil {
			return ret, fmt.Errorf(
				"%w: albumID is required if track is not a single", ErrBadParams,
			)
		}
		albumID = params.AlbumID
	}

	track, err := s.repo.CreateTrackWithAlbum(ctx, repository.CreateTrackParams{
		Name:                params.Name,
		ArtistID:            params.ArtistID,
		UploadBy:            params.UploadBy,
		IsGloballyAvailable: params.IsGloballyAvailable,
		AlbumID:             albumID,
		NewAlbum:            newAlbum,
	})
	if err != nil {
		if errors.Is(err, repository.ErrAlreadyExists) {
			if params.IsSingle {
				return ret, fmt.Errorf(
					"%w: album with this name already exists",
					NewErrAlreadyExists("album", params.Name),
				)
			}
			return ret, fmt.Errorf(
				"%w: album already has this track",
				NewErrAlreadyExists("track", params.Name),
			)
		}
		return ret, fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}
	ret.TrackId = track.ID

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

	err = s.repo.AddToTranscodingQueue(ctx, track.ID, tmpFname)
	if err != nil {
		return ret, fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}

	s.transcodingQueueSignal <- struct{}{}

	return ret, nil
}

func (s *TrackService) DeleteTrack(ctx context.Context, userID, trackID int32) error {
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrUnauthorized
		}
		return fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}
	if !user.IsSuperuser {
		return ErrSuperuserRequired
	}

	track, err := s.getTrack(ctx, trackID)
	if err != nil {
		return err
	}

	errs := make([]error, 0, 5)
	if track.FastPresetName != nil {
		errs = append(errs, s.objStorage.RemoveTrack(ctx, *track.FastPresetName))
	}
	if track.StandardPresetName != nil {
		errs = append(errs, s.objStorage.RemoveTrack(ctx, *track.StandardPresetName))
	}
	if track.HighPresetName != nil {
		errs = append(errs, s.objStorage.RemoveTrack(ctx, *track.HighPresetName))
	}
	if track.LosslessPresetName != nil {
		errs = append(errs, s.objStorage.RemoveTrack(ctx, *track.LosslessPresetName))
	}
	for _, err := range errs {
		if err != nil {
			return fmt.Errorf("can't remove track file: %w", err)
		}
	}
	err = s.objStorage.RemoveTrack(ctx, originalTrackStorageKey(track.ID))
	if err != nil {
		return fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}
	if err := s.repo.DeleteTrack(ctx, trackID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return NewErrNotFound("track", trackID)
		}
		return fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}
	return nil
}

func (s *TrackService) GetMeta(
	ctx context.Context, trackID int32,
) (api.TrackMetadata, error) {
	trackInfo, err := s.repo.GetTrack(ctx, trackID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return api.TrackMetadata{}, NewErrNotFound("track", trackID)
		}
		return api.TrackMetadata{}, fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}

	return apiTrackMetadataFromRepositoryTrack(trackInfo), nil
}

func (s *TrackService) GetUserMeta(
	ctx context.Context, userID, trackID int32,
) (api.TrackMetadata, error) {
	trackInfo, err := s.getTrackForUser(ctx, userID, trackID)
	if err != nil {
		return api.TrackMetadata{}, err
	}

	return apiTrackMetadataFromRepositoryTrack(trackInfo), nil
}

func (s *TrackService) GetUserTracks(
	ctx context.Context, userID int32,
) ([]api.TrackMetadata, error) {
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrUnauthorized
		}
		return nil, fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}

	var tracks []repository.Track
	if user.IsSuperuser {
		tracks, err = s.repo.GetAllTracks(ctx)
	} else {
		tracks, err = s.repo.GetUserTracks(ctx, userID)
	}
	if err != nil {
		return nil, fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}
	ret := make([]api.TrackMetadata, 0, len(tracks))
	for _, track := range tracks {
		albumID, err := s.repo.GetAlbumIDByTrackID(ctx, track.ID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				s.logger.Warn(
					"track without assosiated album",
					slog.Int("trackID", int(track.ID)),
				)
				continue
			}
			return nil, fmt.Errorf("%w: exact - %w", ErrUnknownDBError, err)
		}
		track.AlbumID = albumID
		ret = append(ret, api.TrackMetadata{
			ArtistId:            track.ArtistID,
			Name:                track.Name,
			AlbumId:             albumID,
			DurationMs:          track.DurationMs,
			TrackId:             track.ID,
			TrackFastPreset:     track.FastPresetName,
			TrackStandardPreset: track.StandardPresetName,
			TrackHighPreset:     track.HighPresetName,
			TrackLosslessPreset: track.LosslessPresetName,
		})
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
	track, err := s.getTrackForUser(ctx, userID, trackID)
	if err != nil {
		return TrackStream{}, err
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

func (s *TrackService) getTrack(ctx context.Context, trackID int32) (repository.Track, error) {
	track, err := s.repo.GetTrack(ctx, trackID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return track, NewErrNotFound("track", trackID)
		}
		return track, fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}
	return track, nil
}

func (s *TrackService) getTrackForUser(
	ctx context.Context, userID, trackID int32,
) (repository.Track, error) {
	track, err := s.getTrack(ctx, trackID)
	if err != nil {
		return repository.Track{}, err
	}
	canAccess, err := s.repo.CanUserAccessTrack(ctx, userID, trackID)
	if err != nil {
		return repository.Track{}, fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}
	if !canAccess {
		return repository.Track{}, fmt.Errorf("%w: user can't have access to this track", ErrUnauthorized)
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
	track repository.Track, chosenPreset audio.Preset,
) (selectedTrackFile, error) {
	switch chosenPreset {
	// lossless->orig
	case audio.PresetLossless:
		if track.LosslessPresetName != nil {
			return selectedPresetTrackFile(*track.LosslessPresetName, audio.PresetLossless), nil
		}
		return selectedOriginalTrackFile(track.ID), nil
	// H->S->F->orig
	case audio.PresetHigh:
		if track.HighPresetName != nil {
			return selectedPresetTrackFile(*track.HighPresetName, audio.PresetHigh), nil
		} else if track.StandardPresetName != nil {
			return selectedPresetTrackFile(*track.StandardPresetName, audio.PresetStandard), nil
		} else if track.FastPresetName != nil {
			return selectedPresetTrackFile(*track.FastPresetName, audio.PresetFast), nil
		}
		return selectedOriginalTrackFile(track.ID), nil
	// S->F->H->orig
	case audio.PresetStandard:
		if track.StandardPresetName != nil {
			return selectedPresetTrackFile(*track.StandardPresetName, audio.PresetStandard), nil
		} else if track.FastPresetName != nil {
			return selectedPresetTrackFile(*track.FastPresetName, audio.PresetFast), nil
		} else if track.HighPresetName != nil {
			return selectedPresetTrackFile(*track.HighPresetName, audio.PresetHigh), nil
		}
		return selectedOriginalTrackFile(track.ID), nil
	// F->S->H->orig
	case audio.PresetFast:
		if track.FastPresetName != nil {
			return selectedPresetTrackFile(*track.FastPresetName, audio.PresetFast), nil
		} else if track.StandardPresetName != nil {
			return selectedPresetTrackFile(*track.StandardPresetName, audio.PresetStandard), nil
		} else if track.HighPresetName != nil {
			return selectedPresetTrackFile(*track.HighPresetName, audio.PresetHigh), nil
		}
		return selectedOriginalTrackFile(track.ID), nil
	default:
		return selectedTrackFile{}, ErrPresetCantBeSelected
	}
}

func findClosestExistingTrackKey(
	track repository.Track, chosenPreset audio.Preset,
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

func canAccessTrack(track repository.Track, userID int32) bool {
	if track.IsGloballyAvailable {
		return true
	}
	return track.UploadBy != nil && *track.UploadBy == userID
}

func apiTrackMetadataFromRepositoryTrack(track repository.Track) api.TrackMetadata {
	return api.TrackMetadata{
		TrackId:             track.ID,
		ArtistId:            track.ArtistID,
		AlbumId:             track.AlbumID,
		Name:                track.Name,
		DurationMs:          track.DurationMs,
		TrackFastPreset:     track.FastPresetName,
		TrackStandardPreset: track.StandardPresetName,
		TrackHighPreset:     track.HighPresetName,
		TrackLosslessPreset: track.LosslessPresetName,
	}
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
