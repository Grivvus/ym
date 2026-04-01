-- name: CreateTrack :one
INSERT INTO "track" (name, artist_id, upload_by_user, is_globally_available)
    VALUES ($1, $2, $3, $4)
RETURNING *;
