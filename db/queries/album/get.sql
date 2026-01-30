-- name: GetAlbum :one
SELECT "album".id as id, "album".name as name
    from "album" 
WHERE "album".id = $1;
