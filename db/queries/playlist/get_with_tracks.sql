-- name: GetPlaylistWithTracks :many
SELECT "playlist".id, "playlist".name, "track_playlist".track_id
    from "playlist" inner join "track_playlist"
    ON "playlist".id = "track_playlist".playlist_id
    WHERE "playlist".id = $1;
