-- name: CreatePlaylist :one
INSERT into "playlist" (name, owner_id)
VALUES ($1, $2)
RETURNING *;
