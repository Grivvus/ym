-- name: GetAlbum :many
SELECT "album".id, "album".name, "track_album".track_id
    from "album" inner join "track_album"
    ON "album".id = "track_album".album_id;
