-- name: GetUserByUsername :one
SELECT * FROM "user"
WHERE username = $1;
