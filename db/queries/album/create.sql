-- name: CreateAlbum :one
INSERT INTO "album" (name, artist_id, release_year, release_full_date)
    VALUES ($1, $2, $3, $4)
    RETURNING *;
