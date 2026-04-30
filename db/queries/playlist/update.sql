-- name: UpdatePlaylist :one
UPDATE "playlist" SET
    name = $2
WHERE id = $1
RETURNING *;
