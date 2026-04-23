-- name: GetPlaylist :one
SELECT id, name, owner_id
    FROM "playlist"
WHERE "playlist".id = $1;
