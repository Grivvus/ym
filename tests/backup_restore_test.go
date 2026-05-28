package tests

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Grivvus/ym/internal/api"
	"github.com/Grivvus/ym/internal/db"
	"github.com/Grivvus/ym/internal/service"
	"github.com/Grivvus/ym/internal/storage"
	"github.com/Grivvus/ym/internal/utils"
	"github.com/jackc/pgx/v5/pgtype"
)

func (s *IntegrationTestSuite) TestBackupAndRestore_RestoresDatabaseAndStorage() {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	backupService := service.NewBackupService(logger, s.env.Queries, s.env.Storage)
	userPasswordHash, userPasswordSalt, userPasswordParams := utils.HashPassword("password-1")

	user, err := s.env.Queries.CreateUser(ctx, db.CreateUserParams{
		Username:            "backup-user",
		Email:               pgtype.Text{String: "backup@example.com", Valid: true},
		Password:            userPasswordHash,
		Salt:                userPasswordSalt,
		PasswordMemory:      int32(userPasswordParams.Memory),
		PasswordIterations:  int32(userPasswordParams.Iterations),
		PasswordParallelism: int32(userPasswordParams.Parallelism),
		PasswordKeyLength:   int32(userPasswordParams.KeyLength),
		IsSuperuser:         true,
	})
	s.Require().NoError(err)

	sharedUser, err := s.env.Queries.CreateUser(ctx, db.CreateUserParams{
		Username:            "backup-shared-user",
		Email:               pgtype.Text{String: "backup-shared@example.com", Valid: true},
		Password:            []byte("password-hash"),
		Salt:                []byte("password-salt"),
		PasswordMemory:      int32(utils.DefaultPasswordMemory),
		PasswordIterations:  int32(utils.DefaultPasswordIterations),
		PasswordParallelism: int32(utils.DefaultPasswordHashParams().Parallelism),
		PasswordKeyLength:   int32(utils.DefaultPasswordKeyLength),
		IsSuperuser:         false,
	})
	s.Require().NoError(err)

	artist, err := s.env.Queries.CreateArtist(ctx, "backup-artist")
	s.Require().NoError(err)

	album, err := s.env.Queries.CreateAlbum(ctx, db.CreateAlbumParams{
		Name:     "backup-album",
		ArtistID: artist.ID,
	})
	s.Require().NoError(err)

	playlist, err := s.env.Queries.CreatePlaylist(ctx, db.CreatePlaylistParams{
		Name:     "backup-playlist",
		IsPublic: true,
		OwnerID:  user.ID,
	})
	s.Require().NoError(err)

	track, err := s.env.Queries.CreateTrack(ctx, db.CreateTrackParams{
		Name:                "backup-track",
		ArtistID:            artist.ID,
		IsGloballyAvailable: true,
		UploadByUser:        pgtype.Int4{Int32: user.ID, Valid: true},
	})
	s.Require().NoError(err)

	s.Require().NoError(s.env.Queries.AddTrackToAlbum(ctx, db.AddTrackToAlbumParams{
		TrackID: track.ID,
		AlbumID: album.ID,
	}))
	s.Require().NoError(s.env.Queries.AddTrackToPlaylist(ctx, db.AddTrackToPlaylistParams{
		TrackID:    track.ID,
		PlaylistID: playlist.ID,
	}))
	s.Require().NoError(s.env.Queries.SharePlaylistWith(ctx, db.SharePlaylistWithParams{
		PlaylistID:         playlist.ID,
		SharedWithUser:     sharedUser.ID,
		HasWritePermission: true,
	}))

	originalTrackKey := fmt.Sprintf("track%d", track.ID)
	fastTrackKey := originalTrackKey + "_fast"

	_, err = s.env.Queries.AddPostTranscodingInfo(ctx, db.AddPostTranscodingInfoParams{
		ID:                  track.ID,
		DurationMs:          pgtype.Int4{Int32: 12345, Valid: true},
		FastPresetFname:     pgtype.Text{String: fastTrackKey, Valid: true},
		StandardPresetFname: pgtype.Text{Valid: false},
		HighPresetFname:     pgtype.Text{Valid: false},
		LosslessPresetFname: pgtype.Text{Valid: false},
	})
	s.Require().NoError(err)
	_, err = s.env.Queries.AddToTranscodingQueue(ctx, db.AddToTranscodingQueueParams{
		TrackOriginalFileName: originalTrackKey,
		TrackID:               track.ID,
		WasFailed:             false,
		ErrorMsg:              pgtype.Text{Valid: false},
	})
	s.Require().NoError(err)

	userAvatarKey := service.ArtworkOwner{Kind: "user", ID: user.ID, Name: user.Username}.Key()
	artistImageKey := service.ArtworkOwner{Kind: "artist", ID: artist.ID, Name: artist.Name}.Key()
	albumImageKey := service.ArtworkOwner{Kind: "album", ID: album.ID, Name: album.Name}.Key()
	playlistImageKey := service.ArtworkOwner{Kind: "playlist", ID: playlist.ID, Name: playlist.Name}.Key()

	userAvatar := []byte("user-avatar")
	artistImage := []byte("artist-image")
	albumImage := []byte("album-image")
	playlistImage := []byte("playlist-image")
	originalTrack := []byte("original-track-payload")
	fastTrack := []byte("fast-track-payload")

	s.Require().NoError(s.env.Storage.PutImage(ctx, userAvatarKey, bytes.NewReader(userAvatar)))
	s.Require().NoError(s.env.Storage.PutImage(ctx, artistImageKey, bytes.NewReader(artistImage)))
	s.Require().NoError(s.env.Storage.PutImage(ctx, albumImageKey, bytes.NewReader(albumImage)))
	s.Require().NoError(s.env.Storage.PutImage(ctx, playlistImageKey, bytes.NewReader(playlistImage)))
	s.Require().NoError(s.env.Storage.PutTrack(
		ctx, originalTrackKey, bytes.NewReader(originalTrack),
		storage.PutTrackOptions{Size: int64(len(originalTrack))},
	))
	s.Require().NoError(s.env.Storage.PutTrack(
		ctx, fastTrackKey, bytes.NewReader(fastTrack),
		storage.PutTrackOptions{Size: int64(len(fastTrack))},
	))

	backupReader, _, err := backupService.MakeBackup(ctx, service.BackupSettings{
		IncludeImages:           true,
		IncludeTranscodedTracks: true,
	})
	s.Require().NoError(err)
	archiveBytes, err := io.ReadAll(backupReader)
	s.Require().NoError(err)
	s.Require().NoError(backupReader.Close())

	_, err = s.env.DB.Exec(ctx, `
		TRUNCATE TABLE
			"password_reset_code",
			"transcoding_queue",
			"track_playlist",
			"track_album",
			"track",
			"playlist",
			"album",
			"artist",
			"user",
			"backup_status",
			"restore_status"
		RESTART IDENTITY CASCADE
	`)
	s.Require().NoError(err)

	s.Require().NoError(s.env.Storage.RemoveImage(ctx, userAvatarKey))
	s.Require().NoError(s.env.Storage.RemoveImage(ctx, artistImageKey))
	s.Require().NoError(s.env.Storage.RemoveImage(ctx, albumImageKey))
	s.Require().NoError(s.env.Storage.RemoveImage(ctx, playlistImageKey))
	s.Require().NoError(s.env.Storage.RemoveTrack(ctx, originalTrackKey))
	s.Require().NoError(s.env.Storage.RemoveTrack(ctx, fastTrackKey))

	restoreID, err := backupService.StartRestoreOperation(ctx, bytes.NewReader(archiveBytes))
	s.Require().NoError(err)

	var status api.RestoreStatusResponse
	s.Eventually(func() bool {
		var statusErr error
		status, statusErr = backupService.CheckRestoreOperation(context.Background(), restoreID)
		s.Require().NoError(statusErr)
		if status.Status == api.Error {
			s.T().Logf("restore failed: %s", *status.Error)
			return true
		}
		return status.Status == api.Finished
	}, 10*time.Second, 100*time.Millisecond)
	s.Equal(api.Finished, status.Status)

	users, err := s.env.Queries.GetAllUsersForBackup(ctx)
	s.Require().NoError(err)
	s.Len(users, 2)
	s.Equal("backup-user", users[0].Username)

	loginResp := s.loginUser(api.UserAuth{
		Username: "backup-user",
		Password: "password-1",
	})
	s.Equal(http.StatusOK, loginResp.StatusCode)
	s.NotEmpty(loginResp.Body.AccessToken)

	playlistShares, err := s.env.Queries.GetAllPlaylistSharesForBackup(ctx)
	s.Require().NoError(err)
	s.Len(playlistShares, 1)
	s.Equal(playlist.ID, playlistShares[0].PlaylistID)
	s.Equal(sharedUser.ID, playlistShares[0].SharedWithUser)
	s.True(playlistShares[0].HasWritePermission)

	tracks, err := s.env.Queries.GetAllTracksForBackup(ctx)
	s.Require().NoError(err)
	s.Len(tracks, 1)
	s.Equal("backup-track", tracks[0].Name)
	s.True(tracks[0].FastPresetFname.Valid)
	s.Equal(fastTrackKey, tracks[0].FastPresetFname.String)

	gotAvatar, err := s.env.Storage.GetImage(ctx, userAvatarKey)
	s.Require().NoError(err)
	s.Equal(userAvatar, gotAvatar)

	originalReader, err := s.env.Storage.GetTrack(ctx, originalTrackKey)
	s.Require().NoError(err)
	defer func() { s.Require().NoError(originalReader.Close()) }()
	originalPayload, err := io.ReadAll(originalReader)
	s.Require().NoError(err)
	s.Equal(originalTrack, originalPayload)

	fastReader, err := s.env.Storage.GetTrack(ctx, fastTrackKey)
	s.Require().NoError(err)
	defer func() { s.Require().NoError(fastReader.Close()) }()
	fastPayload, err := io.ReadAll(fastReader)
	s.Require().NoError(err)
	s.Equal(fastTrack, fastPayload)

	s.assertNextSequenceValue(
		ctx,
		"public.user_id_seq",
		`SELECT (COALESCE(MAX(id), 0) + 1)::bigint FROM public."user"`,
	)
	s.assertNextSequenceValue(
		ctx,
		"public.artist_id_seq",
		`SELECT (COALESCE(MAX(id), 0) + 1)::bigint FROM public."artist"`,
	)
	s.assertNextSequenceValue(
		ctx,
		"public.album_id_seq",
		`SELECT (COALESCE(MAX(id), 0) + 1)::bigint FROM public."album"`,
	)
	s.assertNextSequenceValue(
		ctx,
		"public.playlist_id_seq",
		`SELECT (COALESCE(MAX(id), 0) + 1)::bigint FROM public."playlist"`,
	)
	s.assertNextSequenceValue(
		ctx,
		"public.track_id_seq",
		`SELECT (COALESCE(MAX(id), 0) + 1)::bigint FROM public."track"`,
	)
	s.assertNextSequenceValue(
		ctx,
		"public.transcoding_queue_id_seq",
		`SELECT (COALESCE(MAX(id), 0) + 1)::bigint FROM public."transcoding_queue"`,
	)
}

func (s *IntegrationTestSuite) TestBackupEndpointStartsAsyncOperationAndDownloadsArchive() {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	userResp := s.registerUser(api.UserAuth{
		Username: "async-backup-user",
		Password: "password",
	})
	s.Require().Equal(http.StatusCreated, userResp.StatusCode)

	artist, err := s.env.Queries.CreateArtist(ctx, "async-backup-artist")
	s.Require().NoError(err)

	track, err := s.env.Queries.CreateTrack(ctx, db.CreateTrackParams{
		Name:                "async-backup-track",
		ArtistID:            artist.ID,
		IsGloballyAvailable: false,
		UploadByUser:        pgtype.Int4{Int32: userResp.Body.UserId, Valid: true},
	})
	s.Require().NoError(err)

	originalTrackKey := fmt.Sprintf("track%d", track.ID)
	originalTrackPayload := []byte("async-original-track-payload")
	s.Require().NoError(s.env.Storage.PutTrack(
		ctx,
		originalTrackKey,
		bytes.NewReader(originalTrackPayload),
		storage.PutTrackOptions{Size: int64(len(originalTrackPayload))},
	))
	s.T().Cleanup(func() {
		_ = s.env.Storage.RemoveTrack(context.Background(), originalTrackKey)
	})

	statusCode, respBody := s.performJSONRequest(
		http.MethodPost,
		"/backup?include_transcoded_tracks=false",
		nil,
		userResp.Body.AccessToken,
	)
	s.Equal(http.StatusAccepted, statusCode)

	var startResp api.BackupStatusResponse
	s.Require().NoError(json.Unmarshal(respBody, &startResp))
	s.NotEmpty(startResp.BackupId)
	s.Equal(api.Pending, startResp.Status)
	s.False(startResp.IncludeImages)
	s.False(startResp.IncludeTranscodedTracks)

	s.T().Cleanup(func() {
		status, err := s.env.Queries.GetBackupStatus(context.Background(), startResp.BackupId)
		if err == nil && status.ArchivePath.Valid {
			_ = os.Remove(status.ArchivePath.String)
		}
	})

	var statusResp api.BackupStatusResponse
	s.Eventually(func() bool {
		statusCode, respBody := s.performJSONRequest(
			http.MethodGet,
			"/backup/"+startResp.BackupId,
			nil,
			userResp.Body.AccessToken,
		)
		s.Require().Equal(http.StatusOK, statusCode)
		s.Require().NoError(json.Unmarshal(respBody, &statusResp))
		if statusResp.Status == api.Error {
			if statusResp.Error != nil {
				s.T().Logf("backup failed: %s", *statusResp.Error)
			}
			return true
		}
		return statusResp.Status == api.Finished
	}, 10*time.Second, 100*time.Millisecond)
	s.Equal(api.Finished, statusResp.Status)
	s.Require().NotNil(statusResp.SizeBytes)
	s.Positive(*statusResp.SizeBytes)
	backupDownloadURL := s.server.URL + "/backup/" + startResp.BackupId + "/download"

	headReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodHead,
		backupDownloadURL,
		nil,
	)
	s.Require().NoError(err)
	headReq.Header.Set("Authorization", "Bearer "+userResp.Body.AccessToken)

	headResp, err := s.client.Do(headReq)
	s.Require().NoError(err)
	defer func() { s.Require().NoError(headResp.Body.Close()) }()

	s.Equal(http.StatusOK, headResp.StatusCode)
	s.Equal("application/zip", headResp.Header.Get("Content-Type"))
	s.Equal("bytes", headResp.Header.Get("Accept-Ranges"))
	s.Equal(fmt.Sprint(*statusResp.SizeBytes), headResp.Header.Get("Content-Length"))
	s.NotEmpty(headResp.Header.Get("ETag"))
	headBody, err := io.ReadAll(headResp.Body)
	s.Require().NoError(err)
	s.Empty(headBody)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		backupDownloadURL,
		nil,
	)
	s.Require().NoError(err)
	req.Header.Set("Authorization", "Bearer "+userResp.Body.AccessToken)

	downloadResp, err := s.client.Do(req)
	s.Require().NoError(err)
	defer func() { s.Require().NoError(downloadResp.Body.Close()) }()

	s.Equal(http.StatusOK, downloadResp.StatusCode)
	s.Equal("application/zip", downloadResp.Header.Get("Content-Type"))
	s.Equal("bytes", downloadResp.Header.Get("Accept-Ranges"))
	s.Equal(fmt.Sprint(*statusResp.SizeBytes), downloadResp.Header.Get("Content-Length"))
	archiveBytes, err := io.ReadAll(downloadResp.Body)
	s.Require().NoError(err)
	s.Positive(len(archiveBytes))

	archive, err := zip.NewReader(bytes.NewReader(archiveBytes), int64(len(archiveBytes)))
	s.Require().NoError(err)
	s.NotNil(findZipEntry(archive, backupManifestPathForTest))
	s.NotNil(findZipEntry(archive, backupDBPathForTest))
	s.NotNil(findZipEntry(archive, "tracks/original/"+originalTrackKey))

	rangeReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		backupDownloadURL,
		nil,
	)
	s.Require().NoError(err)
	rangeReq.Header.Set("Authorization", "Bearer "+userResp.Body.AccessToken)
	rangeReq.Header.Set("Range", "bytes=0-3")

	rangeResp, err := s.client.Do(rangeReq)
	s.Require().NoError(err)
	defer func() { s.Require().NoError(rangeResp.Body.Close()) }()

	s.Equal(http.StatusPartialContent, rangeResp.StatusCode)
	s.Equal("application/zip", rangeResp.Header.Get("Content-Type"))
	s.Equal("bytes", rangeResp.Header.Get("Accept-Ranges"))
	s.Equal("bytes 0-3/"+fmt.Sprint(len(archiveBytes)), rangeResp.Header.Get("Content-Range"))
	s.Equal("4", rangeResp.Header.Get("Content-Length"))
	rangeBody, err := io.ReadAll(rangeResp.Body)
	s.Require().NoError(err)
	s.Equal(archiveBytes[:4], rangeBody)

	invalidRangeReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		backupDownloadURL,
		nil,
	)
	s.Require().NoError(err)
	invalidRangeReq.Header.Set("Authorization", "Bearer "+userResp.Body.AccessToken)
	invalidRangeReq.Header.Set("Range", "bytes="+fmt.Sprint(len(archiveBytes))+"-")

	invalidRangeResp, err := s.client.Do(invalidRangeReq)
	s.Require().NoError(err)
	defer func() { s.Require().NoError(invalidRangeResp.Body.Close()) }()

	s.Equal(http.StatusRequestedRangeNotSatisfiable, invalidRangeResp.StatusCode)
	s.Equal("bytes */"+fmt.Sprint(len(archiveBytes)), invalidRangeResp.Header.Get("Content-Range"))
}

func (s *IntegrationTestSuite) TestRestoreQueuesTracksWhenTranscodedFilesMissingFromBackup() {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	backupService := service.NewBackupService(logger, s.env.Queries, s.env.Storage)

	user, err := s.env.Queries.CreateUser(ctx, db.CreateUserParams{
		Username:            "missing-transcoded-user",
		Password:            []byte("password-hash"),
		Salt:                []byte("password-salt"),
		PasswordMemory:      int32(utils.DefaultPasswordMemory),
		PasswordIterations:  int32(utils.DefaultPasswordIterations),
		PasswordParallelism: int32(utils.DefaultPasswordHashParams().Parallelism),
		PasswordKeyLength:   int32(utils.DefaultPasswordKeyLength),
	})
	s.Require().NoError(err)

	artist, err := s.env.Queries.CreateArtist(ctx, "missing-transcoded-artist")
	s.Require().NoError(err)

	album, err := s.env.Queries.CreateAlbum(ctx, db.CreateAlbumParams{
		Name:     "missing-transcoded-album",
		ArtistID: artist.ID,
	})
	s.Require().NoError(err)

	track, err := s.env.Queries.CreateTrack(ctx, db.CreateTrackParams{
		Name:                "missing-transcoded-track",
		ArtistID:            artist.ID,
		IsGloballyAvailable: false,
		UploadByUser:        pgtype.Int4{Int32: user.ID, Valid: true},
	})
	s.Require().NoError(err)

	s.Require().NoError(s.env.Queries.AddTrackToAlbum(ctx, db.AddTrackToAlbumParams{
		TrackID: track.ID,
		AlbumID: album.ID,
	}))

	originalTrackKey := fmt.Sprintf("track%d", track.ID)
	fastTrackKey := originalTrackKey + "_fast"
	_, err = s.env.Queries.AddPostTranscodingInfo(ctx, db.AddPostTranscodingInfoParams{
		ID:                  track.ID,
		DurationMs:          pgtype.Int4{Int32: 12345, Valid: true},
		FastPresetFname:     pgtype.Text{String: fastTrackKey, Valid: true},
		StandardPresetFname: pgtype.Text{Valid: false},
		HighPresetFname:     pgtype.Text{Valid: false},
		LosslessPresetFname: pgtype.Text{Valid: false},
	})
	s.Require().NoError(err)

	s.Require().NoError(s.env.Storage.PutTrack(
		ctx,
		originalTrackKey,
		bytes.NewReader([]byte("original-track-payload")),
		storage.PutTrackOptions{Size: int64(len("original-track-payload"))},
	))
	s.Require().NoError(s.env.Storage.PutTrack(
		ctx,
		fastTrackKey,
		bytes.NewReader([]byte("fast-track-payload")),
		storage.PutTrackOptions{Size: int64(len("fast-track-payload"))},
	))

	backupReader, _, err := backupService.MakeBackup(ctx, service.BackupSettings{
		IncludeTranscodedTracks: true,
	})
	s.Require().NoError(err)
	archiveBytes, err := io.ReadAll(backupReader)
	s.Require().NoError(err)
	s.Require().NoError(backupReader.Close())

	archiveBytes = s.zipWithoutEntriesWithPrefix(archiveBytes, "tracks/transcoded/")

	restoreID, err := backupService.StartRestoreOperation(ctx, bytes.NewReader(archiveBytes))
	s.Require().NoError(err)

	var status api.RestoreStatusResponse
	s.Eventually(func() bool {
		var statusErr error
		status, statusErr = backupService.CheckRestoreOperation(context.Background(), restoreID)
		s.Require().NoError(statusErr)
		if status.Status == api.Error {
			s.T().Logf("restore failed: %s", *status.Error)
			return true
		}
		return status.Status == api.Finished
	}, 10*time.Second, 100*time.Millisecond)
	s.Equal(api.Finished, status.Status)

	tracks, err := s.env.Queries.GetAllTracksForBackup(ctx)
	s.Require().NoError(err)
	s.Len(tracks, 1)
	s.False(tracks[0].FastPresetFname.Valid)
	s.False(tracks[0].StandardPresetFname.Valid)
	s.False(tracks[0].HighPresetFname.Valid)
	s.False(tracks[0].LosslessPresetFname.Valid)

	queueRows, err := s.env.Queries.GetAllTranscodingQueueForBackup(ctx)
	s.Require().NoError(err)
	s.Len(queueRows, 1)
	s.Equal(track.ID, queueRows[0].TrackID)
	s.Equal(originalTrackKey, queueRows[0].TrackOriginalFileName)
	s.False(queueRows[0].WasFailed)
}

func (s *IntegrationTestSuite) zipWithoutEntriesWithPrefix(payload []byte, prefix string) []byte {
	reader, err := zip.NewReader(bytes.NewReader(payload), int64(len(payload)))
	s.Require().NoError(err)

	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	for _, file := range reader.File {
		if strings.HasPrefix(file.Name, prefix) {
			continue
		}

		header := file.FileHeader
		entryWriter, err := writer.CreateHeader(&header)
		s.Require().NoError(err)

		entryReader, err := file.Open()
		s.Require().NoError(err)
		_, copyErr := io.Copy(entryWriter, entryReader)
		closeErr := entryReader.Close()
		s.Require().NoError(copyErr)
		s.Require().NoError(closeErr)
	}
	s.Require().NoError(writer.Close())

	return output.Bytes()
}

func (s *IntegrationTestSuite) assertNextSequenceValue(
	ctx context.Context, sequenceName, expectedQuery string,
) {
	s.T().Helper()

	var expected int64
	err := s.env.DB.QueryRow(ctx, expectedQuery).Scan(&expected)
	s.Require().NoError(err)

	var actual int64
	err = s.env.DB.QueryRow(ctx, `SELECT nextval($1::regclass)`, sequenceName).Scan(&actual)
	s.Require().NoError(err)
	s.Equal(expected, actual)
}

const (
	backupManifestPathForTest = "manifest.json"
	backupDBPathForTest       = "db/full.json"
)

func findZipEntry(archive *zip.Reader, name string) *zip.File {
	for _, file := range archive.File {
		if file.Name == name {
			return file
		}
	}
	return nil
}
