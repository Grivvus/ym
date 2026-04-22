-- name: GetArtistWithAlbums :many
SELECT "artist".id as artist_id, "artist".name as artist_name,
        "album".id as album_id
    FROM "artist" INNER JOIN "album"
    ON "artist".id = "album".artist_id
WHERE "artist".id = $1;
