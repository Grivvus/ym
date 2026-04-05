-- name: GetUserByID :one
SELECT id, username, email, password, salt 
    FROM "user"
    WHERE id = $1;
