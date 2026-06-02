-- name: SearchTracks :many
SELECT
    t.id,
    t.name,
    t.artist_id,
    ar.name AS artist_name,
    al.id AS album_id,
    al.name AS album_name,
    t.duration_ms,
    t.is_globally_available,
    GREATEST(
        similarity(t.name, sqlc.arg(query)),
        similarity(ar.name, sqlc.arg(query)),
        similarity(al.name, sqlc.arg(query))
    )::real AS score
FROM track t
JOIN artist ar ON ar.id = t.artist_id
JOIN track_album ta ON ta.track_id = t.id
JOIN album al ON al.id = ta.album_id
WHERE
    (
        t.name ILIKE '%' || sqlc.arg(query)::text || '%'
        OR ar.name ILIKE '%' || sqlc.arg(query)::text || '%'
        OR al.name ILIKE '%' || sqlc.arg(query)::text || '%'
        OR t.name % sqlc.arg(query)::text
        OR ar.name % sqlc.arg(query)::text
        OR al.name % sqlc.arg(query)::text
    )
    AND ( -- track is available to user
        t.is_globally_available
        OR t.upload_by_user = sqlc.arg(user_id)::integer
        OR EXISTS (
            SELECT 1
            FROM track_playlist tp
            INNER JOIN playlist p ON p.id = tp.playlist_id
            LEFT JOIN playlist_share_info psi
                ON psi.playlist_id = p.id
                AND psi.shared_with_user = sqlc.arg(user_id)::integer
            WHERE tp.track_id = t.id
                AND (
                    p.owner_id = sqlc.arg(user_id)::integer
                    OR p.is_public IS TRUE
                    OR psi.shared_with_user IS NOT NULL
                ))
    )
ORDER BY
    lower(t.name) = lower(sqlc.arg(query)::text) DESC,
    lower(t.name) ILIKE lower(sqlc.arg(query)::text) || '%' DESC,
    score DESC,
    t.name
LIMIT sqlc.arg(bound)::integer;
