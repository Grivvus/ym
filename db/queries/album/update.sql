-- name: UpdateAlbum :one
UPDATE "album" SET
    name = sqlc.arg(name)::text,
    artist_id = sqlc.arg(artist_id)::integer,
    release_year = sqlc.narg(release_year)::integer,
    release_full_date = sqlc.narg(release_full_date)::date
WHERE id = sqlc.arg(id)::integer
RETURNING *;
