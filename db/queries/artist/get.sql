-- name: GetArtist :one
SELECT "artist".id, "artist".name
    from "artist" 
where "artist".id = $1;
