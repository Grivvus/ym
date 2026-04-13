-- name: RestoreTrack :exec
INSERT INTO public."track" (
    id, name, duration_ms, url,
    fast_preset_fname, standard_preset_fname,
    high_preset_fname, lossless_preset_fname,
    is_globally_available, artist_id, upload_by_user
) VALUES (
    $1, $2, $3, $4,
    $5, $6, $7, $8,
    $9, $10, $11
);
