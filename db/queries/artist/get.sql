-- name: GetArtist :one
SELECT "artist".id, "artist".name
    from "artist" 
where "artist".id = $1;

-- name: GetAllArtists :many
SELECT "id", "name" FROM artist;

-- name: GetArtistsWithFilter :many
SELECT "id", "name" FROM artist
                    -- starts with
    WHERE "name" LIKE $1 || '%'
LIMIT $2;

-- name: GetArtistWithAlbums :many
SELECT "artist".id as artist_id, "artist".name as artist_name,
        "album".id as album_id
    FROM "artist" INNER JOIN "album"
    ON "artist".id = "album".artist_id
WHERE "artist".id = $1;
