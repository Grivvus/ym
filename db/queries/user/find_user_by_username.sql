-- name: GetUserByUsername :one
SELECT id, username, email, password, salt, is_superuser, refresh_version
FROM "user"
WHERE username = $1;
