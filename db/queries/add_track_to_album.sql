-- name: AddTrackToAlbum :exec
INSERT INTO "track_album" (track_id, album_id)
    VALUES ($1, $2);
