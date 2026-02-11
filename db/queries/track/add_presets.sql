-- name: AddTrackPresets :one
UPDATE "track" SET
    fast_preset_fname = $2,
    standard_preset_fname = $3,
    high_preset_fname = $4,
    lossless_preset_fname = $5
WHERE id = $1
RETURNING *;
