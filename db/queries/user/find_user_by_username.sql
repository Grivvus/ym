-- name: GetUserByUsername :one
SELECT id, username, email, password, salt, is_superuser
FROM "user"
WHERE username = $1;
