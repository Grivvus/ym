-- name: UpdateArtist :one
UPDATE "artist" SET
    name = sqlc.arg(name)::text
WHERE id = sqlc.arg(id)::integer
RETURNING id, name;
