-- name: GetPlaylist :one
SELECT id, name, owner_id
    FROM "playlist"
WHERE "playlist".id = $1;

-- name: GetUsersPlaylistByName :one
SELECT p.id FROM playlist as p
    WHERE p.owner_id = $1 AND p.name = $2;

-- name: GetUserPlaylists :many
SELECT "playlist".id, "playlist".name
    FROM "playlist"
    WHERE "playlist".owner_id = $1 OR "playlist".is_public IS TRUE;

-- name: GetPlaylistWithTracks :many
SELECT "playlist".id, "playlist".name, "track_playlist".track_id
    from "playlist" inner join "track_playlist"
    ON "playlist".id = "track_playlist".playlist_id
    WHERE "playlist".id = $1;
