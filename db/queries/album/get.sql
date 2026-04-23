-- name: GetAlbum :one
SELECT id , name, release_year, release_full_date
    FROM "album" 
WHERE "album".id = $1;

-- name: GetAlbumByTrackID :one
SELECT album_id FROM track_album
    WHERE track_id = $1
    LIMIT 1;

-- name: GetAlbumWithTracks :many
SELECT "album".id, "album".name, "album".release_year,
        "album".release_full_date, "track_album".track_id
    FROM "album" INNER JOIN "track_album"
    ON "album".id = "track_album".album_id
WHERE "album".id = $1;
