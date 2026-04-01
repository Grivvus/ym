-- name: GetUserTracks :many
SELECT t.id, t.name, t.artist_id,
t.fast_preset_fname, t.standard_preset_fname,
t.high_preset_fname, t.lossless_preset_fname
    FROM "track" AS t
    WHERE t.is_globally_available OR t.upload_by_user = $1;
