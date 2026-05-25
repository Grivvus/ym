-- name: UpdatePlaylist :one
UPDATE "playlist" SET
    name = sqlc.arg(name)::text,
    is_public = sqlc.arg(is_public)::boolean
WHERE id = sqlc.arg(id)::integer
RETURNING *;
