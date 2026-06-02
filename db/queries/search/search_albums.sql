-- name: SearchAlbums :many
SELECT 
    al.id as album_id,
    al.name as name,
    al.artist_id as artist_id,
    ar.name as artist_name,
    al.release_year as release_year,
    similarity(al.name, sqlc.arg(query)) as score
FROM "album" al
INNER JOIN "artist" ar ON al.artist_id = ar.id
WHERE al.name ILIKE '%' || sqlc.arg(query)::text || '%'
    OR al.name % sqlc.arg(query)::text
ORDER BY
    lower(al.name) = lower(sqlc.arg(query)::text) DESC,
    lower(al.name) ILIKE lower(sqlc.arg(query)::text) || '%' DESC,
    score DESC,
    al.name
LIMIT sqlc.arg(bound)::integer;
