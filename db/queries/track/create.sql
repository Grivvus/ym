-- name: CreateTrack :one
INSERT INTO "track" (name, artist_id, duration)
    VALUES ($1, $2, $3)
RETURNING *;
