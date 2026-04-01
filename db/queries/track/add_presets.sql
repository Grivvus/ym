-- name: AddPostTranscodingInfo :one
UPDATE "track" SET
    duration_ms = $2,
    fast_preset_fname = $3,
    standard_preset_fname = $4,
    high_preset_fname = $5,
    lossless_preset_fname = $6
WHERE id = $1
RETURNING *;
