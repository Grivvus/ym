-- name: CreatePlaylist :one
INSERT into "playlist" (name, owner_id, is_public)
VALUES ($1, $2, $3)
RETURNING *;
