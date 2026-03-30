-- name: GetTrack :one
SELECT t.id, t.name, t.artist_id, ta.album_id,
t.fast_preset_fname, t.standard_preset_fname,
t.high_preset_fname, t.lossless_preset_fname
    FROM "track" AS t
    INNER JOIN "track_album" AS ta
        ON t.id = ta.track_id
    WHERE "track".id = $1
    LIMIT 1;
