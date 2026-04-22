-- name: GetAlbum :one
SELECT id , name, release_year, release_full_date
    FROM "album" 
WHERE "album".id = $1;
