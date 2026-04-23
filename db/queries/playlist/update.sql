-- name: UpdatePlaylist :one
UPDATE public."playlist" SET
    name = $2
WHERE id = $1
RETURNING *;
