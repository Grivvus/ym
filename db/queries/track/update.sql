-- name: UpdateTrack :one
UPDATE "track" SET
    name = sqlc.arg(name)::text,
    artist_id = sqlc.arg(artist_id)::integer,
    is_globally_available = sqlc.arg(is_globally_available)::boolean
WHERE id = sqlc.arg(id)::integer
RETURNING *;
