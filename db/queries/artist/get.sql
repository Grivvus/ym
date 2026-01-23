-- name: GetArtist :many
SELECT "artist".id, "artist".name, "album".id
    from "artist" INNER JOIN "album"
    ON "artist".id = "album".artist_id
where "artist".id = $1;
