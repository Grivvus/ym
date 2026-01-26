-- name: GetArtistWithAlbums :many
SELECT "artist".id as artist_id, "artist".name as artist_name, "album".id as album_id
    from "artist" INNER JOIN "album"
    ON "artist".id = "album".artist_id
where "artist".id = $1;
