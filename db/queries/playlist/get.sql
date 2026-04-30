-- name: GetPlaylist :one
SELECT id, name, owner_id
    FROM "playlist"
WHERE "playlist".id = $1;

-- name: GetUsersPlaylistByName :one
SELECT p.id FROM "playlist" as p
    WHERE p.owner_id = $1 AND p.name = $2;

-- name: GetUserOwnedPlaylists :many
SELECT "playlist".id, "playlist".name
    FROM "playlist"
    WHERE "playlist".owner_id = $1;

-- name: GetPublicPlaylists :many
SELECT p.id, p.name, p.owner_id
    FROM "playlist" p
    WHERE p.is_public IS TRUE AND 
        p.owner_id <> $1;

-- name: GetSharedPlaylists :many
SELECT p.id, p.name, p.owner_id, ps.has_write_permission
    FROM "playlist" p INNER JOIN "playlist_share_info" ps
        ON p.id = ps.playlist_id
    WHERE ps.shared_with_user = $1;

-- name: GetPlaylistWithTracks :many
SELECT "playlist".id, "playlist".name, "track_playlist".track_id
    from "playlist" inner join "track_playlist"
    ON "playlist".id = "track_playlist".playlist_id
    WHERE "playlist".id = $1;
