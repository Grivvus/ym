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
SELECT DISTINCT t.id, t.name, t.artist_id, duration_ms,
t.fast_preset_fname, t.standard_preset_fname,
t.high_preset_fname, t.lossless_preset_fname
    FROM "track" AS t
    WHERE t.is_globally_available
        OR t.upload_by_user = sqlc.arg(user_id)::integer
        OR EXISTS (
            SELECT 1
            FROM "track_playlist" AS tp
            INNER JOIN "playlist" AS p
                ON p.id = tp.playlist_id
            LEFT JOIN "playlist_share_info" AS psi
                ON psi.playlist_id = p.id
                AND psi.shared_with_user = sqlc.arg(user_id)::integer
            WHERE tp.track_id = t.id
                AND (
                    p.owner_id = sqlc.arg(user_id)::integer
                    OR p.is_public IS TRUE
                    OR psi.shared_with_user IS NOT NULL
                )
        );

-- name: CanUserAccessTrack :one
SELECT EXISTS (
    SELECT 1
    FROM "track" AS t
    WHERE t.id = sqlc.arg(track_id)::integer
        AND (
            t.is_globally_available
            OR t.upload_by_user = sqlc.arg(user_id)::integer
            OR EXISTS (
                SELECT 1
                FROM "track_playlist" AS tp
                INNER JOIN "playlist" AS p
                    ON p.id = tp.playlist_id
                LEFT JOIN "playlist_share_info" AS psi
                    ON psi.playlist_id = p.id
                    AND psi.shared_with_user = sqlc.arg(user_id)::integer
                WHERE tp.track_id = t.id
                    AND (
                        p.owner_id = sqlc.arg(user_id)::integer
                        OR p.is_public IS TRUE
                        OR psi.shared_with_user IS NOT NULL
                    )
            )
        )
)::boolean;

-- name: GetAllTracks :many
SELECT t.id, t.name, t.artist_id, duration_ms,
t.fast_preset_fname, t.standard_preset_fname,
t.high_preset_fname, t.lossless_preset_fname
    FROM "track" AS t;
