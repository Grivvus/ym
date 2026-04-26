-- name: GetUserByEmail :one
SELECT id, username, email, password, salt, is_superuser, refresh_version
FROM "user"
WHERE LOWER(email) = LOWER($1);
