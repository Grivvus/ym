-- name: GetUserByID :one
SELECT id, username, email, password, salt, is_superuser
    FROM "user"
    WHERE id = $1;
