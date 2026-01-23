-- name: CreateAlbum :one
INSERT INTO "album" (name, artist_id)
    VALUES ($1, $2)
    RETURNING *;
