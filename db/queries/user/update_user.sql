-- name: UpdateUser :one
UPDATE "user" SET 
    username = $2,
    email = $3,
    updated_at = now()
WHERE id = $1
RETURNING *;
