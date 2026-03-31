-- name: FindUsersPlaylistByName :one
SELECT p.id FROM playlist as p
    WHERE p.owner_id = $1 AND p.name = $2;
