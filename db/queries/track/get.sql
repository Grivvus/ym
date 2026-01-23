-- name: GetTrack :one
SELECT * FROM "track"
    WHERE id = $1;
