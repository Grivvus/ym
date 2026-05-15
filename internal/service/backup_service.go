package service

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Grivvus/ym/internal/api"
	"github.com/Grivvus/ym/internal/db"
	"github.com/Grivvus/ym/internal/dto"
	"github.com/Grivvus/ym/internal/storage"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	backupFormatVersion = 1

	backupManifestPath         = "manifest.json"
	backupDBPath               = "db/full.json"
	backupImagesPrefix         = "images/"
	backupOriginalTracksPrefix = "tracks/original/"
	backupTranscodedPrefix     = "tracks/transcoded/"
)

type BackupSettings struct {
	IncludeImages           bool
	IncludeTranscodedTracks bool
}

type BackupService struct {
	logger     *slog.Logger
	queries    *db.Queries
	objStorage storage.Storage
	backupMu   *sync.Mutex
	restoreMu  *sync.Mutex
}

type backupManifest struct {
	FormatVersion          int       `json:"format_version"`
	CreatedAt              time.Time `json:"created_at"`
	IncludeImages          bool      `json:"include_images"`
	IncludeOriginalTracks  bool      `json:"include_original_tracks"`
	IncludeTranscodedFiles bool      `json:"include_transcoded_tracks"`
}

type archiveObject struct {
	archivePath string
	storageKey  string
}

type removeOnCloseFile struct {
	*os.File
	path string
}

func (f removeOnCloseFile) Close() error {
	closeErr := f.File.Close()
	removeErr := os.Remove(f.path)
	if closeErr != nil {
		return closeErr
	}
	if removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
		return removeErr
	}
	return nil
}

func NewBackupService(
	logger *slog.Logger, queries *db.Queries, storage storage.Storage,
) BackupService {
	return BackupService{
		logger:     logger,
		queries:    queries,
		objStorage: storage,
		backupMu:   &sync.Mutex{},
		restoreMu:  &sync.Mutex{},
	}
}

func (service BackupService) MakeBackup(
	ctx context.Context, settings BackupSettings,
) (backup io.ReadCloser, clen uint, err error) {
	archivePath, size, err := service.makeBackupArchive(ctx, settings)
	if err != nil {
		return emptyBackupReader(), 0, err
	}

	reader, err := os.Open(archivePath)
	if err != nil {
		_ = os.Remove(archivePath)
		return emptyBackupReader(), 0, fmt.Errorf("can't open backup archive: %w", err)
	}

	return removeOnCloseFile{
		File: reader,
		path: archivePath,
	}, uint(size), nil
}

func (service BackupService) StartBackupOperation(
	ctx context.Context, settings BackupSettings,
) (backupID string, err error) {
	service.backupMu.Lock()
	defer service.backupMu.Unlock()

	activeBackup, err := service.queries.GetActiveBackupOperation(ctx)
	if err == nil {
		return "", NewErrAlreadyExists("backup operation", activeBackup.ID)
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}

	backupID, err = newBackupID()
	if err != nil {
		return "", err
	}

	backupID, err = service.queries.StartBackupOperation(ctx, db.StartBackupOperationParams{
		ID:                      backupID,
		IncludeImages:           settings.IncludeImages,
		IncludeTranscodedTracks: settings.IncludeTranscodedTracks,
	})
	if err != nil {
		return "", fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}

	go service.runBackupOperation(context.WithoutCancel(ctx), backupID, settings)

	return backupID, nil
}

func (service BackupService) CheckBackupOperation(
	ctx context.Context, backupID string,
) (response api.BackupStatusResponse, err error) {
	status, err := service.queries.GetBackupStatus(ctx, backupID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return response, NewErrNotFound("backup_status", backupID)
		}
		return response, fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}

	return backupStatusResponse(status)
}

func (service BackupService) DownloadBackup(
	ctx context.Context, backupID string,
) (backup io.ReadCloser, clen uint, err error) {
	status, err := service.queries.GetBackupStatus(ctx, backupID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return emptyBackupReader(), 0, NewErrNotFound("backup_status", backupID)
		}
		return emptyBackupReader(), 0, fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}

	if status.Status != db.StatusFinished {
		return emptyBackupReader(), 0, fmt.Errorf("%w: backup %s is not finished", ErrBadParams, backupID)
	}
	if !status.ArchivePath.Valid || status.ArchivePath.String == "" {
		return emptyBackupReader(), 0, fmt.Errorf("backup %s finished without archive path", backupID)
	}

	reader, err := os.Open(status.ArchivePath.String)
	if err != nil {
		return emptyBackupReader(), 0, fmt.Errorf("can't open backup archive: %w", err)
	}

	size := status.SizeBytes.Int64
	if !status.SizeBytes.Valid {
		info, statErr := reader.Stat()
		if statErr != nil {
			_ = reader.Close()
			return emptyBackupReader(), 0, fmt.Errorf("can't stat backup archive: %w", statErr)
		}
		size = info.Size()
	}

	return reader, uint(size), nil
}

func (service BackupService) runBackupOperation(
	ctx context.Context, backupID string, settings BackupSettings,
) {
	defer func() {
		if recoverValue := recover(); recoverValue != nil {
			err := fmt.Errorf("backup panicked: %v", recoverValue)
			service.logger.Error("backup panicked", "backup_id", backupID, "error", err)
			service.finishBackupWithError(context.Background(), backupID, err)
		}
	}()

	if err := service.queries.MarkBackupOperationStarted(ctx, backupID); err != nil {
		service.logger.Error("can't mark backup as started", "backup_id", backupID, "error", err)
		service.finishBackupWithError(context.Background(), backupID, err)
		return
	}

	service.logger.Info("backup started", "backup_id", backupID)

	archivePath, size, err := service.makeBackupArchive(ctx, settings)
	if err != nil {
		service.logger.Error("backup failed", "backup_id", backupID, "error", err)
		service.finishBackupWithError(context.Background(), backupID, err)
		return
	}

	if err := service.queries.ConfirmBackup(ctx, db.ConfirmBackupParams{
		ID:          backupID,
		ArchivePath: pgtype.Text{String: archivePath, Valid: true},
		SizeBytes:   pgtype.Int8{Int64: size, Valid: true},
	}); err != nil {
		_ = os.Remove(archivePath)
		service.logger.Error(
			"backup finished but status update failed",
			"backup_id", backupID,
			"error", err,
		)
		return
	}

	service.logger.Info("backup finished", "backup_id", backupID)
}

func (service BackupService) makeBackupArchive(
	ctx context.Context, settings BackupSettings,
) (archivePath string, size int64, err error) {
	snapshot, err := service.backupDB(ctx)
	if err != nil {
		return "", 0, err
	}

	createdAt := time.Now().UTC()
	if !settings.IncludeTranscodedTracks {
		service.prepareOriginalOnlySnapshot(&snapshot, createdAt)
	}

	manifest := backupManifest{
		FormatVersion:          backupFormatVersion,
		CreatedAt:              createdAt,
		IncludeImages:          settings.IncludeImages,
		IncludeOriginalTracks:  true,
		IncludeTranscodedFiles: settings.IncludeTranscodedTracks,
	}

	tmpFile, err := os.CreateTemp("", "ym-backup-*.zip")
	if err != nil {
		return "", 0, fmt.Errorf("can't create backup archive: %w", err)
	}
	archivePath = tmpFile.Name()

	cleanupTmp := func() {
		_ = tmpFile.Close()
		_ = os.Remove(archivePath)
	}

	zipWriter := zip.NewWriter(tmpFile)

	if err := writeZipJSON(zipWriter, backupManifestPath, manifest); err != nil {
		cleanupTmp()
		return "", 0, fmt.Errorf("can't write backup manifest: %w", err)
	}
	if err := writeZipJSON(zipWriter, backupDBPath, snapshot); err != nil {
		cleanupTmp()
		return "", 0, fmt.Errorf("can't write database backup: %w", err)
	}
	if settings.IncludeImages {
		if err := service.backupImages(ctx, zipWriter, snapshot); err != nil {
			cleanupTmp()
			return "", 0, err
		}
	}
	if err := service.backupTracks(ctx, zipWriter, snapshot, settings.IncludeTranscodedTracks); err != nil {
		cleanupTmp()
		return "", 0, err
	}

	if err := zipWriter.Close(); err != nil {
		cleanupTmp()
		return "", 0, fmt.Errorf("can't finalize backup archive: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		cleanupTmp()
		return "", 0, fmt.Errorf("can't close backup archive: %w", err)
	}

	info, err := os.Stat(archivePath)
	if err != nil {
		cleanupTmp()
		return "", 0, fmt.Errorf("can't stat backup archive: %w", err)
	}

	return archivePath, info.Size(), nil
}

func (service BackupService) backupDB(ctx context.Context) (dto.FullDBBackup, error) {
	users, err := service.queries.GetAllUsersForBackup(ctx)
	if err != nil {
		return dto.FullDBBackup{}, fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}
	artists, err := service.queries.GetAllArtistsForBackup(ctx)
	if err != nil {
		return dto.FullDBBackup{}, fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}
	albums, err := service.queries.GetAllAlbumsForBackup(ctx)
	if err != nil {
		return dto.FullDBBackup{}, fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}
	playlists, err := service.queries.GetAllPlaylistsForBackup(ctx)
	if err != nil {
		return dto.FullDBBackup{}, fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}
	tracks, err := service.queries.GetAllTracksForBackup(ctx)
	if err != nil {
		return dto.FullDBBackup{}, fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}
	trackAlbums, err := service.queries.GetAllTrackAlbumsForBackup(ctx)
	if err != nil {
		return dto.FullDBBackup{}, fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}
	trackPlaylists, err := service.queries.GetAllTrackPlaylistsForBackup(ctx)
	if err != nil {
		return dto.FullDBBackup{}, fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}
	playlistShares, err := service.queries.GetAllPlaylistSharesForBackup(ctx)
	if err != nil {
		return dto.FullDBBackup{}, fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}
	queueRows, err := service.queries.GetAllTranscodingQueueForBackup(ctx)
	if err != nil {
		return dto.FullDBBackup{}, fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}

	backup := dto.FullDBBackup{
		Users:            make([]dto.User, 0, len(users)),
		Artists:          make([]dto.Artist, 0, len(artists)),
		Albums:           make([]dto.Album, 0, len(albums)),
		Playlists:        make([]dto.Playlist, 0, len(playlists)),
		Tracks:           make([]dto.Track, 0, len(tracks)),
		TrackAlbums:      make([]dto.TrackAlbum, 0, len(trackAlbums)),
		TrackPlaylists:   make([]dto.TrackPlaylist, 0, len(trackPlaylists)),
		PlaylistShares:   make([]dto.PlaylistShareInfo, 0, len(playlistShares)),
		TranscodingQueue: make([]dto.TranscodingQueueRow, 0, len(queueRows)),
	}

	for _, user := range users {
		backup.Users = append(backup.Users, dto.User{
			ID:             user.ID,
			Username:       user.Username,
			IsSuperuser:    user.IsSuperuser,
			Email:          textToPtr(user.Email),
			Password:       base64.StdEncoding.EncodeToString(user.Password),
			Salt:           base64.StdEncoding.EncodeToString(user.Salt),
			RefreshVersion: user.RefreshVersion,
			CreatedAt:      timestamptzToTime(user.CreatedAt),
			UpdatedAt:      timestamptzToTime(user.UpdatedAt),
		})
	}

	for _, artist := range artists {
		backup.Artists = append(backup.Artists, dto.Artist{
			ID:   artist.ID,
			Name: artist.Name,
			URL:  textToPtr(artist.Url),
		})
	}

	for _, album := range albums {
		backup.Albums = append(backup.Albums, dto.Album{
			ID:       album.ID,
			Name:     album.Name,
			ArtistID: album.ArtistID,
		})
	}

	for _, playlist := range playlists {
		backup.Playlists = append(backup.Playlists, dto.Playlist{
			ID:       playlist.ID,
			Name:     playlist.Name,
			IsPublic: playlist.IsPublic,
			OwnerID:  playlist.OwnerID,
		})
	}

	for _, track := range tracks {
		backup.Tracks = append(backup.Tracks, dto.Track{
			ID:                  track.ID,
			ArtistID:            track.ArtistID,
			Name:                track.Name,
			DurationMs:          int4ToPtr(track.DurationMs),
			URL:                 textToPtr(track.Url),
			FastPresetFname:     textToPtr(track.FastPresetFname),
			StandardPresetFname: textToPtr(track.StandardPresetFname),
			HighPresetFname:     textToPtr(track.HighPresetFname),
			LosslessPresetFname: textToPtr(track.LosslessPresetFname),
			IsGloballyAvailable: track.IsGloballyAvailable,
			UploadByUser:        int4ToPtr(track.UploadByUser),
		})
	}

	for _, row := range trackAlbums {
		backup.TrackAlbums = append(backup.TrackAlbums, dto.TrackAlbum{
			TrackID: row.TrackID,
			AlbumID: row.AlbumID,
		})
	}

	for _, row := range trackPlaylists {
		backup.TrackPlaylists = append(backup.TrackPlaylists, dto.TrackPlaylist{
			TrackID:    row.TrackID,
			PlaylistID: row.PlaylistID,
		})
	}

	for _, row := range playlistShares {
		backup.PlaylistShares = append(backup.PlaylistShares, dto.PlaylistShareInfo{
			PlaylistID:         row.PlaylistID,
			SharedWithUser:     row.SharedWithUser,
			HasWritePermission: row.HasWritePermission,
		})
	}

	for _, row := range queueRows {
		backup.TranscodingQueue = append(backup.TranscodingQueue, dto.TranscodingQueueRow{
			ID:                    row.ID,
			TrackID:               row.TrackID,
			AddedAt:               timestampToTime(row.AddedAt),
			TrackOriginalFileName: row.TrackOriginalFileName,
			WasFailed:             row.WasFailed,
			ErrorMsg:              textToPtr(row.ErrorMsg),
		})
	}

	return backup, nil
}

func (service BackupService) backupImages(
	ctx context.Context, zipWriter *zip.Writer, snapshot dto.FullDBBackup,
) error {
	for _, obj := range imageArchiveObjects(snapshot) {
		payload, err := service.objStorage.GetImage(ctx, obj.storageKey)
		if err != nil {
			if errors.Is(err, storage.ErrObjectNotFound) {
				continue
			}
			return fmt.Errorf("can't backup image %q: %w", obj.storageKey, err)
		}

		writer, err := zipWriter.Create(obj.archivePath)
		if err != nil {
			return fmt.Errorf("can't create image entry %q: %w", obj.archivePath, err)
		}
		if _, err := writer.Write(payload); err != nil {
			return fmt.Errorf("can't write image %q to archive: %w", obj.storageKey, err)
		}
	}
	return nil
}

func (service BackupService) backupTracks(
	ctx context.Context, zipWriter *zip.Writer,
	snapshot dto.FullDBBackup, includeTranscoded bool,
) error {
	for _, obj := range originalTrackArchiveObjects(snapshot) {
		if err := service.writeTrackToArchive(ctx, zipWriter, obj); err != nil {
			return err
		}
	}

	if !includeTranscoded {
		return nil
	}

	for _, obj := range transcodedTrackArchiveObjects(snapshot) {
		if err := service.writeTrackToArchive(ctx, zipWriter, obj); err != nil {
			return err
		}
	}

	return nil
}

func (service BackupService) StartRestoreOperation(
	ctx context.Context, backup io.Reader,
) (restoreID string, err error) {
	service.restoreMu.Lock()
	defer service.restoreMu.Unlock()

	activeRestore, err := service.queries.GetActiveRestoreOperation(ctx)
	if err == nil {
		return "", NewErrAlreadyExists("restore operation", activeRestore.ID)
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}

	tmpFile, err := os.CreateTemp("", "ym-restore-*.zip")
	if err != nil {
		return "", fmt.Errorf("can't create restore archive temp file: %w", err)
	}

	cleanupTmp := func() {
		_ = tmpFile.Close()
		_ = os.Remove(tmpFile.Name())
	}

	if _, err := io.Copy(tmpFile, backup); err != nil {
		cleanupTmp()
		return "", fmt.Errorf("can't persist restore archive: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		cleanupTmp()
		return "", fmt.Errorf("can't close restore archive: %w", err)
	}

	restoreID, err = newRestoreID()
	if err != nil {
		cleanupTmp()
		return "", err
	}

	restoreID, err = service.queries.StartRestoreOperation(ctx, restoreID)
	if err != nil {
		cleanupTmp()
		return "", fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}

	go service.runRestoreOperation(context.WithoutCancel(ctx), restoreID, tmpFile.Name())

	return restoreID, nil
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
		if status.Error.Valid {
			response.Error = &status.Error.String
		}
	default:
		return response, fmt.Errorf(
			"%w: invalid status - %v", ErrBadParams, status.Status,
		)
	}

	return response, nil
}

func (service BackupService) runRestoreOperation(
	ctx context.Context, restoreID string, archivePath string,
) {
	defer func() {
		_ = os.Remove(archivePath)
	}()
	defer func() {
		if recoverValue := recover(); recoverValue != nil {
			err := fmt.Errorf("restore panicked: %v", recoverValue)
			service.logger.Error("restore panicked", "restore_id", restoreID, "error", err)
			service.finishRestoreWithError(context.Background(), restoreID, err)
		}
	}()

	if err := service.queries.MarkRestoreOperationStarted(ctx, restoreID); err != nil {
		service.logger.Error("can't mark restore as started", "restore_id", restoreID, "error", err)
		service.finishRestoreWithError(context.Background(), restoreID, err)
		return
	}

	service.logger.Info("restore started", "restore_id", restoreID)

	if err := service.restoreArchive(ctx, archivePath); err != nil {
		service.logger.Error("restore failed", "restore_id", restoreID, "error", err)
		service.finishRestoreWithError(context.Background(), restoreID, err)
		return
	}

	if err := service.queries.ConfirmRestoring(ctx, restoreID); err != nil {
		service.logger.Error(
			"restore finished but status update failed",
			"restore_id", restoreID,
			"error", err,
		)
		return
	}

	service.logger.Info("restore finished", "restore_id", restoreID)
}

func (service BackupService) restoreArchive(ctx context.Context, archivePath string) error {
	archive, err := openZipArchive(archivePath)
	if err != nil {
		return err
	}
	defer func() { _ = archive.Close() }()

	manifest, dump, err := loadBackupMetadata(&archive.Reader)
	if err != nil {
		return err
	}
	if manifest.FormatVersion != backupFormatVersion {
		return fmt.Errorf(
			"%w: unsupported backup format version %d",
			ErrBadParams, manifest.FormatVersion,
		)
	}
	if !manifest.IncludeTranscodedFiles || !archiveHasEntriesWithPrefix(&archive.Reader, backupTranscodedPrefix) {
		queuedAt := manifest.CreatedAt
		if queuedAt.IsZero() {
			queuedAt = time.Now().UTC()
		}
		service.prepareOriginalOnlySnapshot(&dump, queuedAt)
	}

	currentSnapshot, err := service.backupDB(ctx)
	if err != nil {
		return fmt.Errorf("can't snapshot current database before restore: %w", err)
	}

	if err := service.queries.ClearBackupData(ctx); err != nil {
		return fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}
	if err := service.clearStorageState(ctx, currentSnapshot); err != nil {
		return err
	}
	if err := service.restoreDBSnapshot(ctx, dump); err != nil {
		return err
	}
	if _, err := service.queries.SyncBackupSequences(ctx); err != nil {
		return fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}
	if err := service.restoreStorageState(ctx, &archive.Reader); err != nil {
		return err
	}

	return nil
}

func (service BackupService) restoreDBSnapshot(
	ctx context.Context, snapshot dto.FullDBBackup,
) error {
	for _, user := range snapshot.Users {
		password, err := base64.StdEncoding.DecodeString(user.Password)
		if err != nil {
			return fmt.Errorf("%w: invalid encoded password for user %d: %w", ErrBadParams, user.ID, err)
		}
		salt, err := base64.StdEncoding.DecodeString(user.Salt)
		if err != nil {
			return fmt.Errorf("%w: invalid encoded salt for user %d: %w", ErrBadParams, user.ID, err)
		}

		err = service.queries.RestoreUser(ctx, db.RestoreUserParams{
			ID:             user.ID,
			Username:       user.Username,
			IsSuperuser:    user.IsSuperuser,
			Email:          ptrToText(user.Email),
			Password:       password,
			Salt:           salt,
			RefreshVersion: user.RefreshVersion,
			CreatedAt:      timeToTimestamptz(user.CreatedAt),
			UpdatedAt:      timeToTimestamptz(user.UpdatedAt),
		})
		if err != nil {
			return fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
		}
	}

	for _, artist := range snapshot.Artists {
		err := service.queries.RestoreArtist(ctx, db.RestoreArtistParams{
			ID:   artist.ID,
			Name: artist.Name,
			Url:  ptrToText(artist.URL),
		})
		if err != nil {
			return fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
		}
	}

	for _, album := range snapshot.Albums {
		err := service.queries.RestoreAlbum(ctx, db.RestoreAlbumParams{
			ID:       album.ID,
			Name:     album.Name,
			ArtistID: album.ArtistID,
		})
		if err != nil {
			return fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
		}
	}

	for _, playlist := range snapshot.Playlists {
		err := service.queries.RestorePlaylist(ctx, db.RestorePlaylistParams{
			ID:       playlist.ID,
			Name:     playlist.Name,
			IsPublic: playlist.IsPublic,
			OwnerID:  playlist.OwnerID,
		})
		if err != nil {
			return fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
		}
	}

	for _, row := range snapshot.PlaylistShares {
		err := service.queries.RestorePlaylistShareInfo(ctx, db.RestorePlaylistShareInfoParams{
			PlaylistID:         row.PlaylistID,
			SharedWithUser:     row.SharedWithUser,
			HasWritePermission: row.HasWritePermission,
		})
		if err != nil {
			return fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
		}
	}

	for _, track := range snapshot.Tracks {
		err := service.queries.RestoreTrack(ctx, db.RestoreTrackParams{
			ID:                  track.ID,
			Name:                track.Name,
			DurationMs:          ptrToInt4(track.DurationMs),
			Url:                 ptrToText(track.URL),
			FastPresetFname:     ptrToText(track.FastPresetFname),
			StandardPresetFname: ptrToText(track.StandardPresetFname),
			HighPresetFname:     ptrToText(track.HighPresetFname),
			LosslessPresetFname: ptrToText(track.LosslessPresetFname),
			IsGloballyAvailable: track.IsGloballyAvailable,
			ArtistID:            track.ArtistID,
			UploadByUser:        ptrToInt4(track.UploadByUser),
		})
		if err != nil {
			return fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
		}
	}

	for _, row := range snapshot.TrackAlbums {
		err := service.queries.RestoreTrackAlbum(ctx, db.RestoreTrackAlbumParams{
			TrackID: row.TrackID,
			AlbumID: row.AlbumID,
		})
		if err != nil {
			return fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
		}
	}

	for _, row := range snapshot.TrackPlaylists {
		err := service.queries.RestoreTrackPlaylist(ctx, db.RestoreTrackPlaylistParams{
			TrackID:    row.TrackID,
			PlaylistID: row.PlaylistID,
		})
		if err != nil {
			return fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
		}
	}

	for _, row := range snapshot.TranscodingQueue {
		err := service.queries.RestoreTranscodingQueue(ctx, db.RestoreTranscodingQueueParams{
			ID:                    row.ID,
			AddedAt:               timeToTimestamp(row.AddedAt),
			TrackOriginalFileName: row.TrackOriginalFileName,
			TrackID:               row.TrackID,
			WasFailed:             row.WasFailed,
			ErrorMsg:              ptrToText(row.ErrorMsg),
		})
		if err != nil {
			return fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
		}
	}

	return nil
}

func (service BackupService) clearStorageState(
	ctx context.Context, snapshot dto.FullDBBackup,
) error {
	for _, obj := range imageArchiveObjects(snapshot) {
		if err := service.objStorage.RemoveImage(ctx, obj.storageKey); err != nil &&
			!errors.Is(err, storage.ErrObjectNotFound) {
			return fmt.Errorf("can't remove current image %q: %w", obj.storageKey, err)
		}
	}

	seenTrackKeys := make(map[string]struct{})
	for _, obj := range append(
		originalTrackArchiveObjects(snapshot),
		transcodedTrackArchiveObjects(snapshot)...,
	) {
		if _, ok := seenTrackKeys[obj.storageKey]; ok {
			continue
		}
		seenTrackKeys[obj.storageKey] = struct{}{}
		if err := service.objStorage.RemoveTrack(ctx, obj.storageKey); err != nil &&
			!errors.Is(err, storage.ErrObjectNotFound) {
			return fmt.Errorf("can't remove current track object %q: %w", obj.storageKey, err)
		}
	}

	return nil
}

func (service BackupService) restoreStorageState(
	ctx context.Context, archive *zip.Reader,
) error {
	for _, file := range archive.File {
		if file.FileInfo().IsDir() {
			continue
		}

		switch {
		case strings.HasPrefix(file.Name, backupImagesPrefix):
			key, err := decodeArchiveObjectKey(backupImagesPrefix, file.Name)
			if err != nil {
				return err
			}
			rc, err := file.Open()
			if err != nil {
				return fmt.Errorf("can't open archived image %q: %w", file.Name, err)
			}
			err = service.objStorage.PutImage(ctx, key, rc)
			closeErr := rc.Close()
			if err != nil {
				return fmt.Errorf("can't restore image %q: %w", key, err)
			}
			if closeErr != nil {
				return fmt.Errorf("can't close archived image %q: %w", file.Name, closeErr)
			}
		case strings.HasPrefix(file.Name, backupOriginalTracksPrefix):
			if err := service.restoreTrackFile(ctx, file, backupOriginalTracksPrefix); err != nil {
				return err
			}
		case strings.HasPrefix(file.Name, backupTranscodedPrefix):
			if err := service.restoreTrackFile(ctx, file, backupTranscodedPrefix); err != nil {
				return err
			}
		}
	}

	return nil
}

func (service BackupService) restoreTrackFile(
	ctx context.Context, file *zip.File, prefix string,
) error {
	key, err := decodeArchiveObjectKey(prefix, file.Name)
	if err != nil {
		return err
	}

	rc, err := file.Open()
	if err != nil {
		return fmt.Errorf("can't open archived track %q: %w", file.Name, err)
	}
	err = service.objStorage.PutTrack(
		ctx, key, rc, storage.PutTrackOptions{Size: int64(file.UncompressedSize64)},
	)
	closeErr := rc.Close()
	if err != nil {
		return fmt.Errorf("can't restore track %q: %w", key, err)
	}
	if closeErr != nil {
		return fmt.Errorf("can't close archived track %q: %w", file.Name, closeErr)
	}

	return nil
}

func (service BackupService) writeTrackToArchive(
	ctx context.Context, zipWriter *zip.Writer, obj archiveObject,
) error {
	reader, err := service.objStorage.GetTrack(ctx, obj.storageKey)
	if err != nil {
		return fmt.Errorf("can't backup track object %q: %w", obj.storageKey, err)
	}
	defer func() { _ = reader.Close() }()

	writer, err := zipWriter.Create(obj.archivePath)
	if err != nil {
		return fmt.Errorf("can't create track entry %q: %w", obj.archivePath, err)
	}

	if _, err := io.Copy(writer, reader); err != nil {
		return fmt.Errorf("can't write track object %q to archive: %w", obj.storageKey, err)
	}

	return nil
}

func (service BackupService) prepareOriginalOnlySnapshot(
	snapshot *dto.FullDBBackup, queuedAt time.Time,
) {
	queuedTrackIDs := make(map[int32]struct{}, len(snapshot.TranscodingQueue))
	var lastQueueID int64

	for i := range snapshot.Tracks {
		snapshot.Tracks[i].FastPresetFname = nil
		snapshot.Tracks[i].StandardPresetFname = nil
		snapshot.Tracks[i].HighPresetFname = nil
		snapshot.Tracks[i].LosslessPresetFname = nil
	}

	for _, queueRow := range snapshot.TranscodingQueue {
		queuedTrackIDs[queueRow.TrackID] = struct{}{}
		if queueRow.ID > lastQueueID {
			lastQueueID = queueRow.ID
		}
	}

	for _, track := range snapshot.Tracks {
		if _, ok := queuedTrackIDs[track.ID]; ok {
			continue
		}
		lastQueueID++
		snapshot.TranscodingQueue = append(snapshot.TranscodingQueue, dto.TranscodingQueueRow{
			ID:                    lastQueueID,
			TrackID:               track.ID,
			AddedAt:               queuedAt,
			TrackOriginalFileName: originalTrackStorageKey(track.ID),
			WasFailed:             false,
			ErrorMsg:              nil,
		})
	}
}

func archiveHasEntriesWithPrefix(archive *zip.Reader, prefix string) bool {
	for _, file := range archive.File {
		if file.FileInfo().IsDir() {
			continue
		}
		if strings.HasPrefix(file.Name, prefix) {
			return true
		}
	}
	return false
}

func (service BackupService) finishRestoreWithError(
	ctx context.Context, restoreID string, err error,
) {
	updateErr := service.queries.ErrorRestoring(ctx, db.ErrorRestoringParams{
		ID:    restoreID,
		Error: pgtype.Text{String: err.Error(), Valid: true},
	})
	if updateErr != nil {
		service.logger.Error(
			"can't mark restore as failed",
			"restore_id", restoreID,
			"error", updateErr,
		)
	}
}

func (service BackupService) finishBackupWithError(
	ctx context.Context, backupID string, err error,
) {
	updateErr := service.queries.ErrorBackup(ctx, db.ErrorBackupParams{
		ID:    backupID,
		Error: pgtype.Text{String: err.Error(), Valid: true},
	})
	if updateErr != nil {
		service.logger.Error(
			"can't mark backup as failed",
			"backup_id", backupID,
			"error", updateErr,
		)
	}
}

func backupStatusResponse(status db.GetBackupStatusRow) (
	api.BackupStatusResponse, error,
) {
	response := api.BackupStatusResponse{
		BackupId:                status.ID,
		Status:                  string(status.Status),
		IncludeImages:           status.IncludeImages,
		IncludeTranscodedTracks: status.IncludeTranscodedTracks,
	}

	switch status.Status {
	case db.StatusPending, db.StatusStarted, db.StatusFinished:
	case db.StatusError:
		if status.Error.Valid {
			response.Error = &status.Error.String
		}
	default:
		return response, fmt.Errorf(
			"%w: invalid status - %v", ErrBadParams, status.Status,
		)
	}

	if status.SizeBytes.Valid {
		response.SizeBytes = &status.SizeBytes.Int64
	}

	return response, nil
}

func openZipArchive(path string) (*zip.ReadCloser, error) {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid zip archive: %w", ErrBadParams, err)
	}

	return reader, nil
}

func loadBackupMetadata(archive *zip.Reader) (backupManifest, dto.FullDBBackup, error) {
	manifestFile := findZipFile(archive, backupManifestPath)
	if manifestFile == nil {
		return backupManifest{}, dto.FullDBBackup{}, fmt.Errorf("%w: backup manifest is missing", ErrBadParams)
	}

	dbFile := findZipFile(archive, backupDBPath)
	if dbFile == nil {
		return backupManifest{}, dto.FullDBBackup{}, fmt.Errorf("%w: database backup file is missing", ErrBadParams)
	}

	var manifest backupManifest
	if err := readZipJSON(manifestFile, &manifest); err != nil {
		return backupManifest{}, dto.FullDBBackup{}, fmt.Errorf("%w: invalid backup manifest: %w", ErrBadParams, err)
	}

	var dump dto.FullDBBackup
	if err := readZipJSON(dbFile, &dump); err != nil {
		return backupManifest{}, dto.FullDBBackup{}, fmt.Errorf("%w: invalid database backup: %w", ErrBadParams, err)
	}

	return manifest, dump, nil
}

func findZipFile(archive *zip.Reader, name string) *zip.File {
	for _, file := range archive.File {
		if file.Name == name {
			return file
		}
	}
	return nil
}

func readZipJSON(file *zip.File, dst any) error {
	reader, err := file.Open()
	if err != nil {
		return err
	}
	defer func() { _ = reader.Close() }()

	return json.NewDecoder(reader).Decode(dst)
}

func writeZipJSON(zipWriter *zip.Writer, path string, payload any) error {
	writer, err := zipWriter.Create(path)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(payload)
}

func imageArchiveObjects(snapshot dto.FullDBBackup) []archiveObject {
	objects := make([]archiveObject, 0,
		len(snapshot.Users)+len(snapshot.Artists)+len(snapshot.Albums)+len(snapshot.Playlists),
	)

	for _, user := range snapshot.Users {
		key := ArtworkOwner{Kind: "user", ID: user.ID, Name: user.Username}.Key()
		objects = append(objects, archiveObject{
			archivePath: backupImagesPrefix + encodeArchiveObjectKey(key),
			storageKey:  key,
		})
	}
	for _, artist := range snapshot.Artists {
		key := ArtworkOwner{Kind: "artist", ID: artist.ID, Name: artist.Name}.Key()
		objects = append(objects, archiveObject{
			archivePath: backupImagesPrefix + encodeArchiveObjectKey(key),
			storageKey:  key,
		})
	}
	for _, album := range snapshot.Albums {
		key := ArtworkOwner{Kind: "album", ID: album.ID, Name: album.Name}.Key()
		objects = append(objects, archiveObject{
			archivePath: backupImagesPrefix + encodeArchiveObjectKey(key),
			storageKey:  key,
		})
	}
	for _, playlist := range snapshot.Playlists {
		key := ArtworkOwner{Kind: "playlist", ID: playlist.ID, Name: playlist.Name}.Key()
		objects = append(objects, archiveObject{
			archivePath: backupImagesPrefix + encodeArchiveObjectKey(key),
			storageKey:  key,
		})
	}

	return objects
}

func originalTrackArchiveObjects(snapshot dto.FullDBBackup) []archiveObject {
	seen := make(map[string]struct{}, len(snapshot.Tracks)+len(snapshot.TranscodingQueue))
	objects := make([]archiveObject, 0, len(snapshot.Tracks)+len(snapshot.TranscodingQueue))

	for _, track := range snapshot.Tracks {
		key := originalTrackStorageKey(track.ID)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		objects = append(objects, archiveObject{
			archivePath: backupOriginalTracksPrefix + encodeArchiveObjectKey(key),
			storageKey:  key,
		})
	}

	for _, row := range snapshot.TranscodingQueue {
		key := row.TrackOriginalFileName
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		objects = append(objects, archiveObject{
			archivePath: backupOriginalTracksPrefix + encodeArchiveObjectKey(key),
			storageKey:  key,
		})
	}

	return objects
}

func transcodedTrackArchiveObjects(snapshot dto.FullDBBackup) []archiveObject {
	seen := make(map[string]struct{}, len(snapshot.Tracks)*4)
	objects := make([]archiveObject, 0, len(snapshot.Tracks)*4)

	appendTrack := func(key *string) {
		if key == nil {
			return
		}
		if _, ok := seen[*key]; ok {
			return
		}
		seen[*key] = struct{}{}
		objects = append(objects, archiveObject{
			archivePath: backupTranscodedPrefix + encodeArchiveObjectKey(*key),
			storageKey:  *key,
		})
	}

	for _, track := range snapshot.Tracks {
		appendTrack(track.FastPresetFname)
		appendTrack(track.StandardPresetFname)
		appendTrack(track.HighPresetFname)
		appendTrack(track.LosslessPresetFname)
	}

	return objects
}

func originalTrackStorageKey(trackID int32) string {
	return fmt.Sprintf("track%d", trackID)
}

func encodeArchiveObjectKey(key string) string {
	return url.PathEscape(key)
}

func decodeArchiveObjectKey(prefix, fileName string) (string, error) {
	suffix, ok := strings.CutPrefix(fileName, prefix)
	if !ok || suffix == "" {
		return "", fmt.Errorf("%w: invalid archive entry %q", ErrBadParams, fileName)
	}
	key, err := url.PathUnescape(suffix)
	if err != nil {
		return "", fmt.Errorf("%w: invalid archive object key %q: %w", ErrBadParams, fileName, err)
	}
	return key, nil
}

func emptyBackupReader() io.ReadCloser {
	return io.NopCloser(bytes.NewReader(nil))
}

func newRestoreID() (string, error) {
	var payload [8]byte
	if _, err := rand.Read(payload[:]); err != nil {
		return "", fmt.Errorf("can't generate restore id: %w", err)
	}
	return "restore_" + hex.EncodeToString(payload[:]), nil
}

func newBackupID() (string, error) {
	var payload [8]byte
	if _, err := rand.Read(payload[:]); err != nil {
		return "", fmt.Errorf("can't generate backup id: %w", err)
	}
	return "backup_" + hex.EncodeToString(payload[:]), nil
}

func textToPtr(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	str := value.String
	return &str
}

func int4ToPtr(value pgtype.Int4) *int32 {
	if !value.Valid {
		return nil
	}
	v := value.Int32
	return &v
}

func ptrToText(value *string) pgtype.Text {
	if value == nil {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: *value, Valid: true}
}

func ptrToInt4(value *int32) pgtype.Int4 {
	if value == nil {
		return pgtype.Int4{Valid: false}
	}
	return pgtype.Int4{Int32: *value, Valid: true}
}

func timestamptzToTime(value pgtype.Timestamptz) time.Time {
	if !value.Valid {
		return time.Time{}
	}
	return value.Time
}

func timestampToTime(value pgtype.Timestamp) time.Time {
	if !value.Valid {
		return time.Time{}
	}
	return value.Time
}

func timeToTimestamptz(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: value, Valid: !value.IsZero()}
}

func timeToTimestamp(value time.Time) pgtype.Timestamp {
	return pgtype.Timestamp{Time: value, Valid: !value.IsZero()}
}
