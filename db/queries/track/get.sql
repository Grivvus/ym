-- name: GetTrack :one
SELECT t.id, t.name, t.artist_id, a.name AS artist_name, ta.album_id, t.duration_ms,
t.fast_preset_fname, t.standard_preset_fname,
t.high_preset_fname, t.lossless_preset_fname,
t.is_globally_available, t.upload_by_user
    FROM "track" AS t
    INNER JOIN "track_album" AS ta
        ON t.id = ta.track_id
    INNER JOIN "artist" AS a
        ON t.artist_id = a.id
    WHERE t.id = $1
    LIMIT 1;

-- name: GetUserTracks :many
SELECT t.id, t.name, t.artist_id, duration_ms,
t.fast_preset_fname, t.standard_preset_fname,
t.high_preset_fname, t.lossless_preset_fname
    FROM "track" AS t
    WHERE t.is_globally_available OR t.upload_by_user = $1;
