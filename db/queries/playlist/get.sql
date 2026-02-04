-- name: GetPlaylist :one
SELECT "playlist".id as id, "playlist".name as name
    FROM "playlist"
WHERE "playlist".id = $1;
