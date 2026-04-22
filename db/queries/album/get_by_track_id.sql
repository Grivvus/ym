-- name: GetAlbumByTrackID :one
SELECT album_id FROM track_album
    WHERE track_id = $1
    LIMIT 1;
