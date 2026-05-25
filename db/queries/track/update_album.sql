-- name: ReplaceTrackAlbum :exec
WITH deleted AS (
    DELETE FROM "track_album"
    WHERE track_id = sqlc.arg(track_id)::integer
)
INSERT INTO "track_album" (track_id, album_id)
VALUES (sqlc.arg(track_id)::integer, sqlc.arg(album_id)::integer);
