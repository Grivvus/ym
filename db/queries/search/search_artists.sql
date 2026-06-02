-- name: SearchArtists :many
SELECT 
    ar.id as artist_id,
    ar.name as artist_name,
    similarity(ar.name, sqlc.arg(query)) as score
FROM "artist" ar
WHERE ar.name ILIKE '%' || sqlc.arg(query)::text || '%'
    OR ar.name % sqlc.arg(query)::text
ORDER BY
    lower(ar.name) = lower(sqlc.arg(query)::text) DESC,
    lower(ar.name) ILIKE lower(sqlc.arg(query)::text) || '%' DESC,
    score DESC,
    ar.name
LIMIT sqlc.arg(bound)::integer;
