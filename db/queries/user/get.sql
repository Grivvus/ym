-- name: GetAllUsernames :many
SELECT id, username FROM "user";

-- name: GetAllUsernamesExcept :many
SELECT id, username FROM "user"
WHERE id <> $1;

-- name: GetUsername :one
SELECT id, username FROM "user"
WHERE id = $1;
