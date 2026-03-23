-- name: GetUserPlaylists :many
SELECT "playlist".id, "playlist".name
    FROM "playlist"
    WHERE "playlist".owner_id = $1;
