-- name: GetUserByUsername :one
SELECT id, username, email, password, salt 
FROM "user"
WHERE username = $1;
