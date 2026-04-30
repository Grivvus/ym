-- name: SharePlaylistWith :exec
INSERT INTO "playlist_share_info"
    (playlist_id, shared_with_user, has_write_permission)
    VALUES ($1, $2, $3);
