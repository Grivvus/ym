-- name: AddTrackToPlaylist :exec
INSERT INTO "track_playlist" (track_id, playlist_id)
    VALUES ($1, $2);
