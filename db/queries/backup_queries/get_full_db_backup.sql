-- name: GetAllUsersForBackup :many
SELECT id, username, is_superuser, email, password, salt, refresh_version, created_at, updated_at
FROM public."user"
ORDER BY id;

-- name: GetAllArtistsForBackup :many
SELECT id, name, url
FROM public."artist"
ORDER BY id;

-- name: GetAllAlbumsForBackup :many
SELECT id, name, artist_id
FROM public."album"
ORDER BY id;

-- name: GetAllPlaylistsForBackup :many
SELECT id, name, is_public, owner_id
FROM public."playlist"
ORDER BY id;

-- name: GetAllTracksForBackup :many
SELECT id, name, duration_ms, url,
    fast_preset_fname, standard_preset_fname,
    high_preset_fname, lossless_preset_fname,
    is_globally_available, artist_id, upload_by_user
FROM public."track"
ORDER BY id;

-- name: GetAllTrackAlbumsForBackup :many
SELECT track_id, album_id
FROM public."track_album"
ORDER BY track_id, album_id;

-- name: GetAllTrackPlaylistsForBackup :many
SELECT track_id, playlist_id
FROM public."track_playlist"
ORDER BY track_id, playlist_id;

-- name: GetAllPlaylistSharesForBackup :many
SELECT playlist_id, shared_with_user, has_write_permission
FROM public."playlist_share_info"
ORDER BY playlist_id, shared_with_user;

-- name: GetAllTranscodingQueueForBackup :many
SELECT id, added_at, track_original_file_name, track_id, was_failed, error_msg
FROM public."transcoding_queue"
ORDER BY id;
