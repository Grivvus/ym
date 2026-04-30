-- name: SharePlaylistWith :exec
INSERT INTO "playlist_share_info"
    (playlist_id, shared_with_user, has_write_permission)
    VALUES ($1, $2, $3);

-- name: GetSharedUsers :many
SELECT shared_with_user FROM "playlist_share_info"
WHERE playlist_id = $1;

-- name: RevokePlaylistAccess :one
DELETE from "playlist_share_info"
WHERE playlist_id = $1 AND shared_with_user = $2
RETURNING *;
