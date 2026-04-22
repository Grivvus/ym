-- name: GetAlbumWithTracks :many
SELECT "album".id, "album".name, "album".release_year,
        "album".release_full_date, "track_album".track_id
    FROM "album" INNER JOIN "track_album"
    ON "album".id = "track_album".album_id
WHERE "album".id = $1;
